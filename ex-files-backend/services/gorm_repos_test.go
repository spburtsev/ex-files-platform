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
		&models.AuditEntry{},
		&models.Issue{},
		&models.Document{},
		&models.DocumentVersion{},
		&models.Comment{},
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

	workspaces, total, err := wsRepo.FindByManager(manager.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, workspaces, 1)
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

	workspaces, total, err := wsRepo.FindByMember(employee.ID, 10, 0)
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

	// Add emp1 as member — emp2 should be assignable
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

func TestGormDocumentRepo_Versions(t *testing.T) {
	db := setupTestDB(t)
	docRepo := &GormDocumentRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	user := &models.User{Email: "u@t.com", Name: "U", PasswordHash: "h"}
	userRepo.Create(user)

	doc := &models.Document{Name: "doc.pdf", Hash: "h1", Status: models.DocumentStatusPending, UploaderID: user.ID, IssueID: 1}
	docRepo.Create(doc)

	v := &models.DocumentVersion{DocumentID: doc.ID, Version: 1, Hash: "vh1", Size: 100, StorageKey: "key1", UploaderID: user.ID}
	require.NoError(t, docRepo.CreateVersion(v))

	versions, err := docRepo.GetVersions(doc.ID)
	require.NoError(t, err)
	assert.Len(t, versions, 1)

	ver, err := docRepo.GetVersion(v.ID)
	require.NoError(t, err)
	assert.Equal(t, "vh1", ver.Hash)

	latest, err := docRepo.LatestVersionNumber(doc.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, latest)
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

	issues, err := issueRepo.ListByWorkspace(1)
	require.NoError(t, err)
	assert.Len(t, issues, 1)

	allIssues, err := issueRepo.ListAll()
	require.NoError(t, err)
	assert.Len(t, allIssues, 1)
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

// --- Audit Repository ---

func TestGormAuditRepo_AppendAndList(t *testing.T) {
	db := setupTestDB(t)
	auditRepo := &GormAuditRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	user := &models.User{Email: "u@t.com", Name: "U", PasswordHash: "h"}
	userRepo.Create(user)

	entry := &models.AuditEntry{
		Action:     models.AuditActionUserRegistered,
		ActorID:    user.ID,
		TargetType: "user",
	}
	require.NoError(t, auditRepo.Append(entry))

	entries, total, err := auditRepo.List(AuditFilter{}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, entries, 1)
	assert.Equal(t, models.AuditActionUserRegistered, entries[0].Action)
}

func TestGormAuditRepo_ListWithFilters(t *testing.T) {
	db := setupTestDB(t)
	auditRepo := &GormAuditRepository{DB: db}
	userRepo := &GormUserRepository{DB: db}

	user := &models.User{Email: "u@t.com", Name: "U", PasswordHash: "h"}
	userRepo.Create(user)

	auditRepo.Append(&models.AuditEntry{Action: models.AuditActionUserRegistered, ActorID: user.ID, TargetType: "user"})
	auditRepo.Append(&models.AuditEntry{Action: models.AuditActionDocumentUploaded, ActorID: user.ID, TargetType: "document"})

	// Filter by action
	entries, total, err := auditRepo.List(AuditFilter{Action: string(models.AuditActionUserRegistered)}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, entries, 1)

	// Filter by actor
	entries2, total2, err := auditRepo.List(AuditFilter{ActorID: &user.ID}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total2)
	assert.Len(t, entries2, 2)

	// Filter by target type
	entries3, total3, err := auditRepo.List(AuditFilter{TargetType: "document"}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total3)
	assert.Len(t, entries3, 1)

	// Filter by date range
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)
	entries4, total4, err := auditRepo.List(AuditFilter{From: &past, To: &future}, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total4)
	assert.Len(t, entries4, 2)
}
