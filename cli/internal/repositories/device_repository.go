package repositories

import (
	"time"

	"github.com/martinshumberto/sync-manager/common/models"
	"gorm.io/gorm"
)

// DeviceRepository gerencia operações de banco de dados relacionadas a dispositivos
type DeviceRepository struct {
	db *gorm.DB
}

// NewDeviceRepository cria um novo repositório de dispositivos
func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// Create cria um novo dispositivo no banco de dados
func (r *DeviceRepository) Create(device *models.Device) error {
	return r.db.Create(device).Error
}

// FindByID busca um dispositivo pelo ID
func (r *DeviceRepository) FindByID(id uint) (*models.Device, error) {
	var device models.Device
	err := r.db.First(&device, id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// FindByDeviceID busca um dispositivo pelo DeviceID
func (r *DeviceRepository) FindByDeviceID(deviceID string) (*models.Device, error) {
	var device models.Device
	err := r.db.Where("device_id = ?", deviceID).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// FindByUserID busca todos os dispositivos de um usuário
func (r *DeviceRepository) FindByUserID(userID uint) ([]models.Device, error) {
	var devices []models.Device
	err := r.db.Where("user_id = ?", userID).Find(&devices).Error
	if err != nil {
		return nil, err
	}
	return devices, nil
}

// Update atualiza um dispositivo no banco de dados
func (r *DeviceRepository) Update(device *models.Device) error {
	return r.db.Save(device).Error
}

// UpdateLastSeen atualiza o timestamp de última visualização de um dispositivo
func (r *DeviceRepository) UpdateLastSeen(deviceID string) error {
	return r.db.Model(&models.Device{}).
		Where("device_id = ?", deviceID).
		Update("last_seen_at", time.Now()).Error
}

// Delete exclui um dispositivo do banco de dados (soft delete)
func (r *DeviceRepository) Delete(id uint) error {
	return r.db.Delete(&models.Device{}, id).Error
}

// CreateToken cria um novo token para um dispositivo
func (r *DeviceRepository) CreateToken(token *models.DeviceToken) error {
	return r.db.Create(token).Error
}

// FindTokenByValue busca um token pelo seu valor
func (r *DeviceRepository) FindTokenByValue(token string) (*models.DeviceToken, error) {
	var deviceToken models.DeviceToken
	err := r.db.Where("token = ? AND revoked = ? AND expires_at > ?", token, false, time.Now()).
		Preload("Device").
		First(&deviceToken).Error
	if err != nil {
		return nil, err
	}
	return &deviceToken, nil
}

// RevokeToken revoga um token de dispositivo
func (r *DeviceRepository) RevokeToken(tokenID uint) error {
	return r.db.Model(&models.DeviceToken{}).
		Where("id = ?", tokenID).
		Updates(map[string]interface{}{
			"revoked":   true,
			"last_used": time.Now(),
		}).Error
}

// UpdateTokenLastUsed atualiza o timestamp de último uso de um token
func (r *DeviceRepository) UpdateTokenLastUsed(tokenID uint) error {
	return r.db.Model(&models.DeviceToken{}).
		Where("id = ?", tokenID).
		Update("last_used", time.Now()).Error
}
