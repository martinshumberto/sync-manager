package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SyncFolder represents a folder to be synchronized
type SyncFolder struct {
	LocalPath       string   `json:"local_path"`
	RemotePath      string   `json:"remote_path"`
	ExcludePatterns []string `json:"exclude_patterns,omitempty"`
	Enabled         bool     `json:"enabled"`
}

// SyncConfig contains synchronization settings
type SyncConfig struct {
	IntervalMinutes int  `json:"interval_minutes"`
	AutoSync        bool `json:"auto_sync"`
}

// ServerConfig contains settings for connecting to the server
type ServerConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	APIKey   string `json:"api_key,omitempty"`
}

// Config represents the application configuration
type Config struct {
	Server  ServerConfig          `json:"server"`
	Sync    SyncConfig            `json:"sync"`
	Folders map[string]SyncFolder `json:"folders"`

	filePath string
	mu       sync.RWMutex
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			URL: "https://sync-manager.example.com",
		},
		Sync: SyncConfig{
			IntervalMinutes: 15,
			AutoSync:        true,
		},
		Folders: make(map[string]SyncFolder),
	}
}

// LoadConfig loads the configuration from the specified file
func LoadConfig(filePath string) (*Config, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		cfg.filePath = filePath
		return cfg, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.filePath = filePath
	return &cfg, nil
}

// LoadConfigFromPath loads the configuration from the default location
func LoadConfigFromPath(configDir string) (*Config, error) {
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".cloudsync")
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return LoadConfig(filepath.Join(configDir, "config.json"))
}

// SaveConfig saves the configuration to the file
func SaveConfig(cfg *Config) error {
	cfg.mu.RLock()
	filePath := cfg.filePath
	cfg.mu.RUnlock()

	if filePath == "" {
		return fmt.Errorf("config file path not set")
	}

	return SaveConfigToFile(cfg, filePath)
}

// SaveConfigToFile saves the configuration to the specified file
func SaveConfigToFile(cfg *Config, filePath string) error {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetSyncFolder returns the configuration for a specific folder
func (c *Config) GetSyncFolder(id string) (SyncFolder, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	folder, exists := c.Folders[id]
	return folder, exists
}

// SetSyncFolder updates or adds a folder configuration
func (c *Config) SetSyncFolder(id string, folder SyncFolder) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Folders[id] = folder
}

// RemoveSyncFolder removes a folder configuration
func (c *Config) RemoveSyncFolder(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.Folders[id]
	if exists {
		delete(c.Folders, id)
	}
	return exists
}

// GetAllFolders returns all folder configurations
func (c *Config) GetAllFolders() map[string]SyncFolder {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	folders := make(map[string]SyncFolder, len(c.Folders))
	for id, folder := range c.Folders {
		folders[id] = folder
	}

	return folders
}

// UpdateServerConfig updates the server configuration
func (c *Config) UpdateServerConfig(server ServerConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Server = server
}

// UpdateSyncConfig updates the sync configuration
func (c *Config) UpdateSyncConfig(sync SyncConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Sync = sync
}
