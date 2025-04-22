package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/martinshumberto/sync-manager/cli/internal/client"
	"github.com/martinshumberto/sync-manager/cli/internal/commands"
	"github.com/martinshumberto/sync-manager/cli/internal/db"
	"github.com/martinshumberto/sync-manager/cli/internal/repositories"
	"github.com/martinshumberto/sync-manager/cli/internal/services"
	"github.com/martinshumberto/sync-manager/common/config"
	"github.com/martinshumberto/sync-manager/common/models"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// Version information (will be set during build)
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Initialize logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Load configuration
	cfg, configPath, err := loadConfiguration()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Save function to be used by commands
	saveConfig := func() error {
		return config.SaveConfig(cfg, configPath)
	}

	// Initialize database
	dbPath, err := db.GetDefaultDBPath()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get database path")
	}
	dbManager, err := db.NewManager(dbPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer dbManager.Close()

	// Initialize the database schema
	if err := dbManager.InitSchema(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database schema")
	}

	// Create repositories
	folderRepo := repositories.NewFolderRepository(dbManager.GetDB())
	userRepo := repositories.NewUserRepository(dbManager.GetDB())

	// Create services
	folderService := services.NewFolderService(folderRepo, cfg)

	// Create agent client
	agentClient := client.NewAgentClient(cfg, configPath)

	// Default user ID (in a real app, we'd have proper authentication)
	defaultUserID := uint(1)

	// Ensure a user exists
	ensureDefaultUser(userRepo, defaultUserID)

	// Define the root command
	rootCmd := &cobra.Command{
		Use:     "sync-manager",
		Short:   "Sync Manager - File synchronization and backup tool",
		Version: Version,
		Long: `Sync Manager is a file synchronization and backup tool that allows you to
securely store and sync your files across multiple devices using S3-compatible storage.

It provides efficient, background synchronization with minimal resource usage.`,
	}

	// Version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Sync Manager v%s (built %s)\n", Version, BuildTime)
		},
	})

	// Add commands
	addCommands(rootCmd, cfg, configPath, saveConfig, agentClient, folderService, defaultUserID)

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// loadConfiguration loads the configuration or creates a default one
func loadConfiguration() (*config.Config, string, error) {
	// Look for configuration in common places
	configPath := ""

	// Check for config path in environment variable
	if envPath := os.Getenv("SYNC_MANAGER_CONFIG"); envPath != "" {
		configPath = envPath
	}

	// Try to load the configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load config: %w", err)
	}

	// If no config path was specified and none was found, get the default path
	if configPath == "" {
		configPath, err = config.GetConfigPath()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get default config path: %w", err)
		}
	}

	// If device ID is not set, generate one
	if cfg.DeviceID == "" {
		cfg.DeviceID = uuid.New().String()

		// Try to set a default device name
		if cfg.DeviceName == "" {
			hostname, err := os.Hostname()
			if err == nil {
				cfg.DeviceName = hostname
			} else {
				cfg.DeviceName = "sync-manager-device"
			}
		}

		// Save the configuration
		if err := config.SaveConfig(cfg, configPath); err != nil {
			log.Warn().Err(err).Msg("Failed to save configuration")
		}
	}

	return cfg, configPath, nil
}

// addCommands adiciona todos os comandos ao rootCmd
func addCommands(rootCmd *cobra.Command, cfg *config.Config, configPath string,
	saveConfig func() error, agentClient *client.AgentClient,
	folderService *services.FolderService, defaultUserID uint) {

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show sync status of monitored folders",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if agent is running
			if err := agentClient.Health(); err != nil {
				fmt.Println("Agent is not running. Start it with 'sync-manager start'.")
				return nil
			}

			// Get folders from database
			folders, err := folderService.GetUserFolders(defaultUserID)
			if err != nil {
				return fmt.Errorf("failed to get folders: %w", err)
			}

			if len(folders) == 0 {
				fmt.Println("No folders configured for synchronization.")
				return nil
			}

			fmt.Println("Synchronization Status:")
			fmt.Println("----------------------")

			// Display folder status
			for _, folder := range folders {
				status := folder.Status
				if status == "active" {
					status = "Active"
				} else {
					status = "Disabled"
				}

				fmt.Printf("📂 %s (%s)\n", folder.Name, folder.FolderID)
				fmt.Printf("   Status: %s\n", status)

				// Find matching config folder to get the path
				for _, configFolder := range cfg.SyncFolders {
					if configFolder.ID == folder.FolderID {
						fmt.Printf("   Path: %s\n", configFolder.Path)
						break
					}
				}
				fmt.Println()
			}
			return nil
		},
	}
	rootCmd.AddCommand(statusCmd)

	// Start command - starts the agent
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the sync agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			return startAgent()
		},
	}
	rootCmd.AddCommand(startCmd)

	// Stop command - stops the agent
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the sync agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopAgent()
		},
	}
	rootCmd.AddCommand(stopCmd)

	// Add folder management commands
	folderCommands := commands.CreateFolderCommands(cfg, saveConfig, agentClient, folderService)
	for _, cmd := range folderCommands {
		rootCmd.AddCommand(cmd)
	}

	// Add configuration commands
	configCommands := commands.CreateConfigCommands(cfg, saveConfig)
	for _, cmd := range configCommands {
		rootCmd.AddCommand(cmd)
	}

	// Add sync commands
	syncCommands := commands.CreateSyncCommands(cfg, agentClient)
	for _, cmd := range syncCommands {
		rootCmd.AddCommand(cmd)
	}

	// Add device commands
	deviceCommands := commands.CreateDeviceCommands(cfg)
	for _, cmd := range deviceCommands {
		rootCmd.AddCommand(cmd)
	}

	// Add wizard command
	wizardCmd := commands.CreateWizardCommand(cfg, saveConfig)
	rootCmd.AddCommand(wizardCmd)
}

// ensureDefaultUser garante que um usuário padrão existe no banco de dados
func ensureDefaultUser(userRepo *repositories.UserRepository, userID uint) {
	// Verifica se o usuário já existe
	_, err := userRepo.FindByID(userID)
	if err == nil {
		// Usuário já existe
		return
	}

	// Cria um usuário padrão
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "Default Device"
	}

	user := &models.User{
		ID:           userID,
		Email:        "user@localhost",
		Name:         "Local User",
		Status:       "active",
		Verified:     true,
		StorageQuota: 10737418240, // 10GB
	}

	err = userRepo.Create(user)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create default user")
	}
}

// startAgent starts the sync agent
func startAgent() error {
	// In a real implementation, we would use a proper method to start the agent
	// such as systemd, launchd, or a service manager.
	// For now, just simulate starting the agent.

	fmt.Println("Starting Sync Manager agent...")

	// For demo purposes, we'll just print a message
	// In a real implementation, we would:
	// 1. Check if the agent is already running
	// 2. Start the agent as a background service
	// 3. Wait for it to initialize

	fmt.Println("Agent started in the background.")
	return nil
}

// stopAgent stops the sync agent
func stopAgent() error {
	// In a real implementation, we would use a proper method to stop the agent
	// For now, just simulate stopping the agent.

	fmt.Println("Stopping Sync Manager agent...")

	// For demo purposes, we'll just print a message
	// In a real implementation, we would:
	// 1. Check if the agent is running
	// 2. Send a signal to stop it gracefully
	// 3. Wait for it to shut down

	fmt.Println("Agent stopped.")
	return nil
}

// readLine reads a line from stdin
func readLine() string {
	var input string
	fmt.Scanln(&input)
	return strings.TrimSpace(input)
}
