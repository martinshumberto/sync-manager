package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/martinshumberto/sync-manager/common/models"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Manager handles database operations for the CLI
type Manager struct {
	db *gorm.DB
}

// NewManager creates a new database manager
func NewManager(dbPath string) (*Manager, error) {
	// Create directory if it doesn't exist
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Configure GORM logger
	gormLogger := logger.Default.LogMode(logger.Silent)
	if os.Getenv("SYNC_MANAGER_DEBUG") == "true" {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	// Open database connection with GORM
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	log.Info().Str("path", dbPath).Msg("Connected to SQLite database")

	return &Manager{db: db}, nil
}

// Close closes the database connection
func (m *Manager) Close() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// InitSchema initializes the database schema using GORM auto migrate
func (m *Manager) InitSchema() error {
	err := m.db.AutoMigrate(
		&models.User{},
		&models.UserPreference{},
		&models.Device{},
		&models.DeviceToken{},
		&models.ApiToken{},
		&models.Folder{},
		&models.DeviceFolder{},
		&models.FileVersion{},
		&models.SyncEvent{},
	)

	if err != nil {
		return fmt.Errorf("failed to migrate database schema: %w", err)
	}

	log.Info().Msg("Database schema initialized successfully")
	return nil
}

// GetDB returns the underlying database connection
func (m *Manager) GetDB() *gorm.DB {
	return m.db
}

// GetDefaultDBPath returns the default path for the database file
func GetDefaultDBPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	return filepath.Join(configDir, "sync-manager", "sync-manager.db"), nil
}
