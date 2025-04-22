package repositories

import (
	"time"

	"github.com/martinshumberto/sync-manager/common/models"
	"gorm.io/gorm"
)

// UserRepository gerencia operações de banco de dados relacionadas a usuários
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository cria um novo repositório de usuários
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create cria um novo usuário no banco de dados
func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// FindByID busca um usuário pelo ID
func (r *UserRepository) FindByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail busca um usuário pelo email
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update atualiza um usuário no banco de dados
func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// UpdateLastLogin atualiza o timestamp do último login de um usuário
func (r *UserRepository) UpdateLastLogin(userID uint) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("last_login_at", time.Now()).Error
}

// Delete exclui um usuário do banco de dados (soft delete)
func (r *UserRepository) Delete(id uint) error {
	return r.db.Delete(&models.User{}, id).Error
}

// GetUserPreferences obtém as preferências de um usuário
func (r *UserRepository) GetUserPreferences(userID uint) (*models.UserPreference, error) {
	var preferences models.UserPreference
	err := r.db.Where("user_id = ?", userID).First(&preferences).Error
	if err != nil {
		// Se não encontrou, cria preferências padrão
		if err == gorm.ErrRecordNotFound {
			preferences = models.UserPreference{
				UserID:        userID,
				SyncFrequency: 60,
				Theme:         "light",
				Language:      "en",
				Notifications: true,
			}
			err = r.db.Create(&preferences).Error
			if err != nil {
				return nil, err
			}
			return &preferences, nil
		}
		return nil, err
	}
	return &preferences, nil
}

// UpdateUserPreferences atualiza as preferências de um usuário
func (r *UserRepository) UpdateUserPreferences(preferences *models.UserPreference) error {
	return r.db.Save(preferences).Error
}

// UpdateStorageUsed atualiza o espaço de armazenamento usado por um usuário
func (r *UserRepository) UpdateStorageUsed(userID uint, bytesUsed int64) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("storage_used", bytesUsed).Error
}

// GetWithPreloads carrega um usuário com todos os relacionamentos
func (r *UserRepository) GetWithPreloads(userID uint) (*models.User, error) {
	var user models.User
	err := r.db.
		Preload("Devices").
		Preload("Folders").
		First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
