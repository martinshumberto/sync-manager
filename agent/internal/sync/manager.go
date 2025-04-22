package sync

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/martinshumberto/sync-manager/agent/internal/config"
	"github.com/martinshumberto/sync-manager/agent/internal/storage"
	"github.com/martinshumberto/sync-manager/agent/internal/uploader"
	"github.com/martinshumberto/sync-manager/agent/internal/watcher"
	"github.com/rs/zerolog/log"
)

// EventType is a temporary type to work around the compilation error
type EventType = watcher.EventType

// Event is a temporary type to work around the compilation error
type Event struct {
	Type      EventType
	Path      string
	Timestamp time.Time
}

// SyncState represents the state of a sync operation
type SyncState string

const (
	// SyncStateIdle means the sync manager is not synchronizing
	SyncStateIdle SyncState = "idle"
	// SyncStateScanning means the sync manager is scanning for changes
	SyncStateScanning SyncState = "scanning"
	// SyncStateSyncing means the sync manager is syncing changes
	SyncStateSyncing SyncState = "syncing"
	// SyncStateError means the sync manager encountered an error
	SyncStateError SyncState = "error"
	// SyncStatePaused indicates that synchronization is paused
	SyncStatePaused SyncState = "paused"
)

// SyncStats tracks statistics about the sync process
type SyncStats struct {
	TotalFiles      int64
	FilesUploaded   int64
	FilesDownloaded int64
	BytesUploaded   int64
	BytesDownloaded int64
	LastSyncTime    time.Time
	Errors          int
	StartTime       time.Time
	Version         string
}

// SyncManager manages the synchronization between the local file system and the remote storage
type SyncManager struct {
	uploader     *uploader.Uploader
	storage      storage.Storage
	watcher      *watcher.FileWatcher // Use concrete type instead of interface
	config       *config.Config
	stats        SyncStats
	state        SyncState
	deviceID     string
	syncInterval time.Duration
	stopChan     chan struct{}
	cancel       context.CancelFunc
	folders      map[string]*FolderSync
	mu           sync.RWMutex
}

// FolderSync manages synchronization for a specific folder
type FolderSync struct {
	ID              string
	Path            string
	ExcludePatterns []string
	LastSync        time.Time
	TwoWaySync      bool
	Enabled         bool
}

// NewSyncManager creates a new sync manager
func NewSyncManager(cfg *config.Config, storage storage.Storage, uploader *uploader.Uploader) (*SyncManager, error) {
	// Generate a Device ID if it doesn't exist
	deviceID := generateRandomID()

	sm := &SyncManager{
		uploader:     uploader,
		storage:      storage,
		config:       cfg,
		state:        SyncStateIdle,
		deviceID:     deviceID,
		syncInterval: time.Duration(cfg.Sync.IntervalMinutes) * time.Minute,
		stopChan:     make(chan struct{}),
		folders:      make(map[string]*FolderSync),
		stats: SyncStats{
			StartTime: time.Now(),
			Version:   "1.0.0", // Default version
		},
	}

	// Initialize folders from config
	for id, folder := range cfg.GetAllFolders() {
		sm.folders[id] = &FolderSync{
			ID:              id,
			Path:            folder.LocalPath,
			ExcludePatterns: folder.ExcludePatterns,
			LastSync:        time.Time{}, // Never synced
			TwoWaySync:      false,       // Default to one-way sync
			Enabled:         folder.Enabled,
		}
	}

	return sm, nil
}

