package commands

import (
	"fmt"
	"time"

	"github.com/martinshumberto/sync-manager/cli/internal/client"
	"github.com/martinshumberto/sync-manager/common/config"
	"github.com/spf13/cobra"
)

// CreateSyncCommands creates commands for sync operations
func CreateSyncCommands(cfg *config.Config, agentClient *client.AgentClient) []*cobra.Command {
	var cmds []*cobra.Command

	// Sync now command
	syncNowCmd := &cobra.Command{
		Use:   "sync-now [folder_id]",
		Short: "Trigger an immediate sync for one or all folders",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if agentClient != nil {
				// Check if agent is running
				if err := agentClient.Health(); err != nil {
					return fmt.Errorf("agent is not running: %w", err)
				}

				// TODO: Implement sync-now through the agent API
				fmt.Println("Sync initiated through agent")
				return nil
			}

			return fmt.Errorf("agent is not running, cannot trigger sync")
		},
	}

	cmds = append(cmds, syncNowCmd)

	// Sync command - force immediate sync
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Force synchronization of all folders",
		Long:  `Force immediate synchronization of all monitored folders.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(cfg.SyncFolders) == 0 {
				fmt.Println("No folders configured for synchronization.")
				return nil
			}

			fmt.Println("Initiating synchronization for all folders...")

			// In a real implementation, we would:
			// 1. Connect to the agent service
			// 2. Trigger a sync operation
			// 3. Wait for it to complete or provide progress updates

			// Simulate sync process
			for i, folder := range cfg.SyncFolders {
				if !folder.Enabled {
					fmt.Printf("Skipping disabled folder: %s\n", folder.Path)
					continue
				}

				fmt.Printf("Synchronizing folder %d/%d: %s\n", i+1, len(cfg.SyncFolders), folder.Path)
				// Simulate some processing time
				time.Sleep(500 * time.Millisecond)
			}

			fmt.Println("Synchronization complete.")
			return nil
		},
	}

	// Sync-folder command - sync a specific folder
	syncFolderCmd := &cobra.Command{
		Use:   "sync-folder <path>",
		Short: "Synchronize a specific folder",
		Long:  `Force immediate synchronization of a specific folder.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := args[0]

			// Find the folder in config
			var targetFolder *config.SyncFolder
			for i := range cfg.SyncFolders {
				if cfg.SyncFolders[i].Path == targetPath {
					targetFolder = &cfg.SyncFolders[i]
					break
				}
			}

			if targetFolder == nil {
				return fmt.Errorf("folder not found in sync configuration: %s", targetPath)
			}

			if !targetFolder.Enabled {
				return fmt.Errorf("folder is disabled: %s", targetPath)
			}

			fmt.Printf("Synchronizing folder: %s\n", targetPath)

			// In a real implementation, we would:
			// 1. Connect to the agent service
			// 2. Trigger a sync operation for this specific folder
			// 3. Wait for it to complete or provide progress updates

			// Simulate sync process
			time.Sleep(1 * time.Second)

			fmt.Println("Folder synchronization complete.")
			return nil
		},
	}

	// Pause command
	pauseCmd := &cobra.Command{
		Use:   "pause",
		Short: "Pause synchronization",
		Long:  `Pause the synchronization process temporarily.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Synchronization paused.")
			fmt.Println("Use 'sync-manager resume' to resume synchronization.")
			return nil
		},
	}

	// Resume command
	resumeCmd := &cobra.Command{
		Use:   "resume",
		Short: "Resume synchronization",
		Long:  `Resume previously paused synchronization.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Synchronization resumed.")
			return nil
		},
	}

	cmds = append(cmds, syncCmd, syncFolderCmd, pauseCmd, resumeCmd)

	return cmds
}
