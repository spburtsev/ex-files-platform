package services

import (
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/models"
)

type GormDocumentRepository struct {
	DB *gorm.DB
}

func (r *GormDocumentRepository) Create(doc *models.Document) error {
	return r.DB.Create(doc).Error
}

func (r *GormDocumentRepository) FindByID(id uint) (*models.Document, error) {
	var doc models.Document
	if err := r.DB.Preload("Uploader").Preload("Reviewer").First(&doc, id).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *GormDocumentRepository) ListByIssue(issueID uint, search, status string, limit, offset int) ([]models.Document, int64, error) {
	var docs []models.Document
	var total int64

	q := r.DB.Model(&models.Document{}).Where("issue_id = ?", issueID)

	if search != "" {
		q = q.Where("name ILIKE ?", "%"+search+"%")
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Preload("Uploader").Order("created_at DESC").Limit(limit).Offset(offset).Find(&docs).Error; err != nil {
		return nil, 0, err
	}

	return docs, total, nil
}

func (r *GormDocumentRepository) Update(doc *models.Document) error {
	return r.DB.Save(doc).Error
}

func (r *GormDocumentRepository) Delete(id uint) error {
	return r.DB.Delete(&models.Document{}, id).Error
}

func (r *GormDocumentRepository) CreateVersion(version *models.DocumentVersion) error {
	return r.DB.Create(version).Error
}

func (r *GormDocumentRepository) GetVersions(documentID uint) ([]models.DocumentVersion, error) {
	var versions []models.DocumentVersion
	if err := r.DB.Preload("Uploader").Where("document_id = ?", documentID).Order("version DESC").Find(&versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

func (r *GormDocumentRepository) GetVersion(id uint) (*models.DocumentVersion, error) {
	var version models.DocumentVersion
	if err := r.DB.First(&version, id).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *GormDocumentRepository) LatestVersionNumber(documentID uint) (int, error) {
	var maxVersion int
	err := r.DB.Model(&models.DocumentVersion{}).
		Where("document_id = ?", documentID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion).Error
	return maxVersion, err
}
