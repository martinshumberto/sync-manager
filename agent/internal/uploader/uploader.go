package uploader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/martinshumberto/sync-manager/agent/internal/config"
	"github.com/martinshumberto/sync-manager/agent/internal/storage"
	commonconfig "github.com/martinshumberto/sync-manager/common/config"
	"github.com/rs/zerolog/log"
)

// UploadTask represents a file to be uploaded
type UploadTask struct {
	FilePath    string            // Full path to the file on disk
	Key         string            // Remote key for storage
	FolderID    string            // ID of the synced folder
	Priority    int               // Priority level (higher means more important)
	Metadata    map[string]string // Additional metadata for the file
	RetryCount  int               // Number of times this task has been retried
	LastAttempt time.Time         // When the task was last attempted
}

// UploadResult represents the result of an upload operation
type UploadResult struct {
	Task      UploadTask // The original task
	Success   bool       // Whether the upload was successful
	Error     error      // Error if any occurred
	VersionID string     // Version ID from the storage provider
	Hash      string     // SHA256 hash of the file
	Size      int64      // Size of the file in bytes
}

// Uploader handles file uploads with concurrency control and throttling
type Uploader struct {
	store          storage.Storage
	taskQueue      chan UploadTask
	resultChan     chan UploadResult
	maxConcurrency int
	throttleBytes  int64 // bytes per second, 0 for no throttling
	workers        sync.WaitGroup
	mutex          sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
	running        bool
}

