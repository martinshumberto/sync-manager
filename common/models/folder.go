package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Folder represents a synchronization folder in the system
type Folder struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	UserID            uint           `json:"user_id" gorm:"index"`
	FolderID          string         `json:"folder_id" gorm:"uniqueIndex;size:36"`
	Name              string         `json:"name"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
	Status            string         `json:"status" gorm:"default:active"`
	EncryptionEnabled bool           `json:"encryption_enabled" gorm:"default:false"`
	EncryptionKeyID   string         `json:"encryption_key_id,omitempty"`
}

// DeviceFolder represents the mapping between a device and a folder
type DeviceFolder struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	DeviceID        uint           `json:"device_id" gorm:"index"`
	FolderID        uint           `json:"folder_id" gorm:"index"`
	Folder          Folder         `json:"-" gorm:"foreignKey:FolderID"`
	LocalPath       string         `json:"local_path"`
	SyncEnabled     bool           `json:"sync_enabled" gorm:"default:true"`
	SyncDirection   string         `json:"sync_direction" gorm:"default:bidirectional"`
	ExcludePatterns StringArray    `json:"exclude_patterns" gorm:"type:text"`
	LastSyncAt      *time.Time     `json:"last_sync_at,omitempty"`
	Status          string         `json:"status" gorm:"default:active"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// StringArray is a type for storing arrays of strings in the database
type StringArray []string

// Value implements the driver.Valuer interface for database storage
func (sa StringArray) Value() (driver.Value, error) {
	return json.Marshal(sa)
}

// Scan implements the sql.Scanner interface for database retrieval
func (sa *StringArray) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &sa)
}

// FileVersion represents a version of a file in a sync folder
type FileVersion struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	FolderID     uint           `json:"folder_id" gorm:"index"`
	Folder       Folder         `json:"-" gorm:"foreignKey:FolderID"`
	RelativePath string         `json:"relative_path"`
	VersionID    string         `json:"version_id"`
	Size         int64          `json:"size"`
	Hash         string         `json:"hash"`
	ModifiedAt   time.Time      `json:"modified_at"`
	DeviceID     uint           `json:"device_id" gorm:"index"`
	MimeType     string         `json:"mime_type,omitempty"`
	Metadata     string         `json:"metadata,omitempty" gorm:"type:text"`
	Deleted      bool           `json:"deleted" gorm:"default:false"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// SyncEvent represents a synchronization event in the system
type SyncEvent struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	DeviceID      uint           `json:"device_id" gorm:"index"`
	FolderID      uint           `json:"folder_id" gorm:"index"`
	Folder        Folder         `json:"-" gorm:"foreignKey:FolderID"`
	FileVersionID uint           `json:"file_version_id" gorm:"index"`
	FileVersion   FileVersion    `json:"-" gorm:"foreignKey:FileVersionID"`
	EventType     string         `json:"event_type"`
	RelativePath  string         `json:"relative_path"`
	Timestamp     time.Time      `json:"timestamp"`
	Details       string         `json:"details,omitempty" gorm:"type:text"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// CreateFolderRequest represents the request to create a new sync folder
type CreateFolderRequest struct {
	Name              string `json:"name" validate:"required"`
	EncryptionEnabled bool   `json:"encryption_enabled"`
}

// UpdateFolderRequest represents the request to update a sync folder
type UpdateFolderRequest struct {
	Name              string `json:"name"`
	Status            string `json:"status" validate:"omitempty,oneof=active paused disabled"`
	EncryptionEnabled bool   `json:"encryption_enabled"`
}

// FolderResponse represents the response with folder information
type FolderResponse struct {
	ID                uint      `json:"id"`
	FolderID          string    `json:"folder_id"`
	Name              string    `json:"name"`
	CreatedAt         time.Time `json:"created_at"`
	Status            string    `json:"status"`
	EncryptionEnabled bool      `json:"encryption_enabled"`
}

// AddDeviceFolderRequest represents the request to add a folder to a device
type AddDeviceFolderRequest struct {
	FolderID        uint     `json:"folder_id" validate:"required"`
	LocalPath       string   `json:"local_path" validate:"required"`
	SyncDirection   string   `json:"sync_direction" validate:"omitempty,oneof=bidirectional upload download"`
	ExcludePatterns []string `json:"exclude_patterns"`
}

// UpdateDeviceFolderRequest represents the request to update a device-folder mapping
type UpdateDeviceFolderRequest struct {
	LocalPath       string   `json:"local_path"`
	SyncEnabled     bool     `json:"sync_enabled"`
	SyncDirection   string   `json:"sync_direction" validate:"omitempty,oneof=bidirectional upload download"`
	ExcludePatterns []string `json:"exclude_patterns"`
}
