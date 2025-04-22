package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	common_config "github.com/martinshumberto/sync-manager/common/config"
)

// FileInfo represents information about a file in storage
type FileInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string // Entity tag (unique identifier)
}

// StorageProvider identifies the type of storage provider
type StorageProvider string

const (
	ProviderS3    StorageProvider = "s3"
	ProviderGCS   StorageProvider = "gcs"
	ProviderMinio StorageProvider = "minio" // local development
	ProviderLocal StorageProvider = "local"
)

// Storage defines the interface for file storage operations
type Storage interface {
	// UploadFile uploads a file to storage and returns the version ID (if available)
	UploadFile(ctx context.Context, key string, reader io.Reader, metadata map[string]string) (string, error)

	// DownloadFile downloads a file from storage and returns its metadata
	DownloadFile(ctx context.Context, key string, writer io.Writer, versionID string) (map[string]string, error)

	// DeleteFile deletes a file from storage
	DeleteFile(ctx context.Context, key string) error

	// ListFiles lists files in storage with the given prefix
	ListFiles(ctx context.Context, prefix string) ([]FileInfo, error)

	// FileExists checks if a file exists in storage
	FileExists(ctx context.Context, key string) (bool, error)

	// GetProvider returns the storage provider type
	GetProvider() StorageProvider
}

// StorageFactory creates storage implementations based on configuration
func StorageFactory(cfg *common_config.Config) (Storage, error) {
	switch StorageProvider(cfg.StorageProvider) {
	case ProviderS3:
		s3cfg := NewS3ConfigFromCommon(&cfg.S3Config)
		return NewS3Storage(s3cfg)
	case ProviderMinio:
		minioCfg := NewMinioConfigFromCommon(&cfg.MinioConfig)
		return NewMinioStorage(minioCfg)
	case ProviderGCS:
		gcsCfg := NewGCSConfigFromCommon(&cfg.GCSConfig)
		return NewGCSStorage(gcsCfg)
	case ProviderLocal:
		localCfg := NewLocalConfigFromCommon(&cfg.LocalConfig)
		return NewLocalStorage(localCfg)
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", cfg.StorageProvider)
	}
}
