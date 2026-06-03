package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&models.User{},
		&models.Workspace{},
		&models.WorkspaceMember{},
		&models.Issue{},
		&models.Document{},
		&models.Comment{},
		&models.IssueReviewer{},
		&models.DocumentApproval{},
	)
	require.NoError(t, err)
	return db
}

// --- User Repository ---

func TestGormUserRepo_CreateAndFind(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormUserRepository{DB: db}

	user := &models.User{Email: "alice@test.com", Name: "Alice", PasswordHash: "hash123", Role: models.RoleEmployee}
	require.NoError(t, repo.Create(user))
	assert.NotZero(t, user.ID)

	found, err := repo.FindByEmail("alice@test.com")
	require.NoError(t, err)
	assert.Equal(t, "Alice", found.Name)

	found2, err := repo.FindByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "alice@test.com", found2.Email)
}

func TestGormUserRepo_FindByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormUserRepository{DB: db}

	_, err := repo.FindByEmail("nonexistent@test.com")
	assert.Error(t, err)
}

func TestGormUserRepo_ListAll(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormUserRepository{DB: db}

	repo.Create(&models.User{Email: "a@t.com", Name: "A", PasswordHash: "h"})
	repo.Create(&models.User{Email: "b@t.com", Name: "B", PasswordHash: "h"})

	users, err := repo.ListAll()
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestGormUserRepo_UpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormUserRepository{DB: db}

	user := &models.User{Email: "alice@test.com", Name: "Alice", PasswordHash: "oldhash"}
	repo.Create(user)

	err := repo.UpdatePassword(user.ID, "newhash")
	require.NoError(t, err)

	found, _ := repo.FindByID(user.ID)
	assert.Equal(t, "newhash", found.PasswordHash)
}

// --- Workspace Repository ---

func TestGormWorkspaceRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	wsRepo := &GormWorkspaceRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	manager := &models.User{Email: "mgr@test.com", Name: "Manager", PasswordHash: "h", Role: models.RoleManager}
	userRepo.Create(manager)

	ws := &models.Workspace{Name: "Test WS", ManagerID: manager.ID}
	require.NoError(t, wsRepo.Create(ws))
	assert.NotZero(t, ws.ID)

	found, err := wsRepo.FindByID(ws.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test WS", found.Name)

	found.Name = "Updated WS"
	require.NoError(t, wsRepo.Update(found))

	found2, _ := wsRepo.FindByID(ws.ID)
	assert.Equal(t, "Updated WS", found2.Name)

	workspaces, total, err := wsRepo.FindByManager(manager.ID, "", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, workspaces, 1)
}

