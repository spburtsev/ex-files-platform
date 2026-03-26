package services

import (
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/models"
)

type GormWorkspaceRepository struct {
	DB *gorm.DB
}

func (r *GormWorkspaceRepository) Create(workspace *models.Workspace) error {
	return r.DB.Create(workspace).Error
}

func (r *GormWorkspaceRepository) FindByID(id uint) (*models.Workspace, error) {
	var ws models.Workspace
	if err := r.DB.First(&ws, id).Error; err != nil {
		return nil, err
	}
	return &ws, nil
}

func (r *GormWorkspaceRepository) FindByManager(managerID uint, limit, offset int) ([]models.Workspace, int64, error) {
	var workspaces []models.Workspace
	var total int64

	q := r.DB.Model(&models.Workspace{}).Where("manager_id = ?", managerID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&workspaces).Error; err != nil {
		return nil, 0, err
	}
	return workspaces, total, nil
}

func (r *GormWorkspaceRepository) FindByMember(userID uint, limit, offset int) ([]models.Workspace, int64, error) {
	var workspaces []models.Workspace
	var total int64

	q := r.DB.Model(&models.Workspace{}).
		Joins("JOIN workspace_members ON workspace_members.workspace_id = workspaces.id AND workspace_members.deleted_at IS NULL").
		Where("workspace_members.user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("workspaces.created_at DESC").Limit(limit).Offset(offset).Find(&workspaces).Error; err != nil {
		return nil, 0, err
	}
	return workspaces, total, nil
}

func (r *GormWorkspaceRepository) Update(workspace *models.Workspace) error {
	return r.DB.Save(workspace).Error
}

func (r *GormWorkspaceRepository) Delete(id uint) error {
	return r.DB.Delete(&models.Workspace{}, id).Error
}

func (r *GormWorkspaceRepository) AddMember(member *models.WorkspaceMember) error {
	// Check for a soft-deleted record and restore it instead of inserting a duplicate.
	var existing models.WorkspaceMember
	err := r.DB.Unscoped().
		Where("workspace_id = ? AND user_id = ?", member.WorkspaceID, member.UserID).
		First(&existing).Error
	if err == nil && existing.DeletedAt.Valid {
		// Restore the soft-deleted row
		member.ID = existing.ID
		return r.DB.Unscoped().Model(member).Updates(map[string]any{
			"deleted_at": nil,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
	}
	return r.DB.Create(member).Error
}

func (r *GormWorkspaceRepository) RemoveMember(workspaceID, userID uint) error {
	return r.DB.
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Delete(&models.WorkspaceMember{}).Error
}

func (r *GormWorkspaceRepository) GetMembers(workspaceID uint) ([]models.User, error) {
	var users []models.User
	err := r.DB.
		Joins("JOIN workspace_members ON workspace_members.user_id = users.id AND workspace_members.deleted_at IS NULL").
		Where("workspace_members.workspace_id = ?", workspaceID).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
