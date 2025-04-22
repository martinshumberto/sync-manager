package syncmanager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/martinshumberto/sync-manager/agent/internal/config"
	"github.com/martinshumberto/sync-manager/agent/internal/watcher"
)

// SyncStatus represents the status of a synchronization operation
type SyncStatus string

const (
	// StatusIdle indicates that the sync manager is idle
	StatusIdle SyncStatus = "idle"
	// StatusSyncing indicates that synchronization is in progress
	StatusSyncing SyncStatus = "syncing"
	// StatusError indicates that an error occurred during synchronization
	StatusError SyncStatus = "error"
)

// SyncStats tracks synchronization statistics
type SyncStats struct {
	LastSync        time.Time `json:"last_sync"`
	FilesUploaded   int64     `json:"files_uploaded"`
	FilesDownloaded int64     `json:"files_downloaded"`
	BytesUploaded   int64     `json:"bytes_uploaded"`
	BytesDownloaded int64     `json:"bytes_downloaded"`
	Errors          int64     `json:"errors"`
}

// FolderState tracks the state of a synchronized folder
type FolderState struct {
	ID              string     `json:"id"`
	LocalPath       string     `json:"local_path"`
	RemotePath      string     `json:"remote_path"`
	Status          SyncStatus `json:"status"`
	LastError       string     `json:"last_error,omitempty"`
	Stats           SyncStats  `json:"stats"`
	ExcludePatterns []string   `json:"exclude_patterns,omitempty"`
	Enabled         bool       `json:"enabled"`
}

// SyncManager handles synchronization of folders
type SyncManager struct {
	config         *config.Config
	fileWatcher    *watcher.FileWatcher
	folderStates   map[string]*FolderState
	syncInterval   time.Duration
	syncInProgress bool
	status         SyncStatus
	eventHandlers  []func(folder string, status SyncStatus)
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// NewSyncManager creates a new sync manager
func NewSyncManager(cfg *config.Config) (*SyncManager, error) {
	fw, err := watcher.NewFileWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	sm := &SyncManager{
		config:       cfg,
		fileWatcher:  fw,
		folderStates: make(map[string]*FolderState),
		syncInterval: time.Duration(cfg.Sync.IntervalMinutes) * time.Minute,
		status:       StatusIdle,
		ctx:          ctx,
		cancel:       cancel,
	}

	fw.AddHandler(sm.handleFileEvent)

	for id, folder := range cfg.Folders {
		sm.folderStates[id] = &FolderState{
			ID:              id,
			LocalPath:       folder.LocalPath,
			RemotePath:      folder.RemotePath,
			Status:          StatusIdle,
			ExcludePatterns: folder.ExcludePatterns,
			Enabled:         folder.Enabled,
			Stats: SyncStats{
				LastSync: time.Time{}, // Zero time means never synced
			},
		}
	}

	return sm, nil
}

// Start starts the sync manager
func (sm *SyncManager) Start() error {
	log.Info().Msg("Starting sync manager")

	sm.fileWatcher.Start()

	for _, folderState := range sm.folderStates {
		if !folderState.Enabled {
			log.Info().Str("folder", folderState.ID).Msg("Folder disabled, skipping")
			continue
		}

		if err := os.MkdirAll(folderState.LocalPath, 0755); err != nil {
			log.Error().Err(err).Str("path", folderState.LocalPath).Msg("Failed to create folder")
			continue
		}

		if err := sm.fileWatcher.WatchDirectory(folderState.LocalPath); err != nil {
			log.Error().Err(err).Str("path", folderState.LocalPath).Msg("Failed to watch folder")
			continue
		}

		log.Info().Str("folder", folderState.ID).Str("path", folderState.LocalPath).Msg("Watching folder")
	}

	sm.wg.Add(1)
	go sm.periodicSync()

	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		if err := sm.SyncAll(); err != nil {
			log.Error().Err(err).Msg("Initial sync failed")
		}
	}()

	return nil
}

// Stop stops the sync manager
func (sm *SyncManager) Stop() {
	log.Info().Msg("Stopping sync manager")

	sm.cancel()
	sm.fileWatcher.Stop()
	sm.wg.Wait()

	log.Info().Msg("Sync manager stopped")
}

// periodicSync performs synchronization at regular intervals
func (sm *SyncManager) periodicSync() {
	defer sm.wg.Done()

	ticker := time.NewTicker(sm.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			if err := sm.SyncAll(); err != nil {
				log.Error().Err(err).Msg("Periodic sync failed")
			}
		}
	}
}

