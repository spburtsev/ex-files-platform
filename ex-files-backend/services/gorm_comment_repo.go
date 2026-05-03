package services

import (
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/models"
)

type GormCommentRepository struct {
	DB *gorm.DB
}

func (r *GormCommentRepository) Create(comment *models.Comment) error {
	return r.DB.Create(comment).Error
}

func (r *GormCommentRepository) FindByID(id uint) (*models.Comment, error) {
	var comment models.Comment
	if err := r.DB.Preload("Author").First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *GormCommentRepository) ListByDocument(documentID uint) ([]models.Comment, error) {
	var comments []models.Comment
	if err := r.DB.Where("document_id = ?", documentID).
		Preload("Author").
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

func (r *GormCommentRepository) Delete(id uint) error {
	return r.DB.Delete(&models.Comment{}, id).Error
}
