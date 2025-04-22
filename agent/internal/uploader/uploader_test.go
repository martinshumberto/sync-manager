package uploader

import (
	"context"
	"io"
	"testing"

	"github.com/martinshumberto/sync-manager/agent/internal/storage"
	"github.com/stretchr/testify/assert"
)

// mockStorage implements the Storage interface for testing
type mockStorage struct{}

func (m *mockStorage) UploadFile(ctx context.Context, key string, reader io.Reader, metadata map[string]string) (string, error) {
	return "mock-version-id", nil
}

func (m *mockStorage) DownloadFile(ctx context.Context, key string, writer io.Writer, versionID string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (m *mockStorage) DeleteFile(ctx context.Context, key string) error {
	return nil
}

func (m *mockStorage) ListFiles(ctx context.Context, prefix string) ([]storage.FileInfo, error) {
	return []storage.FileInfo{}, nil
}

func (m *mockStorage) FileExists(ctx context.Context, key string) (bool, error) {
	return true, nil
}

// GetProvider returns the storage provider type
func (m *mockStorage) GetProvider() storage.StorageProvider {
	return storage.ProviderLocal
}

// ConfigMock is a structure to simulate the configuration
type ConfigMock struct {
	MaxConcurrency int
	ThrottleBytes  int64
}

func TestNewUploader(t *testing.T) {
	cfg := &ConfigMock{
		MaxConcurrency: 4,
		ThrottleBytes:  1024,
	}

	mockStorage := &mockStorage{}

	uploader := NewUploaderWithConfig(mockStorage, cfg.MaxConcurrency, cfg.ThrottleBytes)

	assert.NotNil(t, uploader)
	assert.Equal(t, 4, uploader.maxConcurrency)
	assert.Equal(t, int64(1024), uploader.throttleBytes)
}

func TestUploader_StartStop(t *testing.T) {
	cfg := &ConfigMock{
		MaxConcurrency: 2,
		ThrottleBytes:  0,
	}

	mockStorage := &mockStorage{}
	uploader := NewUploaderWithConfig(mockStorage, cfg.MaxConcurrency, cfg.ThrottleBytes)
	uploader.Start()
	assert.True(t, uploader.running)
	uploader.Stop()

	assert.False(t, uploader.running)
}

// NewUploaderWithConfig is a helper to create an uploader with specific values for testing
func NewUploaderWithConfig(store storage.Storage, maxConcurrency int, throttleBytes int64) *Uploader {
	ctx, cancel := context.WithCancel(context.Background())

	return &Uploader{
		store:          store,
		taskQueue:      make(chan UploadTask, 1000),
		resultChan:     make(chan UploadResult, 100),
		maxConcurrency: maxConcurrency,
		throttleBytes:  throttleBytes,
		ctx:            ctx,
		cancel:         cancel,
	}
}