// SyncAll synchronizes all enabled folders
func (sm *SyncManager) SyncAll() error {
	sm.mu.Lock()
	if sm.syncInProgress {
		sm.mu.Unlock()
		return fmt.Errorf("sync already in progress")
	}
	sm.syncInProgress = true
	sm.setGlobalStatus(StatusSyncing)
	sm.mu.Unlock()

	defer func() {
		sm.mu.Lock()
		sm.syncInProgress = false
		sm.setGlobalStatus(StatusIdle)
		sm.mu.Unlock()
	}()

	log.Info().Msg("Starting synchronization of all folders")

	var wg sync.WaitGroup
	var syncErr error
	var errMu sync.Mutex

	for id, folderState := range sm.folderStates {
		if !folderState.Enabled {
			continue
		}

		wg.Add(1)
		go func(id string, state *FolderState) {
			defer wg.Done()

			if err := sm.syncFolder(id); err != nil {
				errMu.Lock()
				syncErr = err
				errMu.Unlock()
			}
		}(id, folderState)
	}

	wg.Wait()
	return syncErr
}

// SyncFolder synchronizes a specific folder
func (sm *SyncManager) SyncFolder(folderID string) error {
	sm.mu.RLock()
	folderState, exists := sm.folderStates[folderID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("folder %s does not exist", folderID)
	}

	if !folderState.Enabled {
		return fmt.Errorf("folder %s is disabled", folderID)
	}

	return sm.syncFolder(folderID)
}

// syncFolder performs the actual synchronization of a folder
func (sm *SyncManager) syncFolder(folderID string) error {
	sm.mu.Lock()
	folderState := sm.folderStates[folderID]
	folderState.Status = StatusSyncing
	sm.notifyStatusChange(folderID, StatusSyncing)
	sm.mu.Unlock()

	defer func() {
		sm.mu.Lock()
		folderState.Status = StatusIdle
		sm.notifyStatusChange(folderID, StatusIdle)
		sm.mu.Unlock()
	}()

	log.Info().Str("folder", folderID).Msg("Synchronizing folder")

	// 1. Scan local directory for files
	localFiles := make(map[string]time.Time)
	err := filepath.Walk(folderState.LocalPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("Error accessing path")
			return nil // Continue with other files
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(folderState.LocalPath, path)
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("Failed to get relative path")
			return nil
		}

		// Check exclusion patterns
		for _, pattern := range folderState.ExcludePatterns {
			matched, err := filepath.Match(pattern, relPath)
			if err != nil {
				log.Error().Err(err).Str("pattern", pattern).Msg("Invalid pattern")
				continue
			}
			if matched {
				return nil // Skip excluded files
			}
		}

		// Store file info
		localFiles[relPath] = info.ModTime()
		return nil
	})

	if err != nil {
		log.Error().Err(err).Str("folder", folderID).Msg("Failed to scan local directory")
		return err
	}

	// 2. Upload new and modified files
	var filesUploaded int64
	var bytesUploaded int64

	for relPath, _ := range localFiles {
		// Construct remote key (used in real implementation)
		remoteKey := filepath.Join(folderState.RemotePath, relPath)
		localPath := filepath.Join(folderState.LocalPath, relPath)

		// Check if we should upload this file
		// In a real implementation, we would check against remote state
		// For now, we'll upload all files

		file, err := os.Open(localPath)
		if err != nil {
			log.Error().Err(err).Str("path", localPath).Msg("Failed to open file")
			continue
		}

		// Get file size
		fileInfo, err := file.Stat()
		if err != nil {
			file.Close()
			log.Error().Err(err).Str("path", localPath).Msg("Failed to get file info")
			continue
		}

		// Upload file
		log.Debug().
			Str("file", relPath).
			Str("remote_key", remoteKey).
			Msg("Uploading file")

		// In a real implementation, we would use the storage interface here
		// For demonstration purposes, we'll just simulate an upload
		time.Sleep(100 * time.Millisecond)

		file.Close()

		// Update stats
		filesUploaded++
		bytesUploaded += fileInfo.Size()
	}

	// Update sync statistics
	sm.mu.Lock()
	folderState.Stats.LastSync = time.Now()
	folderState.Stats.FilesUploaded += filesUploaded
	folderState.Stats.BytesUploaded += bytesUploaded
	sm.mu.Unlock()

	log.Info().
		Str("folder", folderID).
		Int64("files_uploaded", filesUploaded).
		Int64("bytes_uploaded", bytesUploaded).
		Msg("Folder synchronized")
	return nil
}