func TestGormWorkspaceRepo_SearchByName(t *testing.T) {
	db := setupTestDB(t)
	wsRepo := &GormWorkspaceRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	manager := &models.User{Email: "mgr2@test.com", Name: "Mgr", PasswordHash: "h", Role: models.RoleManager}
	require.NoError(t, userRepo.Create(manager))

	for _, name := range []string{"Alpha Project", "Beta Plan", "Gamma Initiative"} {
		require.NoError(t, wsRepo.Create(&models.Workspace{Name: name, ManagerID: manager.ID}))
	}

	got, total, err := wsRepo.FindByManager(manager.ID, "ALPHA", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, got, 1)
	assert.Equal(t, "Alpha Project", got[0].Name)

	_, total, err = wsRepo.FindByManager(manager.ID, "a P", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total) // "Alpha Project" + "Beta Plan"

	_, total, err = wsRepo.FindByManager(manager.ID, "zzz", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)

	_, total, err = wsRepo.FindByManager(manager.ID, "", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
}

func TestGormWorkspaceRepo_FilterByStatus(t *testing.T) {
	db := setupTestDB(t)
	wsRepo := &GormWorkspaceRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	manager := &models.User{Email: "mgr-status@t.com", Name: "Mgr", PasswordHash: "h", Role: models.RoleManager}
	require.NoError(t, userRepo.Create(manager))

	for _, name := range []string{"A1", "A2"} {
		require.NoError(t, wsRepo.Create(&models.Workspace{Name: name, ManagerID: manager.ID, Status: models.WorkspaceStatusActive}))
	}
	require.NoError(t, wsRepo.Create(&models.Workspace{Name: "Z1", ManagerID: manager.ID, Status: models.WorkspaceStatusArchived}))

	_, total, err := wsRepo.FindByManager(manager.ID, "", models.WorkspaceStatusActive, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	_, total, err = wsRepo.FindByManager(manager.ID, "", models.WorkspaceStatusArchived, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	_, total, err = wsRepo.FindByManager(manager.ID, "", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
}

func TestGormWorkspaceRepo_SearchByMember(t *testing.T) {
	db := setupTestDB(t)
	wsRepo := &GormWorkspaceRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	manager := &models.User{Email: "mgr3@test.com", Name: "Mgr", PasswordHash: "h", Role: models.RoleManager}
	require.NoError(t, userRepo.Create(manager))
	member := &models.User{Email: "mem@test.com", Name: "Mem", PasswordHash: "h", Role: models.RoleEmployee}
	require.NoError(t, userRepo.Create(member))

	for _, name := range []string{"Alpha Project", "Beta Plan"} {
		ws := &models.Workspace{Name: name, ManagerID: manager.ID}
		require.NoError(t, wsRepo.Create(ws))
		require.NoError(t, wsRepo.AddMember(&models.WorkspaceMember{WorkspaceID: ws.ID, UserID: member.ID}))
	}

	got, total, err := wsRepo.FindByMember(member.ID, "alpha", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, got, 1)
	assert.Equal(t, "Alpha Project", got[0].Name)
}

func TestGormWorkspaceRepo_Members(t *testing.T) {
	db := setupTestDB(t)
	wsRepo := &GormWorkspaceRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	manager := &models.User{Email: "mgr@t.com", Name: "Mgr", PasswordHash: "h", Role: models.RoleManager}
	userRepo.Create(manager)
	employee := &models.User{Email: "emp@t.com", Name: "Emp", PasswordHash: "h", Role: models.RoleEmployee}
	userRepo.Create(employee)

	ws := &models.Workspace{Name: "WS", ManagerID: manager.ID}
	wsRepo.Create(ws)

	member := &models.WorkspaceMember{WorkspaceID: ws.ID, UserID: employee.ID}
	require.NoError(t, wsRepo.AddMember(member))

	members, err := wsRepo.GetMembers(ws.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, "Emp", members[0].Name)

	workspaces, total, err := wsRepo.FindByMember(employee.ID, "", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, workspaces, 1)

	require.NoError(t, wsRepo.RemoveMember(ws.ID, employee.ID))

	members2, _ := wsRepo.GetMembers(ws.ID)
	assert.Len(t, members2, 0)
}

func TestGormWorkspaceRepo_AssignableUsers(t *testing.T) {
	db := setupTestDB(t)
	wsRepo := &GormWorkspaceRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	manager := &models.User{Email: "mgr@t.com", Name: "Mgr", PasswordHash: "h", Role: models.RoleManager}
	userRepo.Create(manager)
	emp1 := &models.User{Email: "e1@t.com", Name: "Emp1", PasswordHash: "h", Role: models.RoleEmployee}
	userRepo.Create(emp1)
	emp2 := &models.User{Email: "e2@t.com", Name: "Emp2", PasswordHash: "h", Role: models.RoleEmployee}
	userRepo.Create(emp2)

	ws := &models.Workspace{Name: "WS", ManagerID: manager.ID}
	wsRepo.Create(ws)

	// Add emp1 as member - emp2 should be assignable
	wsRepo.AddMember(&models.WorkspaceMember{WorkspaceID: ws.ID, UserID: emp1.ID})

	assignable, err := wsRepo.GetAssignableUsers(ws.ID)
	require.NoError(t, err)
	// emp2 should be assignable, emp1 already a member, manager excluded
	assert.Len(t, assignable, 1)
	assert.Equal(t, "Emp2", assignable[0].Name)
}

func TestGormWorkspaceRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	wsRepo := &GormWorkspaceRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	manager := &models.User{Email: "m@t.com", Name: "M", PasswordHash: "h", Role: models.RoleManager}
	userRepo.Create(manager)

	ws := &models.Workspace{Name: "ToDelete", ManagerID: manager.ID}
	wsRepo.Create(ws)

	require.NoError(t, wsRepo.Delete(ws.ID))

	_, err := wsRepo.FindByID(ws.ID)
	assert.Error(t, err)
}

// --- Document Repository ---

func TestGormDocumentRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	docRepo := &GormDocumentRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	user := &models.User{Email: "u@t.com", Name: "U", PasswordHash: "h"}
	userRepo.Create(user)

	issue := &models.Issue{Title: "Issue1", CreatorID: user.ID, AssigneeID: user.ID, WorkspaceID: 1}
	db.Create(issue)

	doc := &models.Document{
		Name:       "test.pdf",
		MimeType:   "application/pdf",
		Size:       1024,
		Hash:       "abc123",
		Status:     models.DocumentStatusPending,
		UploaderID: user.ID,
		IssueID:    issue.ID,
	}
	require.NoError(t, docRepo.Create(doc))
	assert.NotZero(t, doc.ID)

	found, err := docRepo.FindByID(doc.ID)
	require.NoError(t, err)
	assert.Equal(t, "test.pdf", found.Name)

	foundByHash, err := docRepo.FindByHash("abc123")
	require.NoError(t, err)
	assert.Equal(t, doc.ID, foundByHash.ID)

	doc.Status = models.DocumentStatusInReview
	require.NoError(t, docRepo.Update(doc))

	docs, total, err := docRepo.ListByIssue(issue.ID, "", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, docs, 1)

	// Note: name search uses ILIKE which is PostgreSQL-specific, skip in SQLite tests

	// Filter by status
	docs3, total3, err := docRepo.ListByIssue(issue.ID, "", "in_review", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total3)
	assert.Len(t, docs3, 1)
}

func TestGormDocumentRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	docRepo := &GormDocumentRepository{DB: db}

	doc := &models.Document{Name: "del.pdf", Hash: "h", Status: models.DocumentStatusPending, UploaderID: 1, IssueID: 1}
	docRepo.Create(doc)

	require.NoError(t, docRepo.Delete(doc.ID))

	_, err := docRepo.FindByID(doc.ID)
	assert.Error(t, err)
}

// --- Issue Repository ---

func TestGormIssueRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	issueRepo := &GormIssueRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	user := &models.User{Email: "u@t.com", Name: "U", PasswordHash: "h"}
	userRepo.Create(user)

	issue := &models.Issue{Title: "Test Issue", Description: "Desc", CreatorID: user.ID, AssigneeID: user.ID, WorkspaceID: 1}
	require.NoError(t, issueRepo.Create(issue))
	assert.NotZero(t, issue.ID)

	found, err := issueRepo.FindByID(issue.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Issue", found.Title)

	issues, err := issueRepo.ListByWorkspace(1, "", nil, false)
	require.NoError(t, err)
	assert.Len(t, issues, 1)

	allIssues, err := issueRepo.ListAll()
	require.NoError(t, err)
	assert.Len(t, allIssues, 1)
}

func TestGormIssueRepo_SetGetReviewers(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormIssueRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	for _, n := range []string{"A", "B", "C"} {
		require.NoError(t, userRepo.Create(&models.User{Email: n + "@t.com", Name: n, PasswordHash: "h"}))
	}
	issue := &models.Issue{Title: "I", CreatorID: 1, AssigneeID: 1, WorkspaceID: 1}
	require.NoError(t, repo.Create(issue))

	require.NoError(t, repo.SetReviewers(issue.ID, []uint{1, 2}))
	revs, err := repo.GetReviewers(issue.ID)
	require.NoError(t, err)
	assert.Len(t, revs, 2)

	// Replace-set: {2,3} drops reviewer 1 and adds 3.
	require.NoError(t, repo.SetReviewers(issue.ID, []uint{2, 3}))
	revs2, err := repo.GetReviewers(issue.ID)
	require.NoError(t, err)
	ids := map[uint]bool{}
	for _, u := range revs2 {
		ids[u.ID] = true
	}
	assert.True(t, ids[2] && ids[3] && !ids[1])

	// FindByID preloads the panel (with the related users).
	found, err := repo.FindByID(issue.ID)
	require.NoError(t, err)
	assert.Len(t, found.Reviewers, 2)
}

// --- Document Approval Repository ---

func TestGormDocumentApprovalRepo_IdempotentAndCount(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormDocumentApprovalRepository{DB: db}

	require.NoError(t, repo.Create(&models.DocumentApproval{DocumentID: 42, ReviewerID: 5}))
	// Duplicate approval by the same reviewer is a no-op, not an error.
	require.NoError(t, repo.Create(&models.DocumentApproval{DocumentID: 42, ReviewerID: 5}))
	require.NoError(t, repo.Create(&models.DocumentApproval{DocumentID: 42, ReviewerID: 6}))

	all, err := repo.ListByDocument(42)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	n, err := repo.CountByReviewers(42, []uint{5, 6, 7})
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	// A reviewer no longer on the panel stops counting.
	n2, err := repo.CountByReviewers(42, []uint{6})
	require.NoError(t, err)
	assert.Equal(t, int64(1), n2)

	// DeleteByDocument clears the tally (used on resubmit).
	require.NoError(t, repo.DeleteByDocument(42))
	n3, err := repo.CountByReviewers(42, []uint{5, 6, 7})
	require.NoError(t, err)
	assert.Equal(t, int64(0), n3)
}

func TestGormIssueRepo_FilterByStatus(t *testing.T) {
	db := setupTestDB(t)
	issueRepo := &GormIssueRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	user := &models.User{Email: "filter@t.com", Name: "F", PasswordHash: "h"}
	userRepo.Create(user)

	open1 := &models.Issue{Title: "Open One", CreatorID: user.ID, AssigneeID: user.ID, WorkspaceID: 9}
	open2 := &models.Issue{Title: "Open Two", CreatorID: user.ID, AssigneeID: user.ID, WorkspaceID: 9}
	res := &models.Issue{Title: "Resolved One", CreatorID: user.ID, AssigneeID: user.ID, WorkspaceID: 9, Resolved: true}
	require.NoError(t, issueRepo.Create(open1))
	require.NoError(t, issueRepo.Create(open2))
	require.NoError(t, issueRepo.Create(res))

	falseVal, trueVal := false, true

	all, err := issueRepo.ListByWorkspace(9, "", nil, false)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	openIssues, err := issueRepo.ListByWorkspace(9, "", &falseVal, false)
	require.NoError(t, err)
	assert.Len(t, openIssues, 2)

	resolvedIssues, err := issueRepo.ListByWorkspace(9, "", &trueVal, false)
	require.NoError(t, err)
	assert.Len(t, resolvedIssues, 1)

	searched, err := issueRepo.ListByWorkspace(9, "two", nil, false)
	require.NoError(t, err)
	require.Len(t, searched, 1)
	assert.Equal(t, "Open Two", searched[0].Title)
}

func TestGormIssueRepo_FilterByArchived(t *testing.T) {
	db := setupTestDB(t)
	issueRepo := &GormIssueRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	user := &models.User{Email: "arch@t.com", Name: "A", PasswordHash: "h"}
	userRepo.Create(user)

	active1 := &models.Issue{Title: "Active One", CreatorID: user.ID, AssigneeID: user.ID, WorkspaceID: 10}
	active2 := &models.Issue{Title: "Active Two", CreatorID: user.ID, AssigneeID: user.ID, WorkspaceID: 10}
	archived := &models.Issue{Title: "Archived One", CreatorID: user.ID, AssigneeID: user.ID, WorkspaceID: 10, Archived: true}
	require.NoError(t, issueRepo.Create(active1))
	require.NoError(t, issueRepo.Create(active2))
	require.NoError(t, issueRepo.Create(archived))

	active, err := issueRepo.ListByWorkspace(10, "", nil, false)
	require.NoError(t, err)
	assert.Len(t, active, 2)

	archivedList, err := issueRepo.ListByWorkspace(10, "", nil, true)
	require.NoError(t, err)
	assert.Len(t, archivedList, 1)
	assert.Equal(t, "Archived One", archivedList[0].Title)
}

// --- Comment Repository ---

func TestGormCommentRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	commentRepo := &GormCommentRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	user := &models.User{Email: "u@t.com", Name: "U", PasswordHash: "h"}
	userRepo.Create(user)

	comment := &models.Comment{DocumentID: 1, AuthorID: user.ID, Body: "Test comment"}
	require.NoError(t, commentRepo.Create(comment))
	assert.NotZero(t, comment.ID)

	found, err := commentRepo.FindByID(comment.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test comment", found.Body)

	comments, err := commentRepo.ListByDocument(1)
	require.NoError(t, err)
	assert.Len(t, comments, 1)
}

func TestGormDocumentRepo_ListByIssue_ApprovedFirst(t *testing.T) {
	db := setupTestDB(t)
	docRepo := &GormDocumentRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	user := &models.User{Email: "u@t.com", Name: "U", PasswordHash: "h"}
	require.NoError(t, userRepo.Create(user))

	issue := &models.Issue{Title: "I", CreatorID: user.ID, AssigneeID: user.ID, WorkspaceID: 1}
	require.NoError(t, db.Create(issue).Error)

	// Approved doc was uploaded *first* (oldest CreatedAt). Pending docs come after.
	approved := &models.Document{
		Name: "approved.pdf", MimeType: "application/pdf", Size: 1, Hash: "h-approved",
		Status: models.DocumentStatusApproved, UploaderID: user.ID, IssueID: issue.ID,
	}
	require.NoError(t, docRepo.Create(approved))
	// Force the approved doc to be older than the pending ones.
	require.NoError(t, db.Model(approved).Update("created_at", time.Now().Add(-1*time.Hour)).Error)

	pending1 := &models.Document{
		Name: "pending-1.pdf", MimeType: "application/pdf", Size: 1, Hash: "h-pending-1",
		Status: models.DocumentStatusPending, UploaderID: user.ID, IssueID: issue.ID,
	}
	require.NoError(t, docRepo.Create(pending1))
	pending2 := &models.Document{
		Name: "pending-2.pdf", MimeType: "application/pdf", Size: 1, Hash: "h-pending-2",
		Status: models.DocumentStatusPending, UploaderID: user.ID, IssueID: issue.ID,
	}
	require.NoError(t, docRepo.Create(pending2))

	docs, total, err := docRepo.ListByIssue(issue.ID, "", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, docs, 3)
	// Approved must be first regardless of upload time.
	assert.Equal(t, models.DocumentStatusApproved, docs[0].Status)
	assert.Equal(t, "approved.pdf", docs[0].Name)
	// Remaining two are pending, ordered by created_at DESC.
	assert.Equal(t, models.DocumentStatusPending, docs[1].Status)
	assert.Equal(t, models.DocumentStatusPending, docs[2].Status)
	assert.Equal(t, "pending-2.pdf", docs[1].Name)
	assert.Equal(t, "pending-1.pdf", docs[2].Name)
}

func TestGormIssueRepo_ListMyCurrentIssues(t *testing.T) {
	db := setupTestDB(t)
	issueRepo := &GormIssueRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	me := &models.User{Email: "me@t.com", Name: "Me", PasswordHash: "h"}
	other := &models.User{Email: "o@t.com", Name: "Other", PasswordHash: "h"}
	require.NoError(t, userRepo.Create(me))
	require.NoError(t, userRepo.Create(other))
	ws := &models.Workspace{Name: "WS", ManagerID: other.ID}
	require.NoError(t, db.Create(ws).Error)

	mk := func(title string, assignee, creator uint, resolved, archived bool) {
		require.NoError(t, db.Create(&models.Issue{
			Title: title, AssigneeID: assignee, CreatorID: creator, WorkspaceID: ws.ID,
			Resolved: resolved, Archived: archived,
		}).Error)
	}
	mk("assignee-mine", me.ID, other.ID, false, false) // included (assignee)
	mk("creator-mine", other.ID, me.ID, false, false)  // included (creator)
	mk("resolved", me.ID, other.ID, true, false)       // excluded
	mk("archived", me.ID, other.ID, false, true)       // excluded
	mk("not-mine", other.ID, other.ID, false, false)   // excluded

	got, total, err := issueRepo.ListMyCurrentIssues(me.ID, "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, got, 2)
	// Workspace + manager preloads are populated for the dashboard table.
	assert.Equal(t, "WS", got[0].Workspace.Name)
	assert.Equal(t, "Other", got[0].Workspace.Manager.Name)
}

func TestGormIssueRepo_ListUnresolvedWithDeadline(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormIssueRepository{DB: db}
	dl := time.Now().Add(24 * time.Hour)
	require.NoError(t, db.Create(&models.Issue{Title: "with-deadline", CreatorID: 1, AssigneeID: 1, WorkspaceID: 1, Deadline: &dl}).Error)
	require.NoError(t, db.Create(&models.Issue{Title: "no-deadline", CreatorID: 1, AssigneeID: 1, WorkspaceID: 1}).Error)
	require.NoError(t, db.Create(&models.Issue{Title: "resolved", CreatorID: 1, AssigneeID: 1, WorkspaceID: 1, Deadline: &dl, Resolved: true}).Error)

	got, err := repo.ListUnresolvedWithDeadline()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "with-deadline", got[0].Title)
}

func TestGormIssueRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormIssueRepository{DB: db}
	issue := &models.Issue{Title: "old", CreatorID: 1, AssigneeID: 1, WorkspaceID: 1}
	require.NoError(t, repo.Create(issue))

	issue.Title = "new"
	issue.Resolved = true
	require.NoError(t, repo.Update(issue))

	got, err := repo.FindByID(issue.ID)
	require.NoError(t, err)
	assert.Equal(t, "new", got.Title)
	assert.True(t, got.Resolved)
}

