package services

import (
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/models"
)

type GormIssueRepository struct {
	DB *gorm.DB
}

func (r *GormIssueRepository) ListAll() ([]models.Issue, error) {
	var issues []models.Issue
	err := r.DB.Preload("Creator").Preload("Assignee").Find(&issues).Error
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func (r *GormIssueRepository) ListByWorkspace(workspaceID uint) ([]models.Issue, error) {
	var issues []models.Issue
	err := r.DB.Preload("Creator").Preload("Assignee").
		Where("workspace_id = ?", workspaceID).
		Order("created_at DESC").
		Find(&issues).Error
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func (r *GormIssueRepository) FindByID(id uint) (*models.Issue, error) {
	var issue models.Issue
	err := r.DB.Preload("Creator").Preload("Assignee").First(&issue, id).Error
	if err != nil {
		return nil, err
	}
	return &issue, nil
}

func (r *GormIssueRepository) Create(issue *models.Issue) error {
	return r.DB.Create(issue).Error
}

func (r *GormIssueRepository) Update(issue *models.Issue) error {
	return r.DB.Save(issue).Error
}
