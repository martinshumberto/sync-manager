package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/martinshumberto/sync-manager/common/config"
	"github.com/spf13/cobra"
)

// CreateInitCommand returns the initialization command
func CreateInitCommand(cfg *config.Config, saveFn func() error) *cobra.Command {
	// Init command - simple initialization
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize sync-manager configuration",
		Long:  `Initialize the basic configuration for sync-manager.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Initializing sync-manager...")

			// Check if configuration looks initialized already
			if cfg.DeviceID != "" && cfg.StorageProvider != "" && cfg.S3Config.Bucket != "" {
				fmt.Println("Configuration appears to be already initialized.")
				fmt.Println("Use 'sync-manager config' commands to modify settings or 'sync-manager wizard' for a guided setup.")
				fmt.Println("To reset the configuration, use 'sync-manager config reset'.")
				return nil
			}

			// Basic configuration
			fmt.Println("Setting up basic configuration...")

			// Keep device ID and name if already set
			if cfg.DeviceID == "" || cfg.DeviceName == "" {
				fmt.Println("Device not yet registered, generating device ID...")
				// DeviceID is already generated in main.go, we just need to inform the user
			}

			// Set basic defaults
			if cfg.StorageProvider == "" {
				cfg.StorageProvider = "s3"
			}

			if cfg.S3Config.Region == "" {
				cfg.S3Config.Region = "us-east-1"
			}

			if cfg.MaxConcurrency == 0 {
				cfg.MaxConcurrency = 4
			}

			// Ask for S3 bucket if not set
			if cfg.S3Config.Bucket == "" {
				fmt.Print("Enter S3 bucket name (or press Enter to configure later): ")
				var bucket string
				fmt.Scanln(&bucket)

				if bucket != "" {
					cfg.S3Config.Bucket = bucket
				} else {
					fmt.Println("No bucket specified. You can configure it later with 'sync-manager config set storage.s3.bucket <name>'.")
				}
			}

			// Create default sync directory if desired
			fmt.Print("Create a default sync folder in your home directory? [Y/n]: ")
			var createDefault string
			fmt.Scanln(&createDefault)

			if createDefault != "n" && createDefault != "N" {
				homeDir, err := os.UserHomeDir()
				if err == nil {
					syncDir := filepath.Join(homeDir, "Sync")

					// Create directory if it doesn't exist
					if _, err := os.Stat(syncDir); os.IsNotExist(err) {
						if err := os.MkdirAll(syncDir, 0755); err != nil {
							fmt.Printf("Failed to create sync directory: %v\n", err)
						} else {
							fmt.Printf("Created sync directory at: %s\n", syncDir)

							// Add to configuration
							folderID := "default"
							syncFolder := config.SyncFolder{
								ID:         folderID,
								Path:       syncDir,
								Enabled:    true,
								Exclude:    []string{"*.tmp", "*.bak", ".DS_Store"},
								TwoWaySync: true,
							}

							cfg.SyncFolders = append(cfg.SyncFolders, syncFolder)
						}
					} else {
						fmt.Printf("Sync directory already exists at: %s\n", syncDir)

						// Add to configuration if not already there
						found := false
						for _, folder := range cfg.SyncFolders {
							if folder.Path == syncDir {
								found = true
								break
							}
						}

						if !found {
							folderID := "default"
							syncFolder := config.SyncFolder{
								ID:         folderID,
								Path:       syncDir,
								Enabled:    true,
								Exclude:    []string{"*.tmp", "*.bak", ".DS_Store"},
								TwoWaySync: true,
							}

							cfg.SyncFolders = append(cfg.SyncFolders, syncFolder)
						}
					}
				}
			}

			// Save configuration
			if err := saveFn(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Println("\nInitialization complete!")
			fmt.Println("For a more detailed setup, run 'sync-manager wizard'.")
			fmt.Println("To start the sync agent, run 'sync-manager start'.")

			return nil
		},
	}

	return initCmd
}
