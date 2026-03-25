package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	assignv1 "github.com/spburtsev/ex-files-backend/gen/assignments/v1"
	"github.com/spburtsev/ex-files-backend/handlers"
)

func serveAssignment(handler gin.HandlerFunc, method, path, routePattern string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Handle(method, routePattern, handler)
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	return w
}

func TestGetUsers(t *testing.T) {
	h := &handlers.AssignmentsHandler{}
	w := serveAssignment(h.GetUsers, http.MethodGet, "/users", "/users")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-protobuf", w.Header().Get("Content-Type"))

	var resp assignv1.GetUsersResponse
	require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Users, 4)
	assert.Equal(t, "Alex Johnson", resp.Users[0].Name)
}

func TestGetAssignments(t *testing.T) {
	h := &handlers.AssignmentsHandler{}
	w := serveAssignment(h.GetAssignments, http.MethodGet, "/assignments", "/assignments")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp assignv1.GetAssignmentsResponse
	require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Assignments, 8)
	assert.Equal(t, "Sorting Algorithms Report", resp.Assignments[0].Title)
}

func TestGetAssignment(t *testing.T) {
	h := &handlers.AssignmentsHandler{}

	t.Run("existing_id", func(t *testing.T) {
		w := serveAssignment(h.GetAssignment, http.MethodGet, "/assignments/a3", "/assignments/:id")

		assert.Equal(t, http.StatusOK, w.Code)

		var resp assignv1.GetAssignmentResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "a3", resp.Assignment.Id)
		assert.Equal(t, "Arrays & Linked Lists", resp.Assignment.Title)
		assert.Equal(t, "Maria Chen", resp.User.Name)
	})

	t.Run("unknown_id_returns_first", func(t *testing.T) {
		w := serveAssignment(h.GetAssignment, http.MethodGet, "/assignments/nonexistent", "/assignments/:id")

		assert.Equal(t, http.StatusOK, w.Code)

		var resp assignv1.GetAssignmentResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "a1", resp.Assignment.Id)
	})
}
