package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/martinshumberto/sync-manager/common/config"
	"github.com/spf13/cobra"
)

// CreateWizardCommand returns the interactive wizard command
func CreateWizardCommand(cfg *config.Config, saveFn func() error) *cobra.Command {
	// Wizard command - interactive setup
	wizardCmd := &cobra.Command{
		Use:   "wizard",
		Short: "Interactive configuration wizard",
		Long:  `Start an interactive configuration wizard to set up sync-manager.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("===================================================")
			fmt.Println("Welcome to the Sync Manager Configuration Wizard")
			fmt.Println("===================================================")
			fmt.Println("This wizard will guide you through setting up Sync Manager.")
			fmt.Println("Press Ctrl+C at any time to exit.")
			fmt.Println()

			// In a real implementation, we would use a UI library like promptui
			// For now, we'll simulate the interaction with simple fmt.Scan calls

			// Step 1: Configure storage
			fmt.Println("Step 1: Configure Storage")
			fmt.Println("------------------------")

			// Ask for storage provider
			fmt.Println("Select storage provider:")
			fmt.Println("1. MinIO (local development)")
			fmt.Println("2. Amazon S3")
			fmt.Println("3. Google Cloud Storage")
			fmt.Println("4. Local filesystem")
			fmt.Print("Enter choice [1]: ")

			var storageChoice string
			fmt.Scanln(&storageChoice)

			if storageChoice == "" {
				storageChoice = "1"
			}

			// Set storage provider based on choice
			switch storageChoice {
			case "1":
				cfg.StorageProvider = "minio"
				fmt.Println("\nConfiguring MinIO storage:")

				fmt.Print("Enter MinIO endpoint [localhost:9000]: ")
				var endpoint string
				fmt.Scanln(&endpoint)
				if endpoint == "" {
					endpoint = "localhost:9000"
				}
				cfg.MinioConfig.Endpoint = endpoint

				fmt.Print("Enter MinIO region [us-east-1]: ")
				var region string
				fmt.Scanln(&region)
				if region == "" {
					region = "us-east-1"
				}
				cfg.MinioConfig.Region = region

				fmt.Print("Enter MinIO bucket [sync-manager]: ")
				var bucket string
				fmt.Scanln(&bucket)
				if bucket == "" {
					bucket = "sync-manager"
				}
				cfg.MinioConfig.Bucket = bucket

				fmt.Print("Enter MinIO access key [minioadmin]: ")
				var accessKey string
				fmt.Scanln(&accessKey)
				if accessKey == "" {
					accessKey = "minioadmin"
				}
				cfg.MinioConfig.AccessKey = accessKey

				fmt.Print("Enter MinIO secret key [minioadmin]: ")
				var secretKey string
				fmt.Scanln(&secretKey)
				if secretKey == "" {
					secretKey = "minioadmin"
				}
				cfg.MinioConfig.SecretKey = secretKey

				fmt.Print("Use SSL? [y/N]: ")
				var useSSL string
				fmt.Scanln(&useSSL)
				cfg.MinioConfig.UseSSL = useSSL == "y" || useSSL == "Y"

				fmt.Println("\nMinIO configuration complete!")
			case "2":
				cfg.StorageProvider = "s3"
				fmt.Println("\nConfiguring Amazon S3 storage:")

				fmt.Print("Enter AWS region [us-east-1]: ")
				var region string
				fmt.Scanln(&region)
				if region == "" {
					region = "us-east-1"
				}
				cfg.S3Config.Region = region

				fmt.Print("Enter S3 bucket name: ")
				var bucket string
				fmt.Scanln(&bucket)
				if bucket != "" {
					cfg.S3Config.Bucket = bucket
				}

				fmt.Print("Use a custom endpoint? (for compatible services) [y/N]: ")
				var customEndpoint string
				fmt.Scanln(&customEndpoint)

				if customEndpoint == "y" || customEndpoint == "Y" {
					fmt.Print("Enter endpoint URL: ")
					var endpoint string
					fmt.Scanln(&endpoint)
					cfg.S3Config.Endpoint = endpoint

					fmt.Print("Enter access key: ")
					var accessKey string
					fmt.Scanln(&accessKey)
					cfg.S3Config.AccessKey = accessKey

					fmt.Print("Enter secret key: ")
					var secretKey string
					fmt.Scanln(&secretKey)
					cfg.S3Config.SecretKey = secretKey

					fmt.Print("Use path style? [y/N]: ")
					var pathStyle string
					fmt.Scanln(&pathStyle)
					cfg.S3Config.PathStyle = pathStyle == "y" || pathStyle == "Y"
				}

				fmt.Println("\nS3 configuration complete!")
			case "3":
				cfg.StorageProvider = "gcs"
				fmt.Println("\nConfiguring Google Cloud Storage:")

				fmt.Print("Enter GCS project ID: ")
				var projectID string
				fmt.Scanln(&projectID)
				cfg.GCSConfig.ProjectID = projectID

				fmt.Print("Enter GCS bucket name: ")
				var bucket string
				fmt.Scanln(&bucket)
				cfg.GCSConfig.Bucket = bucket

				fmt.Print("Enter path to credentials file (leave empty for default credentials): ")
				var credentialsFile string
				fmt.Scanln(&credentialsFile)
				cfg.GCSConfig.CredentialsFile = credentialsFile

				fmt.Println("\nGCS configuration complete!")
			case "4":
				cfg.StorageProvider = "local"
				fmt.Println("\nConfiguring local filesystem storage:")

				// Determine default directory
				homeDir, err := os.UserHomeDir()
				defaultDir := filepath.Join(homeDir, "sync-manager-data")
				if err != nil {
					defaultDir = "./sync-manager-data"
				}

				fmt.Printf("Enter root directory [%s]: ", defaultDir)
				var rootDir string
				fmt.Scanln(&rootDir)
				if rootDir == "" {
					rootDir = defaultDir
				}
				cfg.LocalConfig.RootDir = rootDir

				// Create directory if it doesn't exist
				if _, err := os.Stat(rootDir); os.IsNotExist(err) {
					if err := os.MkdirAll(rootDir, 0755); err != nil {
						fmt.Printf("Warning: Failed to create directory: %v\n", err)
					} else {
						fmt.Printf("Created storage directory at: %s\n", rootDir)
					}
				}

				fmt.Println("\nLocal storage configuration complete!")
			default:
				fmt.Println("Invalid choice. Using MinIO as default.")
				cfg.StorageProvider = "minio"
			}

			// Step 2: Configure sync settings
			fmt.Println("\nStep 2: Configure Sync Settings")
			fmt.Println("------------------------------")

			// Sync interval
			fmt.Print("Enter sync interval in minutes [5]: ")
			var intervalStr string
			fmt.Scanln(&intervalStr)

			if intervalStr == "" {
				cfg.SyncInterval = 5 * time.Minute
			} else {
				var interval int
				fmt.Sscanf(intervalStr, "%d", &interval)
				if interval < 1 {
					interval = 5
				}
				cfg.SyncInterval = time.Duration(interval) * time.Minute
			}

			// Concurrency
			fmt.Print("Enter max concurrent transfers [4]: ")
			var concurrencyStr string
			fmt.Scanln(&concurrencyStr)

			if concurrencyStr == "" {
				cfg.MaxConcurrency = 4
			} else {
				var concurrency int
				fmt.Sscanf(concurrencyStr, "%d", &concurrency)
				if concurrency < 1 {
					concurrency = 4
				}
				cfg.MaxConcurrency = concurrency
			}

			// Bandwidth limit
			fmt.Print("Enter bandwidth limit in KB/s (0 for unlimited) [0]: ")
			var bandwidthStr string
			fmt.Scanln(&bandwidthStr)

			if bandwidthStr == "" {
				cfg.ThrottleBytes = 0
			} else {
				var bandwidth int64
				fmt.Sscanf(bandwidthStr, "%d", &bandwidth)
				cfg.ThrottleBytes = bandwidth * 1024 // Convert KB/s to bytes/s
			}

			// Step 3: Add folders
			fmt.Println("\nStep 3: Add Folders to Sync")
			fmt.Println("---------------------------")

			addMoreFolders := true
			for addMoreFolders {
				fmt.Print("Enter folder path to sync: ")
				var folderPath string
				fmt.Scanln(&folderPath)

				if folderPath == "" {
					fmt.Println("No folder path entered. Skipping folder addition.")
					addMoreFolders = false
					continue
				}

				// Expand ~ to home directory if present
				if folderPath == "~" || folderPath[:2] == "~/" {
					home, err := os.UserHomeDir()
					if err == nil {
						if folderPath == "~" {
							folderPath = home
						} else {
							folderPath = filepath.Join(home, folderPath[2:])
						}
					}
				}

				// Check if folder exists
				_, err := os.Stat(folderPath)
				if os.IsNotExist(err) {
					fmt.Printf("Folder %s does not exist. Do you want to create it? [Y/n]: ", folderPath)
					var createFolder string
					fmt.Scanln(&createFolder)

					if createFolder != "n" && createFolder != "N" {
						if err := os.MkdirAll(folderPath, 0755); err != nil {
							fmt.Printf("Failed to create folder: %v\n", err)
							continue
						}
						fmt.Println("Folder created successfully.")
					} else {
						fmt.Println("Folder creation skipped.")
						continue
					}
				}

				// Set up exclusion patterns
				fmt.Print("Enter file patterns to exclude (comma-separated, e.g. *.tmp,*.bak): ")
				var excludePatternsStr string
				fmt.Scanln(&excludePatternsStr)

				var excludePatterns []string
				if excludePatternsStr != "" {
					for _, pattern := range filepath.SplitList(excludePatternsStr) {
						if pattern != "" {
							excludePatterns = append(excludePatterns, pattern)
						}
					}
				}

				// Create folder configuration
				folderID := fmt.Sprintf("folder-%d", len(cfg.SyncFolders)+1)
				syncFolder := config.SyncFolder{
					ID:         folderID,
					Path:       folderPath,
					Enabled:    true,
					Exclude:    excludePatterns,
					TwoWaySync: true,
				}

				// Add to configuration
				cfg.SyncFolders = append(cfg.SyncFolders, syncFolder)

				fmt.Printf("Folder %s added successfully.\n", folderPath)

				// Ask if user wants to add more folders
				fmt.Print("Do you want to add another folder? [Y/n]: ")
				var addMore string
				fmt.Scanln(&addMore)

				addMoreFolders = addMore != "n" && addMore != "N"
			}

			// Save configuration
			if err := saveFn(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Println("\nConfiguration complete!")
			fmt.Println("===================================================")
			fmt.Println("Sync Manager has been successfully configured.")
			fmt.Println("You can now start the sync agent with: sync-manager start")
			fmt.Println("===================================================")

			return nil
		},
	}

	return wizardCmd
}
