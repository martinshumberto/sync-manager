package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/martinshumberto/sync-manager/cli/internal/repositories"
	"github.com/martinshumberto/sync-manager/common/config"
	"github.com/martinshumberto/sync-manager/common/models"
)

// FolderService lida com a lógica de negócios relacionada a pastas
type FolderService struct {
	folderRepo *repositories.FolderRepository
	config     *config.Config
}

// NewFolderService cria um novo serviço de pasta
func NewFolderService(folderRepo *repositories.FolderRepository, config *config.Config) *FolderService {
	return &FolderService{
		folderRepo: folderRepo,
		config:     config,
	}
}

// CreateFolder cria uma nova pasta no banco de dados e na configuração
func (s *FolderService) CreateFolder(userID uint, name string, path string, encryptionEnabled bool, priority int, twoWaySync bool) (*models.Folder, error) {
	// Cria um ID único para a pasta
	folderID := uuid.New().String()

	// Cria a pasta no banco de dados
	folder := &models.Folder{
		UserID:            userID,
		FolderID:          folderID,
		Name:              name,
		Status:            "active",
		EncryptionEnabled: encryptionEnabled,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := s.folderRepo.Create(folder)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar pasta no banco de dados: %w", err)
	}

	// Adiciona a pasta à configuração
	s.config.SyncFolders = append(s.config.SyncFolders, config.SyncFolder{
		ID:         folderID,
		Path:       path,
		Enabled:    true,
		Exclude:    []string{},
		Priority:   priority,
		TwoWaySync: twoWaySync,
	})

	// Nota: A configuração precisa ser salva pelo chamador

	return folder, nil
}

// GetFolder busca uma pasta pelo ID único
func (s *FolderService) GetFolder(folderID string) (*models.Folder, error) {
	return s.folderRepo.FindByFolderID(folderID)
}

// GetUserFolders busca todas as pastas de um usuário
func (s *FolderService) GetUserFolders(userID uint) ([]models.Folder, error) {
	return s.folderRepo.FindByUserID(userID)
}

// UpdateFolder atualiza uma pasta no banco de dados e na configuração
func (s *FolderService) UpdateFolder(folderID string, name, status string, encryptionEnabled bool) error {
	// Busca a pasta primeiro
	folder, err := s.folderRepo.FindByFolderID(folderID)
	if err != nil {
		return fmt.Errorf("erro ao buscar pasta para atualização: %w", err)
	}

	// Atualiza os campos
	folder.Name = name
	folder.Status = status
	folder.EncryptionEnabled = encryptionEnabled
	folder.UpdatedAt = time.Now()

	// Salva no banco de dados
	err = s.folderRepo.Update(folder)
	if err != nil {
		return fmt.Errorf("erro ao atualizar pasta no banco de dados: %w", err)
	}

	// Atualiza na configuração
	for i, configFolder := range s.config.SyncFolders {
		if configFolder.ID == folderID {
			s.config.SyncFolders[i].Enabled = (status == "active")
			break
		}
	}

	// Nota: A configuração precisa ser salva pelo chamador

	return nil
}

// DeleteFolder remove uma pasta do banco de dados e da configuração
func (s *FolderService) DeleteFolder(folderID string) error {
	// Busca a pasta primeiro
	folder, err := s.folderRepo.FindByFolderID(folderID)
	if err != nil {
		return fmt.Errorf("erro ao buscar pasta para exclusão: %w", err)
	}

	// Remove do banco de dados
	err = s.folderRepo.Delete(folder.ID)
	if err != nil {
		return fmt.Errorf("erro ao excluir pasta do banco de dados: %w", err)
	}

	// Remove da configuração
	for i, configFolder := range s.config.SyncFolders {
		if configFolder.ID == folderID {
			s.config.SyncFolders = append(s.config.SyncFolders[:i], s.config.SyncFolders[i+1:]...)
			break
		}
	}

	// Nota: A configuração precisa ser salva pelo chamador

	return nil
}

// AssociateFolderWithDevice associa uma pasta a um dispositivo
func (s *FolderService) AssociateFolderWithDevice(deviceID uint, folderID string, localPath string, syncDirection string, excludePatterns []string) error {
	// Busca a pasta primeiro
	folder, err := s.folderRepo.FindByFolderID(folderID)
	if err != nil {
		return fmt.Errorf("erro ao buscar pasta para associação: %w", err)
	}

	// Cria a associação
	excludePatternsArray := models.StringArray(excludePatterns)
	deviceFolder := &models.DeviceFolder{
		DeviceID:        deviceID,
		FolderID:        folder.ID,
		LocalPath:       localPath,
		SyncEnabled:     true,
		SyncDirection:   syncDirection,
		ExcludePatterns: excludePatternsArray,
		Status:          "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	return s.folderRepo.AddDeviceFolder(deviceFolder)
}

// GetDeviceFolders busca todas as pastas associadas a um dispositivo
func (s *FolderService) GetDeviceFolders(deviceID uint) ([]models.DeviceFolder, error) {
	return s.folderRepo.FindDeviceFolders(deviceID)
}

// UpdateFolderStatus atualiza o status de uma pasta
func (s *FolderService) UpdateFolderStatus(folderID string, enabled bool) error {
	// Busca a pasta primeiro
	folder, err := s.folderRepo.FindByFolderID(folderID)
	if err != nil {
		return fmt.Errorf("erro ao buscar pasta para atualização de status: %w", err)
	}

	// Atualiza o status
	status := "active"
	if !enabled {
		status = "disabled"
	}

	folder.Status = status
	folder.UpdatedAt = time.Now()

	// Salva no banco de dados
	err = s.folderRepo.Update(folder)
	if err != nil {
		return fmt.Errorf("erro ao atualizar status da pasta no banco de dados: %w", err)
	}

	// Atualiza na configuração
	for i, configFolder := range s.config.SyncFolders {
		if configFolder.ID == folderID {
			s.config.SyncFolders[i].Enabled = enabled
			break
		}
	}

	// Nota: A configuração precisa ser salva pelo chamador

	return nil
}
