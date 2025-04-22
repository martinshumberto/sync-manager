package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	common_config "github.com/martinshumberto/sync-manager/common/config"
	"github.com/rs/zerolog/log"
)

// LocalConfig holds configuration for local file storage
type LocalConfig struct {
	RootDir string
}

// NewLocalConfigFromCommon converts a common.LocalConfig to storage.LocalConfig
func NewLocalConfigFromCommon(commonCfg *common_config.LocalConfig) *LocalConfig {
	return &LocalConfig{
		RootDir: commonCfg.RootDir,
	}
}

// LocalStorage implements the Storage interface using the local file system
type LocalStorage struct {
	rootDir string
	config  *LocalConfig
}

// GetProvider returns the storage provider type
func (l *LocalStorage) GetProvider() StorageProvider {
	return ProviderLocal
}

// NewLocalStorage creates a new local storage client
func NewLocalStorage(cfg *LocalConfig) (*LocalStorage, error) {
	rootDir := filepath.Clean(cfg.RootDir)
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory: %w", err)
	}

	metadataDir := filepath.Join(rootDir, ".sync-manager")
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	return &LocalStorage{
		rootDir: rootDir,
		config:  cfg,
	}, nil
}

// UploadFile uploads a file to local storage
func (l *LocalStorage) UploadFile(ctx context.Context, key string, reader io.Reader, metadata map[string]string) (string, error) {
	key = strings.TrimPrefix(key, "/")

	filePath := filepath.Join(l.rootDir, key)
	dirPath := filepath.Dir(filePath)

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	tempFile := filePath + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)

	size, err := io.Copy(writer, reader)
	if err != nil {
		file.Close()
		os.Remove(tempFile)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(tempFile)
		return "", fmt.Errorf("failed to close file: %w", err)
	}

	hash := hex.EncodeToString(hasher.Sum(nil))

	metadataPath := l.getMetadataPath(key)
	metadataDir := filepath.Dir(metadataPath)
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		os.Remove(tempFile)
		return "", fmt.Errorf("failed to create metadata directory: %w", err)
	}

	metadata["hash_sha256"] = hash
	metadata["size"] = fmt.Sprintf("%d", size)
	metadata["modified_time"] = time.Now().UTC().Format(time.RFC3339)

	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		os.Remove(tempFile)
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataJson, 0644); err != nil {
		os.Remove(tempFile)
		return "", fmt.Errorf("failed to write metadata: %w", err)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile)
		os.Remove(metadataPath)
		return "", fmt.Errorf("failed to move file: %w", err)
	}

	log.Debug().
		Str("path", filePath).
		Str("hash", hash).
		Int64("size", size).
		Msg("Uploaded file to local storage")

	return hash, nil
}

// DownloadFile downloads a file from local storage
func (l *LocalStorage) DownloadFile(ctx context.Context, key string, writer io.Writer, versionID string) (map[string]string, error) {
	key = strings.TrimPrefix(key, "/")

	filePath := filepath.Join(l.rootDir, key)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(writer, file); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	metadata, err := l.readMetadata(key)
	if err != nil {
		return make(map[string]string), nil
	}

	log.Debug().
		Str("path", filePath).
		Msg("Downloaded file from local storage")

	return metadata, nil
}

// DeleteFile deletes a file from local storage
func (l *LocalStorage) DeleteFile(ctx context.Context, key string) error {
	key = strings.TrimPrefix(key, "/")

	filePath := filepath.Join(l.rootDir, key)

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	metadataPath := l.getMetadataPath(key)
	_ = os.Remove(metadataPath) // ignore error if metadata doesn't exist

	log.Debug().
		Str("path", filePath).
		Msg("Deleted file from local storage")

	return nil
}

// ListFiles lists files in local storage with the given prefix
func (l *LocalStorage) ListFiles(ctx context.Context, prefix string) ([]FileInfo, error) {
	prefix = strings.TrimPrefix(prefix, "/")

	dirPath := filepath.Join(l.rootDir, prefix)

	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []FileInfo{}, nil
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	var files []FileInfo

	if !info.IsDir() {
		relPath, err := filepath.Rel(l.rootDir, dirPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path: %w", err)
		}

		metadata, _ := l.readMetadata(relPath) // ignore error if metadata doesn't exist

		var hash string
		if h, ok := metadata["hash_sha256"]; ok {
			hash = h
		} else {
			hash = "unknown"
		}

		files = append(files, FileInfo{
			Key:          relPath,
			Size:         info.Size(),
			LastModified: info.ModTime(),
			ETag:         hash,
		})

		return files, nil
	}

	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}

		relPath, err := filepath.Rel(l.rootDir, path)
		if err != nil {
			return err
		}

		metadata, _ := l.readMetadata(relPath) // ignore error if metadata doesn't exist

		var hash string
		if h, ok := metadata["hash_sha256"]; ok {
			hash = h
		} else {
			hash = "unknown"
		}

		files = append(files, FileInfo{
			Key:          relPath,
			Size:         info.Size(),
			LastModified: info.ModTime(),
			ETag:         hash,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	log.Debug().
		Str("prefix", prefix).
		Int("count", len(files)).
		Msg("Listed files from local storage")

	return files, nil
}

// FileExists checks if a file exists in local storage
func (l *LocalStorage) FileExists(ctx context.Context, key string) (bool, error) {
	key = strings.TrimPrefix(key, "/")

	filePath := filepath.Join(l.rootDir, key)

	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat file: %w", err)
	}

	return true, nil
}

// getMetadataPath returns the path to the metadata file for a key
func (l *LocalStorage) getMetadataPath(key string) string {
	return filepath.Join(l.rootDir, ".sync-manager", key+".meta")
}

// readMetadata reads the metadata for a key
func (l *LocalStorage) readMetadata(key string) (map[string]string, error) {
	metadataPath := l.getMetadataPath(key)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata map[string]string
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return metadata, nil
}