func TestGormDocumentApprovalRepo_ListByDocumentIDs(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormDocumentApprovalRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}
	u := &models.User{Email: "r@t.com", Name: "Rev", PasswordHash: "h"}
	require.NoError(t, userRepo.Create(u))

	require.NoError(t, repo.Create(&models.DocumentApproval{DocumentID: 10, ReviewerID: u.ID}))
	require.NoError(t, repo.Create(&models.DocumentApproval{DocumentID: 20, ReviewerID: u.ID}))
	require.NoError(t, repo.Create(&models.DocumentApproval{DocumentID: 30, ReviewerID: u.ID}))

	got, err := repo.ListByDocumentIDs([]uint{10, 20})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "Rev", got[0].Reviewer.Name) // preload populated

	empty, err := repo.ListByDocumentIDs(nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestGormDocumentRepo_FindByIssueAndHash(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormDocumentRepository{DB: db}
	require.NoError(t, repo.Create(&models.Document{Name: "a", MimeType: "application/pdf", Size: 1, Hash: "dup", Status: models.DocumentStatusPending, UploaderID: 1, IssueID: 1}))
	require.NoError(t, repo.Create(&models.Document{Name: "b", MimeType: "application/pdf", Size: 1, Hash: "dup", Status: models.DocumentStatusPending, UploaderID: 1, IssueID: 2}))

	got, err := repo.FindByIssueAndHash(2, "dup")
	require.NoError(t, err)
	assert.Equal(t, uint(2), got.IssueID)
	assert.Equal(t, "b", got.Name)

	_, err = repo.FindByIssueAndHash(3, "dup")
	assert.Error(t, err)
}

func TestGormCommentRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := &GormCommentRepository{DB: db}
	c := &models.Comment{DocumentID: 1, AuthorID: 1, Body: "to delete"}
	require.NoError(t, repo.Create(c))

	require.NoError(t, repo.Delete(c.ID))
	_, err := repo.FindByID(c.ID)
	assert.Error(t, err)
}
