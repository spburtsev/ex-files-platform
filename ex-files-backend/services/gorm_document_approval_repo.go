package services

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/spburtsev/ex-files-backend/models"
)

type GormDocumentApprovalRepository struct {
	DB *gorm.DB
}

func (r *GormDocumentApprovalRepository) Create(approval *models.DocumentApproval) error {
	return r.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(approval).Error
}

func (r *GormDocumentApprovalRepository) ListByDocument(documentID uint) ([]models.DocumentApproval, error) {
	var approvals []models.DocumentApproval
	err := r.DB.Preload("Reviewer").
		Where("document_id = ?", documentID).
		Order("created_at ASC").
		Find(&approvals).Error
	if err != nil {
		return nil, err
	}
	return approvals, nil
}

func (r *GormDocumentApprovalRepository) ListByDocumentIDs(documentIDs []uint) ([]models.DocumentApproval, error) {
	if len(documentIDs) == 0 {
		return nil, nil
	}
	var approvals []models.DocumentApproval
	err := r.DB.Preload("Reviewer").
		Where("document_id IN ?", documentIDs).
		Order("created_at ASC").
		Find(&approvals).Error
	if err != nil {
		return nil, err
	}
	return approvals, nil
}

func (r *GormDocumentApprovalRepository) CountByReviewers(documentID uint, reviewerIDs []uint) (int64, error) {
	if len(reviewerIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := r.DB.Model(&models.DocumentApproval{}).
		Where("document_id = ? AND reviewer_id IN ?", documentID, reviewerIDs).
		Count(&count).Error
	return count, err
}

func (r *GormDocumentApprovalRepository) DeleteByDocument(documentID uint) error {
	return r.DB.Unscoped().Where("document_id = ?", documentID).Delete(&models.DocumentApproval{}).Error
}