// Start starts the sync manager
func (sm *SyncManager) Start() error {
	log.Info().Msg("Starting sync manager")

	// Create a context for all sync operations
	ctx, cancel := context.WithCancel(context.Background())

	// Store the cancel function to be used when stopping
	sm.cancel = cancel

	// Start file watcher
	fw, err := watcher.NewFileWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	sm.watcher = fw

	// Watch all enabled folders
	sm.mu.RLock()
	for _, folder := range sm.folders {
		if folder.Enabled {
			if err := sm.watcher.WatchPath(folder.Path, true, folder.ExcludePatterns); err != nil {
				log.Error().Err(err).Str("path", folder.Path).Msg("Failed to watch folder")
			} else {
				log.Info().Str("path", folder.Path).Msg("Started watching folder")
			}
		}
	}
	sm.mu.RUnlock()

	// Start the file watcher
	sm.watcher.Start()

	// Add handler for file events
	sm.watcher.AddHandler(func(event watcher.Event) {
		sm.handleFileEvent(ctx, Event{
			Path:      event.Path,
			Type:      event.Type,
			Timestamp: event.Timestamp,
		})
	})

	// Start periodic sync
	go sm.periodicSync(ctx)

	// Run initial scan if enabled
	if sm.config.Sync.AutoSync {
		go sm.FullSync(ctx)
	}

	return nil
}

// Stop stops the sync manager
func (sm *SyncManager) Stop() error {
	log.Info().Msg("Stopping sync manager")

	// Cancel context to stop all operations
	if sm.cancel != nil {
		sm.cancel()
	}

	// Close stop channel
	close(sm.stopChan)

	// Stop watcher
	if sm.watcher != nil {
		return sm.watcher.Stop()
	}

	return nil
}

// FullSync performs a full sync of all enabled folders
func (sm *SyncManager) FullSync(ctx context.Context) error {
	sm.mu.Lock()
	sm.state = SyncStateScanning
	sm.mu.Unlock()

	log.Info().Msg("Starting full sync")

	defer func() {
		sm.mu.Lock()
		sm.state = SyncStateIdle
		sm.mu.Unlock()
	}()

	sm.mu.RLock()
	folders := make([]*FolderSync, 0, len(sm.folders))
	for _, folder := range sm.folders {
		if folder.Enabled {
			folders = append(folders, folder)
		}
	}
	sm.mu.RUnlock()

	for _, folder := range folders {
		if err := sm.syncFolder(ctx, folder); err != nil {
			log.Error().Err(err).Str("folder", folder.Path).Msg("Failed to sync folder")
			sm.stats.Errors++
			continue
		}
	}

	sm.mu.Lock()
	sm.stats.LastSyncTime = time.Now()
	sm.mu.Unlock()

	log.Info().
		Int64("uploaded", sm.stats.FilesUploaded).
		Int64("bytes_uploaded", sm.stats.BytesUploaded).
		Msg("Full sync completed")

	return nil
}

