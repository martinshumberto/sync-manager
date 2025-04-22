package models

import (
	"time"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Status  int       `json:"status"`
	Message string    `json:"message"`
	Error   string    `json:"error,omitempty"`
	Time    time.Time `json:"time"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(status int, message string, err error) ErrorResponse {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	return ErrorResponse{
		Status:  status,
		Message: message,
		Error:   errMsg,
		Time:    time.Now(),
	}
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Time    time.Time   `json:"time"`
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(status int, message string, data interface{}) SuccessResponse {
	return SuccessResponse{
		Status:  status,
		Message: message,
		Data:    data,
		Time:    time.Now(),
	}
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	TotalItems int         `json:"total_items"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse(items interface{}, totalItems, page, pageSize int) PaginatedResponse {
	totalPages := totalItems / pageSize
	if totalItems%pageSize > 0 {
		totalPages++
	}

	return PaginatedResponse{
		Items:      items,
		TotalItems: totalItems,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User  UserResponse     `json:"user"`
	Token ApiTokenResponse `json:"token"`
}

// Pagination represents pagination parameters
type Pagination struct {
	Page     int `json:"page" form:"page" default:"1"`
	PageSize int `json:"page_size" form:"page_size" default:"20"`
}
