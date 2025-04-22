package client

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/martinshumberto/sync-manager/common/config"
	"github.com/martinshumberto/sync-manager/common/models"
	"github.com/rs/zerolog/log"
)

// AgentClient represents a client to communicate with the agent
type AgentClient struct {
	Config     *config.Config
	ConfigPath string
}

// NewAgentClient creates a new agent client
func NewAgentClient(cfg *config.Config, configPath string) *AgentClient {
	return &AgentClient{
		Config:     cfg,
		ConfigPath: configPath,
	}
}

// Health checks if the agent is running
func (c *AgentClient) Health() error {
	// Check if agent process is running
	running, err := c.isAgentRunning()
	if err != nil {
		return fmt.Errorf("failed to check agent status: %w", err)
	}

	if !running {
		return fmt.Errorf("agent is not running")
	}

	return nil
}

// GetFolders gets the list of sync folders from the config
func (c *AgentClient) GetFolders() ([]models.FolderResponse, error) {
	// Here we would convert the config folders to FolderResponse
	// In a real implementation, we might communicate with the agent via a pipe or socket

	folders := make([]models.FolderResponse, 0, len(c.Config.SyncFolders))
	for _, folder := range c.Config.SyncFolders {
		folders = append(folders, models.FolderResponse{
			FolderID: folder.ID,
			Name:     filepath.Base(folder.Path),
			Status:   c.getFolderStatus(folder),
		})
	}

	return folders, nil
}

// GetStatus gets the agent status
func (c *AgentClient) GetStatus() (interface{}, error) {
	// In a real implementation, we would get status directly from agent
	status := map[string]interface{}{
		"running": true,
		"version": "dev",
		"folders": len(c.Config.SyncFolders),
	}

	return status, nil
}

// TriggerSync requests the agent to start a sync operation
func (c *AgentClient) TriggerSync(folderID string) error {
	// In a real implementation, we might use a trigger file or IPC
	// For now, we'll just log a message since we don't have LastSyncRequest in the config

	if folderID == "" {
		// Trigger sync for all folders
		log.Info().Msg("Triggering sync for all folders")
		// In a real implementation, we would communicate with the agent
		return nil
	}

	// Find the folder
	for _, folder := range c.Config.SyncFolders {
		if folder.ID == folderID {
			log.Info().Str("folder", folder.Path).Msg("Triggering sync for folder")

			// In a real implementation, we would modify the folder or use IPC
			return nil
		}
	}

	return fmt.Errorf("folder not found: %s", folderID)
}

// Helper method to check if agent is running
func (c *AgentClient) isAgentRunning() (bool, error) {
	// This is a simple implementation for demonstration
	// In a real app, we would use proper system-specific methods

	if runtime.GOOS == "windows" {
		output, err := exec.Command("tasklist").Output()
		if err != nil {
			return false, err
		}
		return strings.Contains(string(output), "sync-manager-agent"), nil
	} else {
		// For Unix-like systems
		// Check if a PID file exists and the process is running
		pidFile := filepath.Join(os.TempDir(), "sync-manager-agent.pid")
		data, err := os.ReadFile(pidFile)
		if err != nil {
			// PID file doesn't exist
			return false, nil
		}

		pid := strings.TrimSpace(string(data))
		if pid == "" {
			return false, nil
		}

		// Check if process is running
		_, err = os.Stat(filepath.Join("/proc", pid))
		return err == nil, nil
	}
}

// Helper method to get the folder status
func (c *AgentClient) getFolderStatus(folder config.SyncFolder) string {
	if !folder.Enabled {
		return "disabled"
	}

	// In a real implementation, we would get the actual status from the agent
	return "active"
}