// syncFolder syncs a specific folder
func (sm *SyncManager) syncFolder(ctx context.Context, folder *FolderSync) error {
	log.Info().Str("folder", folder.Path).Msg("Syncing folder")

	sm.mu.Lock()
	sm.state = SyncStateSyncing
	sm.mu.Unlock()

	// Walk through all files in the folder
	err := filepath.Walk(folder.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories for now
		if info.IsDir() {
			return nil
		}

		// Check if the path matches any exclude patterns
		relPath, err := filepath.Rel(folder.Path, path)
		if err != nil {
			return err
		}

		if watcher.ShouldExclude(relPath, folder.ExcludePatterns) {
			return nil
		}

		// Queue the file for upload
		if err := sm.uploader.QueueFile(path, folder.Path); err != nil {
			log.Error().Err(err).Str("path", path).Msg("Failed to queue file for upload")
			return nil // Continue with other files
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Update last sync time
	folder.LastSync = time.Now()

	// If two-way sync is enabled, download files from remote
	if folder.TwoWaySync {
		if err := sm.downloadFromRemote(ctx, folder); err != nil {
			return fmt.Errorf("failed to download from remote: %w", err)
		}
	}

	return nil
}

// downloadFromRemote downloads files from remote storage for two-way sync
func (sm *SyncManager) downloadFromRemote(ctx context.Context, folder *FolderSync) error {
	log.Info().Str("folder", folder.Path).Msg("Downloading remote changes")

	// Get remote file list for this folder
	remoteFiles, err := sm.storage.ListFiles(ctx, folder.ID)
	if err != nil {
		return fmt.Errorf("failed to list remote files: %w", err)
	}

	// Create a map of local files with their modification times for quick lookup
	localFiles := make(map[string]time.Time)
	err = filepath.Walk(folder.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue despite errors
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(folder.Path, path)
			if err != nil {
				return nil
			}

			// Skip excluded files
			if watcher.ShouldExclude(relPath, folder.ExcludePatterns) {
				return nil
			}

			localFiles[relPath] = info.ModTime()
		}
		return nil
	})

	if err != nil {
		log.Warn().Err(err).Str("folder", folder.Path).Msg("Error scanning local folder")
	}

	// Download files that are newer on remote or don't exist locally
	for _, remoteFile := range remoteFiles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Process file
		}

		// Extract relative path from remote file key
		// Key format is typically: folderID/relative/path/to/file.ext
		remotePath := strings.TrimPrefix(remoteFile.Key, folder.ID+"/")
		localModTime, exists := localFiles[remotePath]

		// Download file if it doesn't exist locally or is newer on remote
		if !exists || remoteFile.LastModified.After(localModTime) {
			localPath := filepath.Join(folder.Path, remotePath)

			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
				log.Error().Err(err).Str("path", localPath).Msg("Failed to create directory")
				continue
			}

			log.Info().Str("file", remotePath).Msg("Downloading file")

			// Create file for writing
			localFile, err := os.Create(localPath)
			if err != nil {
				log.Error().Err(err).Str("path", localPath).Msg("Failed to create local file")
				sm.stats.Errors++
				continue
			}

			// Download the file
			_, err = sm.storage.DownloadFile(ctx, remoteFile.Key, localFile, "")
			localFile.Close() // Close the file regardless of error

			if err != nil {
				log.Error().Err(err).Str("file", remotePath).Msg("Failed to download file")
				sm.stats.Errors++
				continue
			}

			// Update stats
			sm.mu.Lock()
			sm.stats.FilesDownloaded++
			sm.stats.BytesDownloaded += remoteFile.Size
			sm.mu.Unlock()

			// Set file modification time to match remote
			if err := os.Chtimes(localPath, remoteFile.LastModified, remoteFile.LastModified); err != nil {
				log.Warn().Err(err).Str("file", localPath).Msg("Failed to set file modification time")
			}

			log.Debug().
				Str("file", remotePath).
				Int64("size", remoteFile.Size).
				Time("modified", remoteFile.LastModified).
				Msg("File downloaded successfully")
		}
	}

	return nil
}

// handleFileEvent handles a file event from the watcher
func (sm *SyncManager) handleFileEvent(ctx context.Context, event Event) {
	// Find the folder this file belongs to
	var folderPath string
	for _, folder := range sm.folders {
		if event.Path != "" && isSubPath(folder.Path, event.Path) && folder.Enabled {
			folderPath = folder.Path
			break
		}
	}

	if folderPath == "" {
		log.Debug().Str("path", event.Path).Msg("File event for path not in any watched folder")
		return
	}

	log.Debug().
		Str("path", event.Path).
		Str("op", fmt.Sprintf("%v", event.Type)).
		Msg("Got file event")

	switch event.Type {
	case watcher.EventCreate:
		if err := sm.uploader.QueueFile(event.Path, folderPath); err != nil {
			log.Error().Err(err).Str("path", event.Path).Msg("Failed to queue file for upload")
		}
	case watcher.EventUpdate:
		if err := sm.uploader.QueueFile(event.Path, folderPath); err != nil {
			log.Error().Err(err).Str("path", event.Path).Msg("Failed to queue file for upload")
		}
	case watcher.EventDelete, watcher.EventRename:
		// Currently we don't handle remote deletes
		log.Debug().Str("path", event.Path).Msg("File removal detected, currently not propagated to remote")
	}
}

