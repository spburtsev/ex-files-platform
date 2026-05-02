package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
	"github.com/spburtsev/ex-files-backend/services"
)

func auditServer(tokens *mockTokens, repo services.AuditRepository, db *gorm.DB) *handlers.Server {
	return &handlers.Server{
		UserRepo: &mockUserRepo{},
		Tokens:   tokens,
		Hasher:   stubHasher{},
		Audit:    repo,
		DB:       db,
	}
}

func TestAuditList_AppliesFiltersAndPagination(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockAuditRepo{}
	uid := uint(7)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo.On("List", mock.MatchedBy(func(f services.AuditFilter) bool {
		return f.Action == "document.uploaded" &&
			f.TargetType == "document" &&
			f.ActorID != nil && *f.ActorID == uid &&
			f.From != nil && f.From.Equal(from)
	}), 25, 0).Return([]models.AuditEntry{
		{
			ID:        1,
			CreatedAt: time.Now().UTC(),
			Action:    models.AuditActionDocumentUploaded,
			ActorID:   7,
			Actor:     models.User{Model: gormModelID(7), Name: "Alice"},
			Metadata:  datatypes.JSONMap{"name": "report.pdf"},
		},
	}, int64(101), nil)

	srv := newTestServer(t, auditServer(tokens, repo, nil))
	defer srv.Close()

	url := srv.URL + "/audit?action=document.uploaded&targetType=document&actorId=7&perPage=25&page=1&from=" +
		from.Format(time.RFC3339)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, url, nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "101", res.Header.Get("X-Total-Count"))
	assert.Equal(t, "5", res.Header.Get("X-Total-Pages"))
	var got oapi.GetAuditLogResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	require.Len(t, got.Entries, 1)
	assert.Equal(t, "document.uploaded", got.Entries[0].Action)
	assert.Equal(t, "Alice", got.Entries[0].ActorName)
	require.True(t, got.Entries[0].Metadata.IsSet())
	assert.False(t, got.Entries[0].Metadata.IsNull())
}

func TestAuditList_RequiresAuth(t *testing.T) {
	srv := newTestServer(t, auditServer(&mockTokens{}, &mockAuditRepo{}, nil))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/audit")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestAuditStats_NilDBReturns500(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)

	srv := newTestServer(t, auditServer(tokens, &mockAuditRepo{}, nil))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/audit/stats", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
}

func TestAuditStats_AggregatesFromDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Workspace{}, &models.Issue{}, &models.Document{}, &models.AuditEntry{}))

	require.NoError(t, db.Create(&models.User{Model: gormModelID(1), Email: "a@x", Name: "Alice", PasswordHash: "x"}).Error)
	require.NoError(t, db.Create(&models.AuditEntry{Action: models.AuditActionUserLoggedIn, ActorID: 1, CreatedAt: time.Now().UTC()}).Error)
	require.NoError(t, db.Create(&models.AuditEntry{Action: models.AuditActionUserLoggedIn, ActorID: 1, CreatedAt: time.Now().UTC()}).Error)
	require.NoError(t, db.Create(&models.AuditEntry{Action: models.AuditActionDocumentUploaded, ActorID: 1, CreatedAt: time.Now().UTC()}).Error)
	require.NoError(t, db.Create(&models.Document{Model: gormModelID(1), Name: "a", MimeType: "x", Size: 1, Hash: "h", Status: models.DocumentStatusPending, UploaderID: 1, IssueID: 1}).Error)

	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)

	srv := newTestServer(t, auditServer(tokens, &mockAuditRepo{}, db))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/audit/stats", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.AuditStatsResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	// 2 user.logged_in + 1 document.uploaded = 2 distinct actions
	assert.Len(t, got.ActionsByType, 2)
	// 1 pending document
	assert.Len(t, got.DocumentsByStatus, 1)
	assert.Equal(t, "pending", got.DocumentsByStatus[0].Status)
	assert.Equal(t, int64(1), got.DocumentsByStatus[0].Count)
	// One actor with 3 actions
	require.Len(t, got.TopActors, 1)
	assert.Equal(t, "Alice", got.TopActors[0].ActorName)
	assert.Equal(t, int64(3), got.TopActors[0].Count)
}
