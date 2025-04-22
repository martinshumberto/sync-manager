package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/martinshumberto/sync-manager/common/models"
)

// FolderService handles folder-related database operations
type FolderService struct {
	db *sql.DB
}

// NewFolderService creates a new folder service
func NewFolderService(db *sql.DB) *FolderService {
	return &FolderService{db: db}
}

// CreateFolder creates a new folder in the database
func (s *FolderService) CreateFolder(userID int, name string, encryptionEnabled bool) (*models.FolderResponse, error) {
	// Generate a unique folder ID
	folderID := generateUUID()
	now := time.Now()

	// Insert the folder
	query := `
		INSERT INTO folders (
			user_id, folder_id, name, created_at, updated_at, 
			status, encryption_enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.Exec(
		query,
		userID, folderID, name, now, now,
		"active", encryptionEnabled,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	// Get the ID of the inserted folder
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get folder ID: %w", err)
	}

	// Return the folder data
	return &models.FolderResponse{
		ID:                uint(id),
		FolderID:          folderID,
		Name:              name,
		CreatedAt:         now,
		Status:            "active",
		EncryptionEnabled: encryptionEnabled,
	}, nil
}

// GetFolder gets a folder by ID
func (s *FolderService) GetFolder(folderID string) (*models.FolderResponse, error) {
	query := `
		SELECT id, folder_id, name, created_at, status, encryption_enabled
		FROM folders
		WHERE folder_id = ?
	`
	row := s.db.QueryRow(query, folderID)

	var folder models.FolderResponse
	var createdAt string
	err := row.Scan(
		&folder.ID,
		&folder.FolderID,
		&folder.Name,
		&createdAt,
		&folder.Status,
		&folder.EncryptionEnabled,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("folder not found: %s", folderID)
		}
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	// Parse the created_at timestamp
	folder.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return &folder, nil
}

// GetFolders gets all folders for a user
func (s *FolderService) GetFolders(userID int) ([]models.FolderResponse, error) {
	query := `
		SELECT id, folder_id, name, created_at, status, encryption_enabled
		FROM folders
		WHERE user_id = ?
		ORDER BY created_at DESC
	`
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query folders: %w", err)
	}
	defer rows.Close()

	var folders []models.FolderResponse
	for rows.Next() {
		var folder models.FolderResponse
		var createdAt string
		err := rows.Scan(
			&folder.ID,
			&folder.FolderID,
			&folder.Name,
			&createdAt,
			&folder.Status,
			&folder.EncryptionEnabled,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}

		// Parse the created_at timestamp
		folder.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}

		folders = append(folders, folder)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating folders: %w", err)
	}

	return folders, nil
}

// UpdateFolder updates a folder in the database
func (s *FolderService) UpdateFolder(folderID string, name, status string, encryptionEnabled bool) error {
	query := `
		UPDATE folders
		SET name = ?, status = ?, encryption_enabled = ?, updated_at = ?
		WHERE folder_id = ?
	`
	_, err := s.db.Exec(
		query,
		name, status, encryptionEnabled, time.Now(),
		folderID,
	)
	if err != nil {
		return fmt.Errorf("failed to update folder: %w", err)
	}

	return nil
}

// DeleteFolder deletes a folder from the database
func (s *FolderService) DeleteFolder(folderID string) error {
	query := `DELETE FROM folders WHERE folder_id = ?`
	_, err := s.db.Exec(query, folderID)
	if err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	return nil
}

// AddDeviceFolder adds a folder to a device
func (s *FolderService) AddDeviceFolder(
	deviceID int,
	folderID int,
	localPath string,
	syncDirection string,
	excludePatterns []string,
) error {
	// Convert exclude patterns to a string array
	excludePatternsSQL, err := stringArrayToSQL(excludePatterns)
	if err != nil {
		return fmt.Errorf("failed to convert exclude patterns: %w", err)
	}

	query := `
		INSERT INTO device_folders (
			device_id, folder_id, local_path, sync_enabled, 
			sync_direction, exclude_patterns, status
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.Exec(
		query,
		deviceID, folderID, localPath, true,
		syncDirection, excludePatternsSQL, "active",
	)
	if err != nil {
		return fmt.Errorf("failed to add folder to device: %w", err)
	}

	return nil
}

// Helper function to convert a string array to SQL JSON
func stringArrayToSQL(arr []string) (string, error) {
	if len(arr) == 0 {
		return "[]", nil
	}

	// For simplicity, we'll just join with commas and wrap in brackets
	// In a real implementation, you might want to use proper JSON encoding
	result := "["
	for i, s := range arr {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("\"%s\"", s)
	}
	result += "]"

	return result, nil
}

// Helper function to generate a UUID
func generateUUID() string {
	return fmt.Sprintf("fld_%d", time.Now().UnixNano())
}
