package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config is the main configuration struct for CloudSync
type Config struct {
	// General settings
	DeviceID   string `mapstructure:"device_id"`
	DeviceName string `mapstructure:"device_name"`
	LogLevel   string `mapstructure:"log_level"`
	LogPath    string `mapstructure:"log_path"`

	// Sync settings
	SyncInterval   time.Duration `mapstructure:"sync_interval"`
	MaxConcurrency int           `mapstructure:"max_concurrency"`
	ThrottleBytes  int64         `mapstructure:"throttle_bytes"`

	// Storage settings
	StorageProvider string      `mapstructure:"storage_provider"`
	S3Config        S3Config    `mapstructure:"s3"`
	MinioConfig     MinioConfig `mapstructure:"minio"`
	GCSConfig       GCSConfig   `mapstructure:"gcs"`
	LocalConfig     LocalConfig `mapstructure:"local"`

	// API settings
	ApiEndpoint string `mapstructure:"api_endpoint"`
	ApiToken    string `mapstructure:"api_token"`

	// Folders to sync
	SyncFolders []SyncFolder `mapstructure:"sync_folders"`
}

// S3Config holds S3-specific configuration
type S3Config struct {
	Endpoint  string `mapstructure:"endpoint"`
	Region    string `mapstructure:"region"`
	Bucket    string `mapstructure:"bucket"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	UseSSL    bool   `mapstructure:"use_ssl"`
	PathStyle bool   `mapstructure:"path_style"`
}

// MinioConfig holds MinIO-specific configuration
type MinioConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	Region    string `mapstructure:"region"`
	Bucket    string `mapstructure:"bucket"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

// GCSConfig holds Google Cloud Storage specific configuration
type GCSConfig struct {
	ProjectID       string `mapstructure:"project_id"`
	Bucket          string `mapstructure:"bucket"`
	CredentialsFile string `mapstructure:"credentials_file"`
}

// LocalConfig holds local filesystem storage configuration
type LocalConfig struct {
	RootDir string `mapstructure:"root_dir"`
}

// SyncFolder represents a folder to be synchronized
type SyncFolder struct {
	ID         string   `mapstructure:"id"`
	Path       string   `mapstructure:"path"`
	Enabled    bool     `mapstructure:"enabled"`
	Exclude    []string `mapstructure:"exclude"`
	Priority   int      `mapstructure:"priority"`
	TwoWaySync bool     `mapstructure:"two_way_sync"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		DeviceID:        "",
		DeviceName:      "",
		LogLevel:        "info",
		LogPath:         "",
		SyncInterval:    time.Minute * 5,
		MaxConcurrency:  4,
		ThrottleBytes:   0,       // no throttling by default
		StorageProvider: "minio", // Default to MinIO for development
		S3Config: S3Config{
			Region:    "us-east-1",
			UseSSL:    true,
			PathStyle: false,
		},
		MinioConfig: MinioConfig{
			Endpoint:  "localhost:9000",
			Region:    "us-east-1",
			Bucket:    "sync-manager",
			AccessKey: "minioadmin",
			SecretKey: "minioadmin",
			UseSSL:    false,
		},
		GCSConfig: GCSConfig{
			Bucket: "",
		},
		LocalConfig: LocalConfig{
			RootDir: "",
		},
		SyncFolders: []SyncFolder{},
	}
}

// LoadConfig loads the configuration from file
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	viper.SetConfigName("cloudsync")
	viper.SetConfigType("yaml")

	// If config path is provided, use it
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// Otherwise use default locations
		// 1. Current directory
		viper.AddConfigPath(".")

		// 2. User config directory
		userConfigDir, err := os.UserConfigDir()
		if err == nil {
			viper.AddConfigPath(filepath.Join(userConfigDir, "cloudsync"))
		}

		// 3. System config directories
		viper.AddConfigPath("/etc/cloudsync")
	}

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we'll use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// Unmarshal into our config struct
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfig saves the configuration to a file
func SaveConfig(config *Config, path string) error {
	// Set the config values in viper
	viper.Set("device_id", config.DeviceID)
	viper.Set("device_name", config.DeviceName)
	viper.Set("log_level", config.LogLevel)
	viper.Set("log_path", config.LogPath)
	viper.Set("sync_interval", config.SyncInterval)
	viper.Set("max_concurrency", config.MaxConcurrency)
	viper.Set("throttle_bytes", config.ThrottleBytes)
	viper.Set("storage_provider", config.StorageProvider)
	viper.Set("api_endpoint", config.ApiEndpoint)
	viper.Set("api_token", config.ApiToken)
	viper.Set("sync_folders", config.SyncFolders)

	// S3 config
	viper.Set("s3.endpoint", config.S3Config.Endpoint)
	viper.Set("s3.region", config.S3Config.Region)
	viper.Set("s3.bucket", config.S3Config.Bucket)
	viper.Set("s3.access_key", config.S3Config.AccessKey)
	viper.Set("s3.secret_key", config.S3Config.SecretKey)
	viper.Set("s3.use_ssl", config.S3Config.UseSSL)
	viper.Set("s3.path_style", config.S3Config.PathStyle)

	// MinIO config
	viper.Set("minio.endpoint", config.MinioConfig.Endpoint)
	viper.Set("minio.region", config.MinioConfig.Region)
	viper.Set("minio.bucket", config.MinioConfig.Bucket)
	viper.Set("minio.access_key", config.MinioConfig.AccessKey)
	viper.Set("minio.secret_key", config.MinioConfig.SecretKey)
	viper.Set("minio.use_ssl", config.MinioConfig.UseSSL)

	// GCS config
	viper.Set("gcs.project_id", config.GCSConfig.ProjectID)
	viper.Set("gcs.bucket", config.GCSConfig.Bucket)
	viper.Set("gcs.credentials_file", config.GCSConfig.CredentialsFile)

	// Local config
	viper.Set("local.root_dir", config.LocalConfig.RootDir)

	// If path is not provided, use the config file that was loaded
	if path == "" {
		path = viper.ConfigFileUsed()
	}

	// If we still don't have a path, use default
	if path == "" {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		configDir := filepath.Join(userConfigDir, "cloudsync")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}
		path = filepath.Join(configDir, "cloudsync.yaml")
	}

	// Write the config file
	return viper.WriteConfigAs(path)
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate storage provider configuration based on selected provider
	switch config.StorageProvider {
	case "s3":
		if config.S3Config.Bucket == "" {
			return fmt.Errorf("S3 bucket is required")
		}
		// Only require access key and secret key if not using AWS environment credentials
		if config.S3Config.Endpoint != "" {
			if config.S3Config.AccessKey == "" {
				return fmt.Errorf("S3 access key is required when using a custom endpoint")
			}
			if config.S3Config.SecretKey == "" {
				return fmt.Errorf("S3 secret key is required when using a custom endpoint")
			}
		}
	case "minio":
		if config.MinioConfig.Bucket == "" {
			return fmt.Errorf("MinIO bucket is required")
		}
		if config.MinioConfig.Endpoint == "" {
			return fmt.Errorf("MinIO endpoint is required")
		}
		if config.MinioConfig.AccessKey == "" {
			return fmt.Errorf("MinIO access key is required")
		}
		if config.MinioConfig.SecretKey == "" {
			return fmt.Errorf("MinIO secret key is required")
		}
	case "gcs":
		if config.GCSConfig.Bucket == "" {
			return fmt.Errorf("GCS bucket is required")
		}
		if config.GCSConfig.ProjectID == "" {
			return fmt.Errorf("GCS project ID is required")
		}
	case "local":
		if config.LocalConfig.RootDir == "" {
			return fmt.Errorf("Local storage root directory is required")
		}
	default:
		return fmt.Errorf("unsupported storage provider: %s", config.StorageProvider)
	}

	// Ensure sync interval is reasonable
	if config.SyncInterval < time.Second {
		config.SyncInterval = time.Second
	}

	// Ensure max concurrency is reasonable
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 1
	} else if config.MaxConcurrency > 32 {
		config.MaxConcurrency = 32
	}

	return nil
}

// GetConfigPath returns the default configuration path
func GetConfigPath() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(userConfigDir, "cloudsync")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(configDir, "cloudsync.yaml"), nil
}
