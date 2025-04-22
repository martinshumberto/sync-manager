package repositories

import (
	"fmt"

	"github.com/martinshumberto/sync-manager/common/models"
	"gorm.io/gorm"
)

// FolderRepository gerencia operações de banco de dados relacionadas a pastas
type FolderRepository struct {
	db *gorm.DB
}

// NewFolderRepository cria um novo repositório de pastas
func NewFolderRepository(db *gorm.DB) *FolderRepository {
	return &FolderRepository{db: db}
}

// Create cria uma nova pasta no banco de dados
func (r *FolderRepository) Create(folder *models.Folder) error {
	return r.db.Create(folder).Error
}

// FindByID busca uma pasta pelo ID
func (r *FolderRepository) FindByID(id uint) (*models.Folder, error) {
	var folder models.Folder
	err := r.db.First(&folder, id).Error
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

// FindByFolderID busca uma pasta pelo FolderID
func (r *FolderRepository) FindByFolderID(folderID string) (*models.Folder, error) {
	var folder models.Folder
	err := r.db.Where("folder_id = ?", folderID).First(&folder).Error
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

// FindByUserID busca todas as pastas de um usuário
func (r *FolderRepository) FindByUserID(userID uint) ([]models.Folder, error) {
	var folders []models.Folder
	err := r.db.Where("user_id = ?", userID).Find(&folders).Error
	if err != nil {
		return nil, err
	}
	return folders, nil
}

// Update atualiza uma pasta no banco de dados
func (r *FolderRepository) Update(folder *models.Folder) error {
	return r.db.Save(folder).Error
}

// Delete exclui uma pasta do banco de dados (soft delete)
func (r *FolderRepository) Delete(id uint) error {
	return r.db.Delete(&models.Folder{}, id).Error
}

// AddDeviceFolder adiciona uma pasta a um dispositivo
func (r *FolderRepository) AddDeviceFolder(deviceFolder *models.DeviceFolder) error {
	return r.db.Create(deviceFolder).Error
}

// FindDeviceFolders busca as pastas associadas a um dispositivo
func (r *FolderRepository) FindDeviceFolders(deviceID uint) ([]models.DeviceFolder, error) {
	var deviceFolders []models.DeviceFolder
	err := r.db.Where("device_id = ?", deviceID).Preload("Folder").Find(&deviceFolders).Error
	if err != nil {
		return nil, err
	}
	return deviceFolders, nil
}

// UpdateDeviceFolder atualiza uma pasta de dispositivo
func (r *FolderRepository) UpdateDeviceFolder(deviceFolder *models.DeviceFolder) error {
	return r.db.Save(deviceFolder).Error
}

// DeleteDeviceFolder remove uma pasta de um dispositivo
func (r *FolderRepository) DeleteDeviceFolder(deviceID uint, folderID uint) error {
	return r.db.Where("device_id = ? AND folder_id = ?", deviceID, folderID).Delete(&models.DeviceFolder{}).Error
}

// FindWithPreloads carrega uma pasta com relacionamentos
func (r *FolderRepository) FindWithPreloads(folderID string) (*models.Folder, error) {
	var folder models.Folder
	err := r.db.
		Preload("DeviceFolders").
		Where("folder_id = ?", folderID).
		First(&folder).Error
	if err != nil {
		return nil, fmt.Errorf("falha ao carregar pasta com preloads: %w", err)
	}
	return &folder, nil
}
