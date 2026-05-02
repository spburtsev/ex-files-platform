package services

import (
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/models"
)

type GormUserRepository struct {
	DB *gorm.DB
}

func (r *GormUserRepository) FindByEmail(email string) (*models.User, error) {
	var u models.User
	if err := r.DB.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *GormUserRepository) FindByID(id uint) (*models.User, error) {
	var u models.User
	if err := r.DB.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *GormUserRepository) Create(user *models.User) error {
	return r.DB.Create(user).Error
}

func (r *GormUserRepository) ListAll() ([]models.User, error) {
	var users []models.User
	if err := r.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *GormUserRepository) UpdatePassword(userID uint, passwordHash string) error {
	return r.DB.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", passwordHash).Error
}
