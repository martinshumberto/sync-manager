package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Device represents a registered device in the system
type Device struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	UserID        uint           `json:"user_id" gorm:"index"`
	DeviceID      string         `json:"device_id" gorm:"uniqueIndex;size:36"`
	Name          string         `json:"name"`
	LastSeenAt    time.Time      `json:"last_seen_at"`
	Status        string         `json:"status" gorm:"default:active"`
	ClientVersion string         `json:"client_version"`
	Platform      string         `json:"platform"`
	OS            string         `json:"os"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	DeviceFolders []DeviceFolder `json:"-" gorm:"foreignKey:DeviceID"`
}

// DeviceToken represents an authentication token for a device
type DeviceToken struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	DeviceID  uint           `json:"device_id" gorm:"index"`
	Device    Device         `json:"-" gorm:"foreignKey:DeviceID"`
	Token     string         `json:"token" gorm:"uniqueIndex"`
	ExpiresAt time.Time      `json:"expires_at"`
	LastUsed  time.Time      `json:"last_used"`
	Revoked   bool           `json:"revoked" gorm:"default:false"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// DeviceMetadata is a JSON type for storing device-specific metadata
type DeviceMetadata map[string]interface{}

// Value implements the driver.Valuer interface for database storage
func (dm DeviceMetadata) Value() (driver.Value, error) {
	return json.Marshal(dm)
}

// Scan implements the sql.Scanner interface for database retrieval
func (dm *DeviceMetadata) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &dm)
}

// DeviceRegistrationRequest represents a request to register a new device
type DeviceRegistrationRequest struct {
	Name     string `json:"name" validate:"required"`
	Platform string `json:"platform"`
	OS       string `json:"os"`
}

// DeviceRegistrationResponse represents the response to a device registration
type DeviceRegistrationResponse struct {
	Device Device `json:"device"`
	Token  string `json:"token"`
}

// DeviceResponse represents the response with device information
type DeviceResponse struct {
	ID            uint      `json:"id"`
	DeviceID      string    `json:"device_id"`
	Name          string    `json:"name"`
	LastSeenAt    time.Time `json:"last_seen_at"`
	Status        string    `json:"status"`
	ClientVersion string    `json:"client_version"`
	Platform      string    `json:"platform"`
	OS            string    `json:"os"`
}
