package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/martinshumberto/sync-manager/agent/internal/storage"
	sync_manager "github.com/martinshumberto/sync-manager/agent/internal/sync"
	"github.com/martinshumberto/sync-manager/agent/internal/uploader"
	common_config "github.com/martinshumberto/sync-manager/common/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Version information (will be set during build)
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	log.Info().
		Str("version", Version).
		Str("build_time", BuildTime).
		Msg("Starting Sync Manager Agent")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		received := <-sig
		log.Info().
			Str("signal", received.String()).
			Msg("Received signal, shutting down")
		cancel()
	}()

	cfg, err := loadConfiguration()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	setLogLevel(cfg.LogLevel)

	store, err := createStorage(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize storage")
	}

	uploaderInstance := uploader.NewUploader(store, cfg)

	syncManager, err := sync_manager.NewManager(cfg, store, uploaderInstance)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create sync manager")
	}

	uploaderInstance.Start()
	if err := syncManager.Start(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start sync manager")
	}

	log.Info().Msg("Sync Manager Agent started successfully")

	fmt.Println("Sync Manager Agent")
	fmt.Println("---------------")
	fmt.Println("Agent is running in the background.")
	fmt.Println("Monitoring and syncing folders according to configuration.")
	fmt.Println("Use the CLI to manage synced folders and view status.")
	fmt.Println("Press Ctrl+C to exit.")

	<-ctx.Done()

	log.Info().Msg("Shutting down sync manager")
	syncManager.Stop()

	log.Info().Msg("Shutdown complete")
}

func loadConfiguration() (*common_config.Config, error) {
	configPath := ""

	if envPath := os.Getenv("SYNC_MANAGER_CONFIG"); envPath != "" {
		configPath = envPath
	}

	cfg, err := common_config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.DeviceID == "" {
		cfg.DeviceID = generateDeviceID()

		if cfg.DeviceName == "" {
			hostname, err := os.Hostname()
			if err == nil {
				cfg.DeviceName = hostname
			} else {
				cfg.DeviceName = "sync-manager-device"
			}
		}

		configDir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user config directory: %w", err)
		}

		savePath := filepath.Join(configDir, "sync-manager", "sync-manager.yaml")
		if err := common_config.SaveConfig(cfg, savePath); err != nil {
			log.Warn().Err(err).Msg("Failed to save configuration")
		}
	}

	return cfg, nil
}

// createStorage creates a storage implementation based on configuration
func createStorage(cfg *common_config.Config) (storage.Storage, error) {
	return storage.StorageFactory(cfg)
}

// setLogLevel sets the global log level based on configuration
func setLogLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// generateDeviceID generates a unique device ID
func generateDeviceID() string {
	return uuid.New().String()
}
