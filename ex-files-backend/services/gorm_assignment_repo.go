package services

import (
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/models"
)

type GormAssignmentRepository struct {
	DB *gorm.DB
}

func (r *GormAssignmentRepository) ListAll() ([]models.Assignment, error) {
	var assignments []models.Assignment
	err := r.DB.Preload("Creator").Preload("Assignee").Find(&assignments).Error
	if err != nil {
		return nil, err
	}
	return assignments, nil
}

func (r *GormAssignmentRepository) FindByID(id uint) (*models.Assignment, error) {
	var a models.Assignment
	err := r.DB.Preload("Creator").Preload("Assignee").First(&a, id).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *GormAssignmentRepository) Create(a *models.Assignment) error {
	return r.DB.Create(a).Error
}