// periodicSync runs the sync operation periodically
func (sm *SyncManager) periodicSync(ctx context.Context) {
	ticker := time.NewTicker(sm.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sm.FullSync(ctx); err != nil {
				log.Error().Err(err).Msg("Periodic sync failed")
			}
		case <-sm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// GetSyncStats returns the current sync stats
func (sm *SyncManager) GetSyncStats() SyncStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.stats
}

// GetState returns the current sync state
func (sm *SyncManager) GetState() SyncState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// GetFolders returns the list of folders
func (sm *SyncManager) GetFolders() []*FolderSync {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	folders := make([]*FolderSync, 0, len(sm.folders))
	for _, folder := range sm.folders {
		folders = append(folders, folder)
	}
	return folders
}

// SyncFolder syncs a specific folder by ID
func (sm *SyncManager) SyncFolderByID(ctx context.Context, folderID string) error {
	sm.mu.RLock()
	folder, ok := sm.folders[folderID]
	sm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("folder with ID %s not found", folderID)
	}

	return sm.syncFolder(ctx, folder)
}

// AddFolder adds a new folder to be synced
func (sm *SyncManager) AddFolder(folder *FolderSync) error {
	// Validate folder
	if folder.Path == "" {
		return fmt.Errorf("folder path cannot be empty")
	}

	// Check if path exists
	info, err := os.Stat(folder.Path)
	if err != nil {
		return fmt.Errorf("folder path error: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", folder.Path)
	}

	// Generate ID if not provided
	if folder.ID == "" {
		folder.ID = uuid.New().String()
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if already exists
	for _, f := range sm.folders {
		if f.Path == folder.Path {
			return fmt.Errorf("folder already added with path: %s", folder.Path)
		}
	}

	// Add to sync manager
	sm.folders[folder.ID] = folder

	// Add to watcher if enabled
	if folder.Enabled && sm.watcher != nil {
		if err := sm.watcher.AddFolder(folder.Path, folder.ExcludePatterns); err != nil {
			return fmt.Errorf("failed to watch folder: %w", err)
		}
	}

	// Update config
	syncFolder := config.SyncFolder{
		LocalPath:       folder.Path,
		RemotePath:      folder.ID, // Usar ID como caminho remoto por padrão
		ExcludePatterns: folder.ExcludePatterns,
		Enabled:         folder.Enabled,
	}

	sm.config.SetSyncFolder(folder.ID, syncFolder)
	if err := config.SaveConfig(sm.config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// RemoveFolder removes a folder from being synced
func (sm *SyncManager) RemoveFolder(folderID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	folder, ok := sm.folders[folderID]
	if !ok {
		return fmt.Errorf("folder with ID %s not found", folderID)
	}

	// Remove from watcher if it's running
	if sm.watcher != nil && folder.Enabled {
		if err := sm.watcher.RemoveFolder(folder.Path); err != nil {
			log.Error().Err(err).Str("path", folder.Path).Msg("Failed to remove folder from watcher")
			// Continue anyway
		}
	}

	// Remove from folders map
	delete(sm.folders, folderID)

	// Update config
	sm.config.RemoveSyncFolder(folderID)

	// Save config
	if err := config.SaveConfig(sm.config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// EnableFolder enables synchronization for a folder
func (sm *SyncManager) EnableFolder(folderID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	folder, ok := sm.folders[folderID]
	if !ok {
		return fmt.Errorf("folder with ID %s not found", folderID)
	}

	if folder.Enabled {
		return nil // Already enabled
	}

	folder.Enabled = true

	// Add to watcher if it's running
	if sm.watcher != nil {
		if err := sm.watcher.AddFolder(folder.Path, folder.ExcludePatterns); err != nil {
			return fmt.Errorf("failed to watch folder: %w", err)
		}
	}

	// Update config
	if f, exists := sm.config.GetSyncFolder(folderID); exists {
		f.Enabled = true
		sm.config.SetSyncFolder(folderID, f)
	}

	// Save config
	if err := config.SaveConfig(sm.config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// DisableFolder disables synchronization for a folder
func (sm *SyncManager) DisableFolder(folderID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	folder, ok := sm.folders[folderID]
	if !ok {
		return fmt.Errorf("folder with ID %s not found", folderID)
	}

	if !folder.Enabled {
		return nil // Already disabled
	}

	folder.Enabled = false

	// Remove from watcher if it's running
	if sm.watcher != nil {
		if err := sm.watcher.RemoveFolder(folder.Path); err != nil {
			log.Error().Err(err).Str("path", folder.Path).Msg("Failed to remove folder from watcher")
			// Continue anyway
		}
	}

	// Update config
	if f, exists := sm.config.GetSyncFolder(folderID); exists {
		f.Enabled = false
		sm.config.SetSyncFolder(folderID, f)
	}

	// Save config
	if err := config.SaveConfig(sm.config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// UpdateFolder updates a folder's settings
func (sm *SyncManager) UpdateFolder(folderID string, update *FolderSync) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	folder, ok := sm.folders[folderID]
	if !ok {
		return fmt.Errorf("folder with ID %s not found", folderID)
	}

	// Remove from watcher if it was enabled
	if sm.watcher != nil && folder.Enabled {
		if err := sm.watcher.RemoveFolder(folder.Path); err != nil {
			log.Error().Err(err).Str("path", folder.Path).Msg("Failed to remove folder from watcher")
			// Continue anyway
		}
	}

	// Update folder properties
	folder.ExcludePatterns = update.ExcludePatterns
	folder.TwoWaySync = update.TwoWaySync

	// Only update path if it's provided and different
	if update.Path != "" && update.Path != folder.Path {
		// Check if path exists
		info, err := os.Stat(update.Path)
		if err != nil {
			return fmt.Errorf("folder path error: %w", err)
		}

		if !info.IsDir() {
			return fmt.Errorf("path is not a directory: %s", update.Path)
		}

		folder.Path = update.Path
	}

	// Add back to watcher if enabled
	if update.Enabled && sm.watcher != nil {
		if err := sm.watcher.AddFolder(folder.Path, folder.ExcludePatterns); err != nil {
			return fmt.Errorf("failed to watch folder: %w", err)
		}
	}

	folder.Enabled = update.Enabled

	// Update config
	if f, exists := sm.config.GetSyncFolder(folderID); exists {
		f.LocalPath = folder.Path
		f.ExcludePatterns = folder.ExcludePatterns
		f.Enabled = folder.Enabled
		sm.config.SetSyncFolder(folderID, f)
	}

	// Save config
	if err := config.SaveConfig(sm.config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// SyncNow triggers an immediate synchronization of all folders or a specific folder
func (sm *SyncManager) SyncNow(ctx context.Context, folderID string) error {
	if folderID != "" {
		log.Info().Str("folder_id", folderID).Msg("Syncing specific folder")
		return sm.SyncFolderByID(ctx, folderID)
	}

	log.Info().Msg("Syncing all folders")
	return sm.FullSync(ctx)
}

// ReloadConfiguration reloads the configuration from the config manager
func (sm *SyncManager) ReloadConfiguration(ctx context.Context) error {
	// Reload the configuration
	newCfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	sm.mu.Lock()

	// Create a map of existing folders
	existingFolders := make(map[string]*FolderSync)
	for id, folder := range sm.folders {
		existingFolders[id] = folder
	}

	// Update configuration reference
	sm.config = newCfg

	// Check for new folders to add or existing folders to update
	for id, folderConfig := range newCfg.GetAllFolders() {
		if existingFolder, exists := existingFolders[id]; exists {
			// Update existing folder if needed
			if existingFolder.Path != folderConfig.LocalPath ||
				existingFolder.Enabled != folderConfig.Enabled {

				// Update folder properties
				existingFolder.Path = folderConfig.LocalPath
				existingFolder.ExcludePatterns = folderConfig.ExcludePatterns
				existingFolder.Enabled = folderConfig.Enabled

				// Update watcher if needed
				if sm.watcher != nil {
					if existingFolder.Enabled {
						// Remove old path and add new one
						sm.watcher.RemoveFolder(existingFolder.Path)
						sm.watcher.AddFolder(folderConfig.LocalPath, folderConfig.ExcludePatterns)
					} else {
						// Just remove from watcher
						sm.watcher.RemoveFolder(existingFolder.Path)
					}
				}
			}

			// Remove from existing folders map
			delete(existingFolders, id)
		} else {
			// Add new folder
			sm.folders[id] = &FolderSync{
				ID:              id,
				Path:            folderConfig.LocalPath,
				ExcludePatterns: folderConfig.ExcludePatterns,
				LastSync:        time.Time{}, // Never synced
				TwoWaySync:      false,       // Default to one-way sync
				Enabled:         folderConfig.Enabled,
			}

			// Add to watcher if enabled
			if folderConfig.Enabled && sm.watcher != nil {
				if err := sm.watcher.AddFolder(folderConfig.LocalPath, folderConfig.ExcludePatterns); err != nil {
					log.Error().Err(err).Str("path", folderConfig.LocalPath).Msg("Failed to watch new folder")
				}
			}
		}
	}

	// Any folders still in existingFolders need to be removed
	for id, folder := range existingFolders {
		if sm.watcher != nil && folder.Enabled {
			sm.watcher.RemoveFolder(folder.Path)
		}
		delete(sm.folders, id)
	}

	// Update sync interval if changed
	newInterval := time.Duration(newCfg.Sync.IntervalMinutes) * time.Minute
	if sm.syncInterval != newInterval {
		sm.syncInterval = newInterval
		log.Info().Dur("interval", sm.syncInterval).Msg("Updated sync interval")
	}

	sm.mu.Unlock()

	log.Info().Msg("Configuration reloaded successfully")
	return nil
}

// PauseSync pauses the synchronization process
func (sm *SyncManager) PauseSync() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == SyncStateIdle || sm.state == SyncStateSyncing || sm.state == SyncStateScanning {
		log.Info().Msg("Pausing synchronization")
		sm.state = SyncStatePaused
	}
}

// ResumeSync resumes the synchronization process
func (sm *SyncManager) ResumeSync() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == SyncStatePaused {
		log.Info().Msg("Resuming synchronization")
		sm.state = SyncStateIdle
	}
}

// Health returns the health status of the sync manager
func (sm *SyncManager) Health() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	status := map[string]interface{}{
		"state":            string(sm.state),
		"uptime":           time.Since(sm.stats.StartTime).String(),
		"folders_count":    len(sm.folders),
		"enabled_folders":  0,
		"last_sync":        sm.stats.LastSyncTime,
		"files_uploaded":   sm.stats.FilesUploaded,
		"files_downloaded": sm.stats.FilesDownloaded,
		"bytes_uploaded":   sm.stats.BytesUploaded,
		"bytes_downloaded": sm.stats.BytesDownloaded,
		"errors":           sm.stats.Errors,
		"version":          sm.stats.Version,
	}

	// Count enabled folders
	for _, folder := range sm.folders {
		if folder.Enabled {
			status["enabled_folders"] = status["enabled_folders"].(int) + 1
		}
	}

	return status
}

// Helper functions

// generateRandomID generates a random ID
func generateRandomID() string {
	// Generate a UUID-like string
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// isSubPath checks if child is a subpath of parent
func isSubPath(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)

	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	return relative != ".." && !filepath.IsAbs(relative)
}
