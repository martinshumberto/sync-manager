package commands

import (
	"fmt"
	"strconv"

	"github.com/martinshumberto/sync-manager/common/config"
	"github.com/spf13/cobra"
)

// CreateConfigCommands returns the configuration-related commands
func CreateConfigCommands(cfg *config.Config, saveFn func() error) []*cobra.Command {
	// Config root command
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage application configuration",
		Long:  `View and modify application configuration settings.`,
	}

	// Config get command
	configGetCmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Display current configuration",
		Long:  `Display the current configuration. If a key is provided, only that setting is shown.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				key := args[0]
				// TODO: Implement fetching specific configuration values
				switch key {
				case "storage.provider":
					fmt.Printf("%s: %s\n", key, cfg.StorageProvider)
				case "storage.s3.bucket":
					fmt.Printf("%s: %s\n", key, cfg.S3Config.Bucket)
				case "storage.minio.bucket":
					fmt.Printf("%s: %s\n", key, cfg.MinioConfig.Bucket)
				case "storage.minio.endpoint":
					fmt.Printf("%s: %s\n", key, cfg.MinioConfig.Endpoint)
				case "storage.gcs.bucket":
					fmt.Printf("%s: %s\n", key, cfg.GCSConfig.Bucket)
				case "storage.local.root_dir":
					fmt.Printf("%s: %s\n", key, cfg.LocalConfig.RootDir)
				case "throttle.bandwidth":
					fmt.Printf("%s: %d bytes/sec\n", key, cfg.ThrottleBytes)
				default:
					fmt.Printf("Unknown configuration key: %s\n", key)
				}
				return nil
			}

			// Display entire configuration
			DisplayConfig(cfg)
			return nil
		},
	}

	// Config set command
	configSetCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Long:  `Set a specific configuration value.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			// Set the appropriate configuration
			switch key {
			case "storage.provider":
				// Verificar se o provedor é suportado
				switch value {
				case "s3", "minio", "gcs", "local":
					cfg.StorageProvider = value
				default:
					return fmt.Errorf("unsupported storage provider: %s (supported: s3, minio, gcs, local)", value)
				}
			case "storage.s3.bucket":
				cfg.S3Config.Bucket = value
			case "storage.s3.region":
				cfg.S3Config.Region = value
			case "storage.s3.endpoint":
				cfg.S3Config.Endpoint = value
			case "storage.s3.access_key":
				cfg.S3Config.AccessKey = value
			case "storage.s3.secret_key":
				cfg.S3Config.SecretKey = value
			case "storage.minio.bucket":
				cfg.MinioConfig.Bucket = value
			case "storage.minio.endpoint":
				cfg.MinioConfig.Endpoint = value
			case "storage.minio.region":
				cfg.MinioConfig.Region = value
			case "storage.minio.access_key":
				cfg.MinioConfig.AccessKey = value
			case "storage.minio.secret_key":
				cfg.MinioConfig.SecretKey = value
			case "storage.gcs.bucket":
				cfg.GCSConfig.Bucket = value
			case "storage.gcs.project_id":
				cfg.GCSConfig.ProjectID = value
			case "storage.gcs.credentials_file":
				cfg.GCSConfig.CredentialsFile = value
			case "storage.local.root_dir":
				cfg.LocalConfig.RootDir = value
			case "throttle.bandwidth":
				// This would need proper parsing for a number
				bandwidth, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid bandwidth value: %s (must be a number)", value)
				}
				cfg.ThrottleBytes = bandwidth
			default:
				return fmt.Errorf("unknown configuration key: %s", key)
			}

			// Save the configuration
			if err := saveFn(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Printf("Configuration %s set to %s\n", key, value)
			return nil
		},
	}

	// Config reset command
	configResetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset configuration to defaults",
		Long:  `Reset all configuration settings to their default values.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ask for confirmation
			fmt.Print("This will reset all configuration to default values. Continue? (y/n): ")
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Operation cancelled.")
				return nil
			}

			// Create a default configuration but keep the device ID
			deviceID := cfg.DeviceID
			deviceName := cfg.DeviceName
			*cfg = *config.DefaultConfig()
			cfg.DeviceID = deviceID
			cfg.DeviceName = deviceName

			// Save the configuration
			if err := saveFn(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Println("Configuration reset to default values.")
			return nil
		},
	}

	// Add subcommands to config command
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configResetCmd)

	return []*cobra.Command{configCmd}
}

// DisplayConfig imprime a configuração atual
func DisplayConfig(cfg *config.Config) {
	fmt.Println("Current Configuration:")
	fmt.Println("---------------------")
	fmt.Printf("Device ID: %s\n", cfg.DeviceID)
	fmt.Printf("Device Name: %s\n", cfg.DeviceName)
	fmt.Printf("Storage Provider: %s\n", cfg.StorageProvider)

	// Exibir detalhes específicos de acordo com o provedor de armazenamento
	switch cfg.StorageProvider {
	case "s3":
		fmt.Println("\nS3 Storage Configuration:")
		fmt.Printf("  Bucket: %s\n", cfg.S3Config.Bucket)
		fmt.Printf("  Region: %s\n", cfg.S3Config.Region)
		if cfg.S3Config.Endpoint != "" {
			fmt.Printf("  Endpoint: %s\n", cfg.S3Config.Endpoint)
		}
		fmt.Printf("  Path Style: %v\n", cfg.S3Config.PathStyle)
		fmt.Printf("  Use SSL: %v\n", cfg.S3Config.UseSSL)
	case "minio":
		fmt.Println("\nMinIO Storage Configuration:")
		fmt.Printf("  Endpoint: %s\n", cfg.MinioConfig.Endpoint)
		fmt.Printf("  Bucket: %s\n", cfg.MinioConfig.Bucket)
		fmt.Printf("  Region: %s\n", cfg.MinioConfig.Region)
		fmt.Printf("  Use SSL: %v\n", cfg.MinioConfig.UseSSL)
	case "gcs":
		fmt.Println("\nGoogle Cloud Storage Configuration:")
		fmt.Printf("  Project ID: %s\n", cfg.GCSConfig.ProjectID)
		fmt.Printf("  Bucket: %s\n", cfg.GCSConfig.Bucket)
		if cfg.GCSConfig.CredentialsFile != "" {
			fmt.Printf("  Credentials File: %s\n", cfg.GCSConfig.CredentialsFile)
		}
	case "local":
		fmt.Println("\nLocal Storage Configuration:")
		fmt.Printf("  Root Directory: %s\n", cfg.LocalConfig.RootDir)
	}

	fmt.Printf("\nMax Concurrency: %d\n", cfg.MaxConcurrency)
	fmt.Printf("Throttle Bandwidth: %d bytes/sec\n", cfg.ThrottleBytes)
	fmt.Printf("Sync Interval: %s\n", cfg.SyncInterval.String())
}
