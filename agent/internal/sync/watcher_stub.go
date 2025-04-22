package sync

import "sync"

// StubEventType represents the type of file system event for the stub
type StubEventType int

const (
	// StubEventCreate is triggered when a file or directory is created
	StubEventCreate StubEventType = iota
	// StubEventUpdate is triggered when a file or directory is updated
	StubEventUpdate
	// StubEventDelete is triggered when a file or directory is deleted
	StubEventDelete
	// StubEventRename is triggered when a file or directory is renamed
	StubEventRename
)

// StubEvent represents a file system event for the stub
type StubEvent struct {
	Type      StubEventType
	Path      string
	Timestamp interface{} // Using interface{} to simplify
}

// StubHandlerFunc is the signature of the function for stub event handlers
type StubHandlerFunc func(StubEvent)

// StubFileWatcher is a stub implementation of the FileWatcher for tests
type StubFileWatcher struct {
	handlers     []StubHandlerFunc
	watchedPaths map[string]bool
	excludes     map[string][]string
	mu           sync.RWMutex
}

// NewStubFileWatcher creates a new stub file watcher
func NewStubFileWatcher() *StubFileWatcher {
	return &StubFileWatcher{
		handlers:     make([]StubHandlerFunc, 0),
		watchedPaths: make(map[string]bool),
		excludes:     make(map[string][]string),
	}
}

// AddHandler registers a handler for file events
func (fw *StubFileWatcher) AddHandler(handler StubHandlerFunc) {
	fw.handlers = append(fw.handlers, handler)
}

// WatchPath adds a path to be observed
func (fw *StubFileWatcher) WatchPath(path string, recursive bool, excludePatterns []string) error {
	fw.watchedPaths[path] = true
	if len(excludePatterns) > 0 {
		fw.excludes[path] = excludePatterns
	}
	return nil
}

// RemovePath stops observing a path
func (fw *StubFileWatcher) RemovePath(path string) error {
	delete(fw.watchedPaths, path)
	delete(fw.excludes, path)
	return nil
}

// Start starts observing file events
func (fw *StubFileWatcher) Start() {
	// Do nothing in tests
}

// Stop stops observing file events
func (fw *StubFileWatcher) Stop() error {
	// Do nothing in tests
	return nil
}

// TriggerEvent manually triggers an event (specific method for tests)
func (fw *StubFileWatcher) TriggerEvent(event StubEvent) {
	for _, handler := range fw.handlers {
		handler(event)
	}
}

// AddFolder is an alias for WatchPath with recursive=true
func (fw *StubFileWatcher) AddFolder(path string, excludePatterns []string) error {
	return fw.WatchPath(path, true, excludePatterns)
}

// RemoveFolder is an alias for RemovePath
func (fw *StubFileWatcher) RemoveFolder(path string) error {
	return fw.RemovePath(path)
}
