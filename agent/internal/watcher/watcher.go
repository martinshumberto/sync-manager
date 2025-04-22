package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

// EventType represents the type of file system event
type EventType int

const (
	// EventCreate is triggered when a file or directory is created
	EventCreate EventType = iota
	// EventUpdate is triggered when a file or directory is updated
	EventUpdate
	// EventDelete is triggered when a file or directory is deleted
	EventDelete
	// EventRename is triggered when a file or directory is renamed
	EventRename
)

// Aliases para compatibilidade com código existente
const (
	Create = EventCreate
	Write  = EventUpdate
	Chmod  = EventUpdate
	Remove = EventDelete
	Rename = EventRename
)

// Event represents a file system event
type Event struct {
	Type      EventType
	Path      string
	Timestamp time.Time
}

// HandlerFunc is the function signature for event handlers
type HandlerFunc = func(Event)

// FileWatcher watches for file system changes
type FileWatcher struct {
	watcher      *fsnotify.Watcher
	watchedPaths map[string]bool
	handlers     []HandlerFunc
	excludes     map[string][]string // Map of root path to exclude patterns
	mu           sync.RWMutex
	done         chan struct{}
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher() (*FileWatcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	fw := &FileWatcher{
		watcher:      fsWatcher,
		watchedPaths: make(map[string]bool),
		handlers:     make([]HandlerFunc, 0),
		excludes:     make(map[string][]string),
		done:         make(chan struct{}),
	}

	return fw, nil
}

// AddHandler registers a handler for file events
func (fw *FileWatcher) AddHandler(handler HandlerFunc) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	fw.handlers = append(fw.handlers, handler)
}

// WatchPath adds a path to be watched
func (fw *FileWatcher) WatchPath(path string, recursive bool, excludePatterns []string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if path exists
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Store exclude patterns for this root
	if len(excludePatterns) > 0 {
		fw.excludes[absPath] = excludePatterns
	}

	if fileInfo.IsDir() && recursive {
		// Watch all subdirectories as well
		err = filepath.Walk(absPath, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				log.Warn().Err(err).Str("path", walkPath).Msg("Error walking directory")
				return nil // Continue despite error
			}

			if !info.IsDir() {
				return nil // Skip files
			}

			// Check if this directory should be excluded
			if fw.shouldExclude(absPath, walkPath) {
				log.Debug().Str("path", walkPath).Msg("Excluding directory from watch")
				return filepath.SkipDir
			}

			if err := fw.watcher.Add(walkPath); err != nil {
				log.Warn().Err(err).Str("path", walkPath).Msg("Failed to watch directory")
				return nil // Continue despite error
			}

			fw.watchedPaths[walkPath] = true
			log.Debug().Str("path", walkPath).Msg("Watching directory")
			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to recursively watch path: %w", err)
		}
	} else {
		// Just watch this single path
		if err := fw.watcher.Add(absPath); err != nil {
			return fmt.Errorf("failed to watch path: %w", err)
		}
		fw.watchedPaths[absPath] = true
		log.Debug().Str("path", absPath).Msg("Watching path")
	}

	return nil
}

// RemovePath stops watching a path
func (fw *FileWatcher) RemovePath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Remove this path and all subdirectories from the watch list
	for watchedPath := range fw.watchedPaths {
		if watchedPath == absPath || isSubdirectory(watchedPath, absPath) {
			if err := fw.watcher.Remove(watchedPath); err != nil {
				log.Warn().Err(err).Str("path", watchedPath).Msg("Failed to remove watch")
			} else {
				delete(fw.watchedPaths, watchedPath)
				log.Debug().Str("path", watchedPath).Msg("Stopped watching path")
			}
		}
	}

	// Remove exclude patterns for this root
	delete(fw.excludes, absPath)

	return nil
}

// Start begins watching for file events
func (fw *FileWatcher) Start() {
	go fw.watch()
}

// Stop stops watching for file events
func (fw *FileWatcher) Stop() error {
	close(fw.done)
	return fw.watcher.Close()
}

// watch processes file events
func (fw *FileWatcher) watch() {
	for {
		select {
		case <-fw.done:
			return
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Convert fsnotify event to our event type
			var eventType EventType
			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				eventType = EventCreate
				// If it's a new directory, we need to watch it too if recursive
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					fw.mu.Lock()
					// Check for any root path this might belong to
					for rootPath := range fw.excludes {
						if isSubdirectory(event.Name, rootPath) && !fw.shouldExclude(rootPath, event.Name) {
							if err := fw.watcher.Add(event.Name); err == nil {
								fw.watchedPaths[event.Name] = true
								log.Debug().Str("path", event.Name).Msg("Watching new directory")
							}
							break
						}
					}
					fw.mu.Unlock()
				}
			case event.Op&fsnotify.Write == fsnotify.Write:
				eventType = EventUpdate
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				eventType = EventDelete
				// Remove from watched paths
				fw.mu.Lock()
				delete(fw.watchedPaths, event.Name)
				fw.mu.Unlock()
			case event.Op&fsnotify.Rename == fsnotify.Rename:
				eventType = EventRename
				// Remove from watched paths
				fw.mu.Lock()
				delete(fw.watchedPaths, event.Name)
				fw.mu.Unlock()
			default:
				continue // Skip other events
			}

			fw.mu.RLock()
			handlers := make([]HandlerFunc, len(fw.handlers))
			copy(handlers, fw.handlers)
			fw.mu.RUnlock()

			for _, handler := range handlers {
				handler(Event{
					Type:      eventType,
					Path:      event.Name,
					Timestamp: time.Now(),
				})
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Error().Err(err).Msg("Watcher error")
		}
	}
}

// ShouldExclude verifica se um caminho deve ser excluído com base em padrões de exclusão
func ShouldExclude(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, path)
		if err != nil {
			// Se o padrão for inválido, ignoramos
			continue
		}
		if matched {
			return true
		}
	}

	return false
}

// shouldExclude verifica se um caminho deve ser excluído da observação
func (fw *FileWatcher) shouldExclude(rootPath, path string) bool {
	if patterns, ok := fw.excludes[rootPath]; ok {
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return false
		}
		return ShouldExclude(relPath, patterns)
	}
	return false
}

// ListWatchedPaths returns a list of all paths being watched
func (fw *FileWatcher) ListWatchedPaths() []string {
	fw.mu.RLock()
	defer fw.mu.RUnlock()

	paths := make([]string, 0, len(fw.watchedPaths))
	for path := range fw.watchedPaths {
		paths = append(paths, path)
	}
	return paths
}

// isSubdirectory checks if child is a subdirectory of parent
func isSubdirectory(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel)
}

// WatchDirectory adds a directory to be watched (alias for WatchPath with recursive=true)
func (fw *FileWatcher) WatchDirectory(path string) error {
	return fw.WatchPath(path, true, nil)
}

// UnwatchDirectory stops watching a directory (alias for RemovePath)
func (fw *FileWatcher) UnwatchDirectory(path string) error {
	return fw.RemovePath(path)
}

// FileEvent is an alias for Event for backward compatibility
type FileEvent = Event

// EventModify is an alias for EventUpdate for backward compatibility
const EventModify = EventUpdate

// AddFolder adds a folder to be watched with its exclusions
func (fw *FileWatcher) AddFolder(path string, excludePatterns []string) error {
	return fw.WatchPath(path, true, excludePatterns)
}

// RemoveFolder removes a folder from observation
func (fw *FileWatcher) RemoveFolder(path string) error {
	return fw.RemovePath(path)
}
