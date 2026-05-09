package services

import (
	"time"

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

func (r *GormIssueRepository) ListByWorkspace(workspaceID uint, search string, resolved *bool, archived bool) ([]models.Issue, error) {
	var issues []models.Issue
	q := r.DB.Preload("Creator").Preload("Assignee").
		Where("workspace_id = ?", workspaceID).
		Where("archived = ?", archived)
	if search != "" {
		q = q.Where("LOWER(title) LIKE LOWER(?)", "%"+search+"%")
	}
	if resolved != nil {
		q = q.Where("resolved = ?", *resolved)
	}
	if err := q.Order("created_at DESC").Find(&issues).Error; err != nil {
		return nil, err
	}
	return issues, nil
}

// ListMyCurrentIssues returns issues where the caller is assignee OR creator
// AND the issue is unresolved + unarchived, with optional case-insensitive
// title search and offset pagination. Workspace + Workspace.Manager are
// preloaded so the dashboard table can render workspace name + manager name
// without extra queries. Ordering: deadline-bearing issues first, soonest
// first, then by createdAt DESC for the rest.
func (r *GormIssueRepository) ListMyCurrentIssues(userID uint, search string, limit, offset int) ([]models.Issue, int64, error) {
	q := r.DB.Model(&models.Issue{}).
		Where("(assignee_id = ? OR creator_id = ?)", userID, userID).
		Where("resolved = ?", false).
		Where("archived = ?", false)
	if search != "" {
		q = q.Where("LOWER(title) LIKE LOWER(?)", "%"+search+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var issues []models.Issue
	if err := q.
		Preload("Workspace").
		Preload("Workspace.Manager").
		Preload("Assignee").
		Preload("Creator").
		Order("CASE WHEN deadline IS NULL THEN 1 ELSE 0 END, deadline ASC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&issues).Error; err != nil {
		return nil, 0, err
	}
	return issues, total, nil
}

// ListUnresolvedWithDeadline returns every still-actionable issue that has
// a deadline set. Used by the deadline scheduler to compute who needs a
// reminder; no preloads to keep the per-tick query light.
func (r *GormIssueRepository) ListUnresolvedWithDeadline() ([]models.Issue, error) {
	var issues []models.Issue
	err := r.DB.
		Where("resolved = ?", false).
		Where("archived = ?", false).
		Where("deadline IS NOT NULL").
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

// DashboardSummary aggregates everything the dashboard endpoint needs in five
// queries. Relevance is "direct involvement": the user is either the assignee
// or the creator of the issue.
//
// The "recent" list uses Postgres-specific GREATEST + correlated subqueries to
// pick the latest of issue.updated_at, child documents' updated_at, and child
// comments' created_at. SQLite (used in tests) doesn't ship GREATEST, so the
// repo test exercising this path is gated to Postgres.
func (r *GormIssueRepository) DashboardSummary(userID uint, dueSoonWindow time.Duration) (DashboardSummary, error) {
	var sum DashboardSummary
	now := time.Now()
	soon := now.Add(dueSoonWindow)

	openScope := func() *gorm.DB {
		return r.DB.Model(&models.Issue{}).
			Where("resolved = ?", false).
			Where("archived = ?", false)
	}

	if err := openScope().Where("assignee_id = ?", userID).Count(&sum.AssignedOpenCount).Error; err != nil {
		return sum, err
	}
	if err := openScope().Where("creator_id = ?", userID).Count(&sum.CreatedOpenCount).Error; err != nil {
		return sum, err
	}

	involved := r.DB.Where("(assignee_id = ? OR creator_id = ?)", userID, userID).
		Where("resolved = ?", false).
		Where("archived = ?", false)

	if err := involved.Session(&gorm.Session{}).
		Preload("Creator").Preload("Assignee").
		Where("deadline IS NOT NULL AND deadline > ? AND deadline <= ?", now, soon).
		Order("deadline ASC").
		Limit(10).
		Find(&sum.DueSoon).Error; err != nil {
		return sum, err
	}

	if err := involved.Session(&gorm.Session{}).
		Preload("Creator").Preload("Assignee").
		Where("deadline IS NOT NULL AND deadline < ?", now).
		Order("deadline ASC").
		Limit(10).
		Find(&sum.Overdue).Error; err != nil {
		return sum, err
	}

	const recentSQL = `
		SELECT i.*, GREATEST(
			i.updated_at,
			COALESCE((SELECT MAX(d.updated_at) FROM documents d
			          WHERE d.issue_id = i.id AND d.deleted_at IS NULL), 'epoch'::timestamptz),
			COALESCE((SELECT MAX(c.created_at) FROM comments c
			          JOIN documents d2 ON d2.id = c.document_id
			          WHERE d2.issue_id = i.id AND d2.deleted_at IS NULL), 'epoch'::timestamptz)
		) AS last_activity_at
		FROM issues i
		WHERE i.deleted_at IS NULL
		  AND (i.assignee_id = ? OR i.creator_id = ?)
		ORDER BY last_activity_at DESC
		LIMIT 5;
	`
	rows, err := r.DB.Raw(recentSQL, userID, userID).Rows()
	if err != nil {
		return sum, err
	}
	defer rows.Close()
	for rows.Next() {
		var iwa IssueWithActivity
		if err := r.DB.ScanRows(rows, &iwa); err != nil {
			return sum, err
		}
		sum.Recent = append(sum.Recent, iwa)
	}
	if err := rows.Err(); err != nil {
		return sum, err
	}

	// Hydrate Creator + Assignee on the recent list (the raw query doesn't
	// preload). One round-trip, simple to reason about.
	if len(sum.Recent) > 0 {
		userIDs := make(map[uint]struct{}, len(sum.Recent)*2)
		for _, iwa := range sum.Recent {
			userIDs[iwa.CreatorID] = struct{}{}
			userIDs[iwa.AssigneeID] = struct{}{}
		}
		ids := make([]uint, 0, len(userIDs))
		for id := range userIDs {
			ids = append(ids, id)
		}
		var users []models.User
		if err := r.DB.Where("id IN ?", ids).Find(&users).Error; err != nil {
			return sum, err
		}
		byID := make(map[uint]models.User, len(users))
		for _, u := range users {
			byID[u.ID] = u
		}
		for i := range sum.Recent {
			sum.Recent[i].Creator = byID[sum.Recent[i].CreatorID]
			sum.Recent[i].Assignee = byID[sum.Recent[i].AssigneeID]
		}
	}

	return sum, nil
}