// handleFileEvent processes a file system event
func (sm *SyncManager) handleFileEvent(event watcher.FileEvent) {
	// Find which folder this event belongs to
	var folderID string
	var folderPath string

	for id, state := range sm.folderStates {
		if filepath.HasPrefix(event.Path, state.LocalPath) {
			folderID = id
			folderPath = state.LocalPath
			break
		}
	}

	if folderID == "" {
		// Event not related to any tracked folder
		return
	}

	// Check if folder is enabled
	sm.mu.RLock()
	folderState := sm.folderStates[folderID]
	enabled := folderState.Enabled
	sm.mu.RUnlock()

	if !enabled {
		return
	}

	// Get relative path within the folder
	relPath, err := filepath.Rel(folderPath, event.Path)
	if err != nil {
		log.Error().Err(err).Str("path", event.Path).Msg("Failed to get relative path")
		return
	}

	// Check if the file matches exclude patterns
	for _, pattern := range folderState.ExcludePatterns {
		matched, err := filepath.Match(pattern, relPath)
		if err != nil {
			log.Error().Err(err).Str("pattern", pattern).Str("path", relPath).Msg("Invalid exclude pattern")
			continue
		}
		if matched {
			log.Debug().Str("path", event.Path).Str("pattern", pattern).Msg("File excluded by pattern")
			return
		}
	}

	// Ignore directory events
	fileInfo, err := os.Stat(event.Path)
	if err == nil && fileInfo.IsDir() {
		return
	}

	// Process the event based on its type
	switch event.Type {
	case watcher.EventCreate, watcher.EventModify:
		log.Debug().
			Str("folder", folderID).
			Str("path", event.Path).
			Str("event", eventTypeToString(event.Type)).
			Msg("File created or modified")

		// Get the remote key for this file
		remoteKey := filepath.Join(folderState.RemotePath, relPath)

		// In a real implementation, we would queue this file for upload
		// For demonstration, we'll just log it
		log.Info().
			Str("path", event.Path).
			Str("remote_key", remoteKey).
			Msg("File queued for upload")

		// Update stats
		if fileInfo != nil {
			sm.mu.Lock()
			folderState.Stats.FilesUploaded++
			folderState.Stats.BytesUploaded += fileInfo.Size()
			sm.mu.Unlock()
		}

	case watcher.EventDelete:
		log.Debug().Str("folder", folderID).Str("path", event.Path).Msg("File deleted")

		// Get the remote key for this file
		remoteKey := filepath.Join(folderState.RemotePath, relPath)

		// In a real implementation, we would queue a delete operation
		// For demonstration, we'll just log it
		log.Info().
			Str("path", event.Path).
			Str("remote_key", remoteKey).
			Msg("File deletion queued")

	case watcher.EventRename:
		log.Debug().Str("folder", folderID).Str("path", event.Path).Msg("File renamed")

		// In a real implementation, we would handle file renames
		// This could involve a delete followed by an upload
		log.Info().
			Str("path", event.Path).
			Msg("File rename detected (handled as delete and create)")
	}
}

// eventTypeToString converts an EventType to a readable string
func eventTypeToString(eventType watcher.EventType) string {
	switch eventType {
	case watcher.EventCreate:
		return "Create"
	case watcher.EventModify:
		return "Modify"
	case watcher.EventDelete:
		return "Delete"
	case watcher.EventRename:
		return "Rename"
	default:
		return "Unknown"
	}
}

// AddEventHandler adds a handler for status change events
func (sm *SyncManager) AddEventHandler(handler func(folder string, status SyncStatus)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.eventHandlers = append(sm.eventHandlers, handler)
}

// notifyStatusChange notifies all registered handlers of a status change
func (sm *SyncManager) notifyStatusChange(folder string, status SyncStatus) {
	for _, handler := range sm.eventHandlers {
		go handler(folder, status)
	}
}

// GetFolderState returns the current state of a folder
func (sm *SyncManager) GetFolderState(folderID string) (*FolderState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.folderStates[folderID]
	if !exists {
		return nil, fmt.Errorf("folder %s does not exist", folderID)
	}

	// Return a copy to prevent concurrent modification
	stateCopy := *state
	return &stateCopy, nil
}

// GetAllFolderStates returns the current state of all folders
func (sm *SyncManager) GetAllFolderStates() map[string]FolderState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	states := make(map[string]FolderState, len(sm.folderStates))
	for id, state := range sm.folderStates {
		states[id] = *state
	}

	return states
}

// setGlobalStatus sets the global status of the sync manager
func (sm *SyncManager) setGlobalStatus(status SyncStatus) {
	sm.status = status
	// Could trigger global status event handlers here
}

// GetStatus returns the current status of the sync manager
func (sm *SyncManager) GetStatus() SyncStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.status
}

// EnableFolder enables synchronization for a folder
func (sm *SyncManager) EnableFolder(folderID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.folderStates[folderID]
	if !exists {
		return fmt.Errorf("folder %s does not exist", folderID)
	}

	if state.Enabled {
		return nil // Already enabled
	}

	state.Enabled = true

	// Start watching the folder
	if err := sm.fileWatcher.WatchDirectory(state.LocalPath); err != nil {
		log.Error().Err(err).Str("path", state.LocalPath).Msg("Failed to watch folder")
		return err
	}

	log.Info().Str("folder", folderID).Msg("Folder enabled")
	return nil
}

