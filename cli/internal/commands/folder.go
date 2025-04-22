package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/martinshumberto/sync-manager/cli/internal/client"
	"github.com/martinshumberto/sync-manager/cli/internal/services"
	"github.com/martinshumberto/sync-manager/common/config"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// CreateFolderCommands creates commands for folder management
func CreateFolderCommands(cfg *config.Config, saveConfig func() error, agentClient *client.AgentClient, folderService *services.FolderService) []*cobra.Command {
	var cmds []*cobra.Command

	// Add folder command
	addCmd := &cobra.Command{
		Use:   "add-folder [path]",
		Short: "Add a folder to sync",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			folderName, _ := cmd.Flags().GetString("name")
			priority, _ := cmd.Flags().GetInt("priority")
			twoWay, _ := cmd.Flags().GetBool("two-way")

			// Check if the folder exists
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("cannot access folder %s: %w", path, err)
			}
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", path)
			}

			// Get absolute path
			absPath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %w", err)
			}

			// If no name is provided, use the folder name
			if folderName == "" {
				folderName = filepath.Base(absPath)
			}

			// Create folder in database
			// In a real app, we'd get the current user's ID
			folder, err := folderService.CreateFolder(1, folderName, absPath, false, priority, twoWay)
			if err != nil {
				return fmt.Errorf("failed to create folder in database: %w", err)
			}

			// Save the configuration
			if err := saveConfig(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Printf("Folder added to sync list: %s\n", absPath)
			fmt.Printf("Folder ID: %s\n", folder.FolderID)
			fmt.Println("The agent will sync this folder when it's running.")
			return nil
		},
	}

	addCmd.Flags().StringP("name", "n", "", "Folder name")
	addCmd.Flags().IntP("priority", "p", 1, "Sync priority (lower numbers are higher priority)")
	addCmd.Flags().BoolP("two-way", "t", false, "Enable two-way sync (changes on remote will be downloaded)")

	cmds = append(cmds, addCmd)

	// List folders command
	listFoldersCmd := &cobra.Command{
		Use:   "list-folders",
		Short: "List all synchronized folders",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(cfg.SyncFolders) == 0 {
				fmt.Println("No folders configured for synchronization.")
				return nil
			}

			// Print as a table
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Path", "Status", "Exclude Patterns"})

			for _, folder := range cfg.SyncFolders {
				status := "Enabled"
				if !folder.Enabled {
					status = "Disabled"
				}
				excludes := "-"
				if len(folder.Exclude) > 0 {
					excludes = strings.Join(folder.Exclude, ", ")
				}
				table.Append([]string{
					folder.ID,
					folder.Path,
					status,
					excludes,
				})
			}
			table.Render()

			return nil
		},
	}

	cmds = append(cmds, listFoldersCmd)

	// Remove folder command
	removeFolderCmd := &cobra.Command{
		Use:   "remove-folder [folder-id]",
		Short: "Remove a folder from synchronization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			folderID := args[0]

			// Find the folder
			var folderPath string
			var folderIndex = -1
			for i, folder := range cfg.SyncFolders {
				if folder.ID == folderID {
					folderPath = folder.Path
					folderIndex = i
					break
				}
			}

			if folderIndex == -1 {
				return fmt.Errorf("folder with ID %s not found", folderID)
			}

			// Remove from database too
			err := folderService.DeleteFolder(folderID)
			if err != nil {
				fmt.Printf("Warning: Failed to remove folder from database: %v\n", err)
				// Continue anyway to clean up the config
			}

			// Remove the folder from config
			cfg.SyncFolders = append(cfg.SyncFolders[:folderIndex], cfg.SyncFolders[folderIndex+1:]...)

			// Save the configuration
			if err := saveConfig(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Printf("Removed folder: %s (ID: %s)\n", folderPath, folderID)
			return nil
		},
	}

	cmds = append(cmds, removeFolderCmd)

	// Enable folder command
	enableFolderCmd := &cobra.Command{
		Use:   "enable-folder [folder-id]",
		Short: "Enable synchronization for a folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			folderID := args[0]

			// Find the folder
			var folderPath string
			found := false
			for i := range cfg.SyncFolders {
				if cfg.SyncFolders[i].ID == folderID {
					folderPath = cfg.SyncFolders[i].Path
					cfg.SyncFolders[i].Enabled = true
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("folder with ID %s not found", folderID)
			}

			// Update in database too
			err := folderService.UpdateFolderStatus(folderID, true)
			if err != nil {
				fmt.Printf("Warning: Failed to update folder status in database: %v\n", err)
				// Continue anyway to update the config
			}

			// Save the configuration
			if err := saveConfig(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Printf("Enabled synchronization for folder: %s (ID: %s)\n", folderPath, folderID)
			return nil
		},
	}

	cmds = append(cmds, enableFolderCmd)

	// Disable folder command
	disableFolderCmd := &cobra.Command{
		Use:   "disable-folder [folder-id]",
		Short: "Disable synchronization for a folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			folderID := args[0]

			// Find the folder
			var folderPath string
			found := false
			for i := range cfg.SyncFolders {
				if cfg.SyncFolders[i].ID == folderID {
					folderPath = cfg.SyncFolders[i].Path
					cfg.SyncFolders[i].Enabled = false
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("folder with ID %s not found", folderID)
			}

			// Update in database too
			err := folderService.UpdateFolderStatus(folderID, false)
			if err != nil {
				fmt.Printf("Warning: Failed to update folder status in database: %v\n", err)
				// Continue anyway to update the config
			}

			// Save the configuration
			if err := saveConfig(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Printf("Disabled synchronization for folder: %s (ID: %s)\n", folderPath, folderID)
			return nil
		},
	}

	cmds = append(cmds, disableFolderCmd)

	// Configure folder command
	configureFolderCmd := &cobra.Command{
		Use:   "configure-folder [folder-id]",
		Short: "Configure synchronization settings for a folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			folderID := args[0]

			// Find the folder
			var folderIndex = -1
			for i, folder := range cfg.SyncFolders {
				if folder.ID == folderID {
					folderIndex = i
					break
				}
			}

			if folderIndex == -1 {
				return fmt.Errorf("folder with ID %s not found", folderID)
			}

			// Get the flags
			name, _ := cmd.Flags().GetString("name")
			twoWay, _ := cmd.Flags().GetBool("two-way")
			priority, _ := cmd.Flags().GetInt("priority")
			excludePattern, _ := cmd.Flags().GetStringArray("exclude")

			// Update the folder configuration
			if name != "" {
				// Update the name in the database too
				status := "active"
				if !cfg.SyncFolders[folderIndex].Enabled {
					status = "disabled"
				}
				err := folderService.UpdateFolder(folderID, name, status, false)
				if err != nil {
					fmt.Printf("Warning: Failed to update folder name in database: %v\n", err)
				}
			}

			if cmd.Flags().Changed("two-way") {
				cfg.SyncFolders[folderIndex].TwoWaySync = twoWay
			}

			if cmd.Flags().Changed("priority") {
				cfg.SyncFolders[folderIndex].Priority = priority
			}

			if len(excludePattern) > 0 {
				cfg.SyncFolders[folderIndex].Exclude = excludePattern
			}

			// Save the configuration
			if err := saveConfig(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Printf("Updated configuration for folder: %s (ID: %s)\n", cfg.SyncFolders[folderIndex].Path, folderID)
			return nil
		},
	}

	configureFolderCmd.Flags().StringP("name", "n", "", "Folder name")
	configureFolderCmd.Flags().BoolP("two-way", "t", false, "Enable two-way sync (changes on remote will be downloaded)")
	configureFolderCmd.Flags().IntP("priority", "p", 0, "Sync priority (lower numbers are higher priority)")
	configureFolderCmd.Flags().StringArrayP("exclude", "e", nil, "Exclude pattern (can be specified multiple times)")

	cmds = append(cmds, configureFolderCmd)

	return cmds
}

// generateFolderID generates a unique folder ID
// This would be a more robust implementation in a real scenario
func generateFolderID() string {
	return fmt.Sprintf("folder_%d", len(time.Now().String()))
}