// NewUploader creates a new uploader
func NewUploader(store storage.Storage, cfg interface{}) *Uploader {
	ctx, cancel := context.WithCancel(context.Background())

	// Use default values if not specified
	maxConcurrency := 4
	var throttleBytes int64 = 0

	// Se a configuração for do tipo commonconfig.Config
	if commCfg, ok := cfg.(*commonconfig.Config); ok {
		maxConcurrency = commCfg.MaxConcurrency
		throttleBytes = commCfg.ThrottleBytes
	} else if _, ok := cfg.(*config.Config); ok {
		// Para compatibilidade com o config interno
		// Aqui podemos adicionar lógica específica se necessário
	}

	return &Uploader{
		store:          store,
		taskQueue:      make(chan UploadTask, 1000), // Buffer up to 1000 tasks
		resultChan:     make(chan UploadResult, 100),
		maxConcurrency: maxConcurrency,
		throttleBytes:  throttleBytes,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start starts the uploader workers
func (u *Uploader) Start() {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if u.running {
		return
	}

	u.running = true
	log.Info().Int("workers", u.maxConcurrency).Msg("Starting uploader")

	// Start worker goroutines
	for i := 0; i < u.maxConcurrency; i++ {
		u.workers.Add(1)
		go u.worker(i)
	}
}

// Stop stops the uploader
func (u *Uploader) Stop() {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if !u.running {
		return
	}

	log.Info().Msg("Stopping uploader")
	u.cancel()
	close(u.taskQueue)
	u.workers.Wait()
	close(u.resultChan)
	u.running = false
}

// QueueUpload adds a file to the upload queue
func (u *Uploader) QueueUpload(task UploadTask) error {
	select {
	case u.taskQueue <- task:
		log.Debug().
			Str("path", task.FilePath).
			Str("key", task.Key).
			Msg("Queued file for upload")
		return nil
	default:
		return fmt.Errorf("upload queue is full")
	}
}

// Results returns the channel where upload results are sent
func (u *Uploader) Results() <-chan UploadResult {
	return u.resultChan
}

// QueueFile enfileira um arquivo para upload com base em seu caminho e pasta raiz
func (u *Uploader) QueueFile(filePath, folderPath string) error {
	// Verificar se o uploader está rodando
	if !u.running {
		return fmt.Errorf("uploader is not running")
	}

	// Obter o caminho relativo do arquivo em relação à pasta
	relPath, err := filepath.Rel(folderPath, filePath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Construir a chave de armazenamento
	// Usamos o folderPath como base para diferenciar diferentes pastas sincronizadas
	storageKey := filepath.ToSlash(relPath)

	// Criar a tarefa de upload
	task := UploadTask{
		FilePath:   filePath,
		Key:        storageKey,
		Priority:   1, // Prioridade padrão
		Metadata:   make(map[string]string),
		RetryCount: 0,
	}

	// Adicionar metadados básicos
	task.Metadata["source_folder"] = folderPath
	task.Metadata["upload_time"] = time.Now().Format(time.RFC3339)

	// Enfileirar a tarefa
	return u.QueueUpload(task)
}

// worker processes upload tasks
func (u *Uploader) worker(id int) {
	defer u.workers.Done()

	log.Debug().Int("worker_id", id).Msg("Upload worker started")

	for task := range u.taskQueue {
		select {
		case <-u.ctx.Done():
			return
		default:
			result := u.processUpload(task)

			// Send result
			select {
			case u.resultChan <- result:
				// Successfully sent result
			case <-u.ctx.Done():
				return
			}

			// If the upload failed, retry it with exponential backoff
			if !result.Success && task.RetryCount < 3 {
				backoff := time.Duration(1<<task.RetryCount) * time.Second
				task.RetryCount++
				task.LastAttempt = time.Now()

				log.Info().
					Str("path", task.FilePath).
					Int("retry", task.RetryCount).
					Dur("backoff", backoff).
					Msg("Scheduling retry")

				// Wait for backoff period, but respect context cancellation
				select {
				case <-time.After(backoff):
					// Try again
					select {
					case u.taskQueue <- task:
						// Re-queued
					case <-u.ctx.Done():
						return
					}
				case <-u.ctx.Done():
					return
				}
			}
		}
	}

	log.Debug().Int("worker_id", id).Msg("Upload worker stopped")
}

// processUpload handles a single upload task
func (u *Uploader) processUpload(task UploadTask) UploadResult {
	result := UploadResult{
		Task:    task,
		Success: false,
	}

	// Check if file exists
	file, err := os.Open(task.FilePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to open file: %w", err)
		return result
	}
	defer file.Close()

	// Get file stats
	fileInfo, err := file.Stat()
	if err != nil {
		result.Error = fmt.Errorf("failed to get file info: %w", err)
		return result
	}

	// Skip directories
	if fileInfo.IsDir() {
		result.Error = fmt.Errorf("cannot upload directory")
		return result
	}

	// Calculate hash
	hash, err := calculateSHA256(file)
	if err != nil {
		result.Error = fmt.Errorf("failed to calculate hash: %w", err)
		return result
	}
	result.Hash = hash

	// Reset file position
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		result.Error = fmt.Errorf("failed to reset file position: %w", err)
		return result
	}

	// Update metadata with file info
	fileSize := fileInfo.Size()
	result.Size = fileSize

	if task.Metadata == nil {
		task.Metadata = make(map[string]string)
	}
	task.Metadata["content_type"] = detectContentType(task.FilePath)
	task.Metadata["hash_sha256"] = hash
	task.Metadata["size"] = fmt.Sprintf("%d", fileSize)
	task.Metadata["modified_time"] = fileInfo.ModTime().UTC().Format(time.RFC3339)

	// Create reader with throttling if needed
	var reader io.Reader = file
	if u.throttleBytes > 0 {
		reader = newThrottledReader(file, u.throttleBytes)
	}

	// Upload the file
	log.Info().
		Str("path", task.FilePath).
		Str("key", task.Key).
		Int64("size", fileSize).
		Msg("Uploading file")

	versionID, err := u.store.UploadFile(u.ctx, task.Key, reader, task.Metadata)
	if err != nil {
		result.Error = fmt.Errorf("failed to upload file: %w", err)
		return result
	}

	result.VersionID = versionID
	result.Success = true

	log.Info().
		Str("path", task.FilePath).
		Str("key", task.Key).
		Str("version", versionID).
		Int64("size", fileSize).
		Msg("Upload successful")

	return result
}

// calculateSHA256 calculates the SHA256 hash of a file
func calculateSHA256(file *os.File) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// detectContentType tries to detect the content type of a file
func detectContentType(filePath string) string {
	// Use extension-based detection for simplicity
	ext := filepath.Ext(filePath)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".zip":
		return "application/zip"
	case ".doc", ".docx":
		return "application/msword"
	case ".xls", ".xlsx":
		return "application/vnd.ms-excel"
	case ".ppt", ".pptx":
		return "application/vnd.ms-powerpoint"
	default:
		return "application/octet-stream"
	}
}

// ThrottledReader wraps an io.Reader with rate limiting
type throttledReader struct {
	reader        io.Reader
	bytesPerSec   int64
	bytesThisSec  int64
	lastTimestamp time.Time
	mu            sync.Mutex
}

func newThrottledReader(reader io.Reader, bytesPerSec int64) *throttledReader {
	return &throttledReader{
		reader:        reader,
		bytesPerSec:   bytesPerSec,
		lastTimestamp: time.Now(),
	}
}

func (t *throttledReader) Read(p []byte) (n int, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// If we've read too much this second, sleep
	now := time.Now()
	elapsed := now.Sub(t.lastTimestamp)

	// Reset counter if more than a second has passed
	if elapsed >= time.Second {
		t.bytesThisSec = 0
		t.lastTimestamp = now
	}

	// If we've read our quota for this interval, sleep
	if t.bytesThisSec >= t.bytesPerSec {
		timeToSleep := time.Second - elapsed
		if timeToSleep > 0 {
			time.Sleep(timeToSleep)
			t.bytesThisSec = 0
			t.lastTimestamp = time.Now()
		}
	}

	// Calculate how many bytes we can read without exceeding the limit
	maxBytes := t.bytesPerSec - t.bytesThisSec
	if maxBytes <= 0 {
		maxBytes = t.bytesPerSec
	}

	// Don't read more than maxBytes or the buffer size
	toRead := len(p)
	if int64(toRead) > maxBytes {
		toRead = int(maxBytes)
	}

	// Read from the underlying reader
	n, err = t.reader.Read(p[:toRead])

	// Update bytes read this second
	t.bytesThisSec += int64(n)

	return n, err
}