// DisableFolder disables synchronization for a folder
func (sm *SyncManager) DisableFolder(folderID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.folderStates[folderID]
	if !exists {
		return fmt.Errorf("folder %s does not exist", folderID)
	}

	if !state.Enabled {
		return nil // Already disabled
	}

	state.Enabled = false

	// Stop watching the folder
	if err := sm.fileWatcher.UnwatchDirectory(state.LocalPath); err != nil {
		log.Error().Err(err).Str("path", state.LocalPath).Msg("Failed to unwatch folder")
		return err
	}

	log.Info().Str("folder", folderID).Msg("Folder disabled")
	return nil
}

// AddFolder adds a new folder to be synchronized
func (sm *SyncManager) AddFolder(id, localPath, remotePath string, excludePatterns []string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.folderStates[id]; exists {
		return fmt.Errorf("folder %s already exists", id)
	}

	// Ensure the local path exists
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Add to config
	sm.config.Folders[id] = config.SyncFolder{
		LocalPath:       localPath,
		RemotePath:      remotePath,
		ExcludePatterns: excludePatterns,
		Enabled:         true,
	}

	// Add to folder states
	sm.folderStates[id] = &FolderState{
		ID:              id,
		LocalPath:       localPath,
		RemotePath:      remotePath,
		Status:          StatusIdle,
		ExcludePatterns: excludePatterns,
		Enabled:         true,
		Stats: SyncStats{
			LastSync: time.Time{}, // Zero time means never synced
		},
	}

	// Start watching the folder
	if err := sm.fileWatcher.WatchDirectory(localPath); err != nil {
		log.Error().Err(err).Str("path", localPath).Msg("Failed to watch folder")
		return err
	}

	// Save the config
	if err := config.SaveConfig(sm.config); err != nil {
		log.Error().Err(err).Msg("Failed to save config")
		return err
	}

	log.Info().Str("folder", id).Str("path", localPath).Msg("Folder added")
	return nil
}

// RemoveFolder removes a folder from synchronization
func (sm *SyncManager) RemoveFolder(folderID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.folderStates[folderID]
	if !exists {
		return fmt.Errorf("folder %s does not exist", folderID)
	}

	// Stop watching the folder
	if state.Enabled {
		if err := sm.fileWatcher.UnwatchDirectory(state.LocalPath); err != nil {
			log.Error().Err(err).Str("path", state.LocalPath).Msg("Failed to unwatch folder")
			// Continue anyway
		}
	}

	// Remove from config
	delete(sm.config.Folders, folderID)

	// Remove from folder states
	delete(sm.folderStates, folderID)

	// Save the config
	if err := config.SaveConfig(sm.config); err != nil {
		log.Error().Err(err).Msg("Failed to save config")
		return err
	}

	log.Info().Str("folder", folderID).Msg("Folder removed")
	return nil
}

// UpdateFolderConfig updates the configuration for a folder
func (sm *SyncManager) UpdateFolderConfig(folderID string, localPath, remotePath string, excludePatterns []string, enabled bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.folderStates[folderID]
	if !exists {
		return fmt.Errorf("folder %s does not exist", folderID)
	}

	// Check if we need to stop watching the old path
	if state.Enabled && (state.LocalPath != localPath || !enabled) {
		if err := sm.fileWatcher.UnwatchDirectory(state.LocalPath); err != nil {
			log.Error().Err(err).Str("path", state.LocalPath).Msg("Failed to unwatch folder")
			// Continue anyway
		}
	}

	// Update config
	sm.config.Folders[folderID] = config.SyncFolder{
		LocalPath:       localPath,
		RemotePath:      remotePath,
		ExcludePatterns: excludePatterns,
		Enabled:         enabled,
	}

	// Update folder state
	state.LocalPath = localPath
	state.RemotePath = remotePath
	state.ExcludePatterns = excludePatterns
	state.Enabled = enabled

	// Start watching the new path if enabled
	if enabled && (state.LocalPath != localPath || !state.Enabled) {
		// Ensure the local path exists
		if err := os.MkdirAll(localPath, 0755); err != nil {
			return fmt.Errorf("failed to create local directory: %w", err)
		}

		if err := sm.fileWatcher.WatchDirectory(localPath); err != nil {
			log.Error().Err(err).Str("path", localPath).Msg("Failed to watch folder")
			return err
		}
	}

	// Save the config
	if err := config.SaveConfig(sm.config); err != nil {
		log.Error().Err(err).Msg("Failed to save config")
		return err
	}

	log.Info().Str("folder", folderID).Msg("Folder configuration updated")
	return nil
}
