package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a system user
type User struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	Email             string         `json:"email" gorm:"uniqueIndex"`
	PasswordHash      string         `json:"-"`
	Name              string         `json:"name"`
	LastLoginAt       time.Time      `json:"last_login_at"`
	Status            string         `json:"status" gorm:"default:active"`
	StorageQuota      int64          `json:"storage_quota" gorm:"default:10737418240"` // Default 10GB em bytes
	StorageUsed       int64          `json:"storage_used" gorm:"default:0"`
	VerificationToken string         `json:"-"`
	Verified          bool           `json:"verified" gorm:"default:false"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
	Devices           []Device       `json:"-" gorm:"foreignKey:UserID"`
	Folders           []Folder       `json:"-" gorm:"foreignKey:UserID"`
}

// UserPreference represents user preferences
type UserPreference struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	UserID        uint           `json:"user_id" gorm:"uniqueIndex"`
	User          User           `json:"-" gorm:"foreignKey:UserID"`
	SyncFrequency int            `json:"sync_frequency" gorm:"default:60"` // In minutes
	Theme         string         `json:"theme" gorm:"default:light"`
	Language      string         `json:"language" gorm:"default:en"`
	Notifications bool           `json:"notifications" gorm:"default:true"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// UserResponse represents the response with user information
type UserResponse struct {
	ID           uint      `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	LastLoginAt  time.Time `json:"last_login_at"`
	Status       string    `json:"status"`
	StorageQuota int64     `json:"storage_quota"`
	StorageUsed  int64     `json:"storage_used"`
	Verified     bool      `json:"verified"`
}

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
}

// UpdateUserRequest represents the request to update user information
type UpdateUserRequest struct {
	Name     string `json:"name"`
	Password string `json:"password" validate:"omitempty,min=8"`
}

// UpdateUserPreferencesRequest represents the request to update user preferences
type UpdateUserPreferencesRequest struct {
	SyncFrequency int    `json:"sync_frequency" validate:"omitempty,min=1"`
	Theme         string `json:"theme" validate:"omitempty,oneof=light dark system"`
	Language      string `json:"language" validate:"omitempty,iso639_1"`
	Notifications bool   `json:"notifications"`
}
