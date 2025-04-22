package models

import (
	"time"

	"gorm.io/gorm"
)

// ApiToken represents an API token in the system
type ApiToken struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"index"`
	User      User           `json:"-" gorm:"foreignKey:UserID"`
	Token     string         `json:"token" gorm:"uniqueIndex;size:64"`
	Name      string         `json:"name"`
	ExpiresAt time.Time      `json:"expires_at"`
	LastUsed  time.Time      `json:"last_used"`
	Revoked   bool           `json:"revoked" gorm:"default:false"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// CreateTokenRequest represents the request to create a new API token
type CreateTokenRequest struct {
	Name      string `json:"name" validate:"required"`
	ExpiresIn int    `json:"expires_in" validate:"required,min=1"` // In days
}

// ApiTokenResponse represents the response with API token information
type ApiTokenResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Token     string    `json:"token,omitempty"` // Only included when first created
	ExpiresAt time.Time `json:"expires_at"`
	LastUsed  time.Time `json:"last_used"`
	CreatedAt time.Time `json:"created_at"`
}

// RevokeTokenRequest represents the request to revoke an API token
type RevokeTokenRequest struct {
	TokenID int `json:"token_id" validate:"required"`
}
