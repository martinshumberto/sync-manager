package sync

import (
	"context"
	"io"
	"testing"

	"github.com/martinshumberto/sync-manager/agent/internal/config"
	"github.com/martinshumberto/sync-manager/agent/internal/storage"
	"github.com/martinshumberto/sync-manager/agent/internal/uploader"
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

func (m *mockStorage) GetProvider() storage.StorageProvider {
	return storage.ProviderLocal
}

// mockUploader implements the necessary interface for testing
type mockUploader struct {
	uploader.Uploader
}

func (m *mockUploader) Start() {}

func (m *mockUploader) Stop() {}

func (m *mockUploader) QueueFile(path, folderPath string) error {
	return nil
}

// mockWatcher is a mock of the FileWatcher for testing
type mockWatcher struct{}

func (m *mockWatcher) Start() {}

func (m *mockWatcher) Stop() error {
	return nil
}

func (m *mockWatcher) WatchPath(path string, recursive bool, excludePatterns []string) error {
	return nil
}

func (m *mockWatcher) RemovePath(path string) error {
	return nil
}

func (m *mockWatcher) AddHandler(handler interface{}) {}

func (m *mockWatcher) AddFolder(path string, excludePatterns []string) error {
	return nil
}

func (m *mockWatcher) RemoveFolder(path string) error {
	return nil
}

func TestNewSyncManager(t *testing.T) {
	cfg := config.DefaultConfig()

	mockStorage := &mockStorage{}
	mockUploader := &mockUploader{}

	manager, err := NewSyncManager(cfg, mockStorage, &mockUploader.Uploader)

	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, SyncStateIdle, manager.state)
}

func TestAddFolder(t *testing.T) {
	cfg := config.DefaultConfig()
	mockStorage := &mockStorage{}
	mockUploader := &mockUploader{}
	manager, _ := NewSyncManager(cfg, mockStorage, &mockUploader.Uploader)
	manager.watcher = &mockWatcher{}

	tmpFolder := t.TempDir()
	folder := &FolderSync{
		ID:              "test-folder",
		Path:            tmpFolder,
		Enabled:         true,
		TwoWaySync:      false,
		ExcludePatterns: []string{"*.tmp"},
	}

	err := manager.AddFolder(folder)
	assert.NoError(t, err)

	assert.Contains(t, manager.folders, "test-folder")
	assert.Equal(t, tmpFolder, manager.folders["test-folder"].Path)
}

func TestRemoveFolder(t *testing.T) {
	cfg := config.DefaultConfig()
	mockStorage := &mockStorage{}
	mockUploader := &mockUploader{}

	manager, _ := NewSyncManager(cfg, mockStorage, &mockUploader.Uploader)
	manager.watcher = &mockWatcher{}

	tmpFolder := t.TempDir()
	folder := &FolderSync{
		ID:              "test-folder",
		Path:            tmpFolder,
		Enabled:         true,
		TwoWaySync:      false,
		ExcludePatterns: []string{"*.tmp"},
	}

	_ = manager.AddFolder(folder)

	err := manager.RemoveFolder("test-folder")
	assert.NoError(t, err)

	_, exists := manager.folders["test-folder"]
	assert.False(t, exists)
}

func TestEnableDisableFolder(t *testing.T) {
	cfg := config.DefaultConfig()
	mockStorage := &mockStorage{}
	mockUploader := &mockUploader{}

	manager, _ := NewSyncManager(cfg, mockStorage, &mockUploader.Uploader)
	manager.watcher = &mockWatcher{}

	tmpFolder := t.TempDir()
	folder := &FolderSync{
		ID:              "test-folder",
		Path:            tmpFolder,
		Enabled:         false, // disabled by default
		TwoWaySync:      false,
		ExcludePatterns: []string{"*.tmp"},
	}

	_ = manager.AddFolder(folder)

	err := manager.EnableFolder("test-folder")
	assert.NoError(t, err)
	assert.True(t, manager.folders["test-folder"].Enabled)

	err = manager.DisableFolder("test-folder")
	assert.NoError(t, err)
	assert.False(t, manager.folders["test-folder"].Enabled)
}
