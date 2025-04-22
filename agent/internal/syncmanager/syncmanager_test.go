package syncmanager

import (
	"os"
	"testing"

	"github.com/martinshumberto/sync-manager/agent/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewSyncManager(t *testing.T) {
	cfg := &config.Config{
		Folders: map[string]config.SyncFolder{
			"folder1": {
				LocalPath:       "/test/path1",
				RemotePath:      "remote/path1",
				ExcludePatterns: []string{"*.tmp"},
				Enabled:         true,
			},
			"folder2": {
				LocalPath:       "/test/path2",
				RemotePath:      "remote/path2",
				ExcludePatterns: []string{},
				Enabled:         false,
			},
		},
		Sync: config.SyncConfig{
			IntervalMinutes: 60,
		},
	}

	sm, err := NewSyncManager(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, sm)

	assert.Equal(t, 2, len(sm.folderStates))
	assert.Equal(t, "folder1", sm.folderStates["folder1"].ID)
	assert.Equal(t, "/test/path1", sm.folderStates["folder1"].LocalPath)
	assert.Equal(t, "remote/path1", sm.folderStates["folder1"].RemotePath)
	assert.Equal(t, []string{"*.tmp"}, sm.folderStates["folder1"].ExcludePatterns)
	assert.True(t, sm.folderStates["folder1"].Enabled)
	assert.Equal(t, StatusIdle, sm.folderStates["folder1"].Status)

	assert.Equal(t, "folder2", sm.folderStates["folder2"].ID)
	assert.False(t, sm.folderStates["folder2"].Enabled)
}

func TestGetStatus(t *testing.T) {
	cfg := &config.Config{
		Folders: make(map[string]config.SyncFolder),
		Sync: config.SyncConfig{
			IntervalMinutes: 60,
		},
	}

	sm, err := NewSyncManager(cfg)
	assert.NoError(t, err)

	assert.Equal(t, StatusIdle, sm.GetStatus())

	sm.status = StatusSyncing
	assert.Equal(t, StatusSyncing, sm.GetStatus())

	sm.setGlobalStatus(StatusError)
	assert.Equal(t, StatusError, sm.GetStatus())
}

func TestFolderState(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "syncmanager_test_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		Folders: map[string]config.SyncFolder{
			"test-folder": {
				LocalPath:       tempDir,
				RemotePath:      "remote/path",
				ExcludePatterns: []string{"*.tmp"},
				Enabled:         true,
			},
		},
		Sync: config.SyncConfig{
			IntervalMinutes: 60,
		},
	}

	sm, err := NewSyncManager(cfg)
	assert.NoError(t, err)

	state, err := sm.GetFolderState("test-folder")
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, "test-folder", state.ID)
	assert.Equal(t, tempDir, state.LocalPath)

	_, err = sm.GetFolderState("non-existent")
	assert.Error(t, err)

	states := sm.GetAllFolderStates()
	assert.Equal(t, 1, len(states))
	assert.Contains(t, states, "test-folder")
}
