package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"strconv"

	authv1 "github.com/spburtsev/ex-files-backend/gen/auth/v1"
	issuesv1 "github.com/spburtsev/ex-files-backend/gen/issues/v1"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

type AuthHandler struct {
	Repo   services.UserRepository
	Tokens services.TokenService
	Hasher services.Hasher
	Audit  services.AuditRepository
}

type registerRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"     binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func setSessionCookie(c *gin.Context, token string) {
	slog.Debug("setting session cookie", "component", "auth", "path", c.Request.URL.Path)
	c.SetCookie("session", token, int((8 * time.Hour).Seconds()), "/", "", false, true)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slog.Debug("register attempt", "component", "auth", "email", req.Email)

	if _, err := h.Repo.FindByEmail(req.Email); err == nil {
		slog.Debug("register failed: email taken", "component", "auth", "email", req.Email)
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	hash, err := h.Hasher.Hash(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	user := models.User{
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: hash,
	}
	if err := h.Repo.Create(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	token, err := h.Tokens.Issue(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	logAudit(h.Audit, models.AuditActionUserRegistered, user.ID, uintPtr(user.ID), "user", map[string]any{
		"email": user.Email,
	})

	setSessionCookie(c, token)
	protobufResponse(c, http.StatusCreated, &authv1.RegisterResponse{
		User:  userToProto(&user),
		Token: token,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slog.Debug("login attempt", "component", "auth", "email", req.Email)

	user, err := h.Repo.FindByEmail(req.Email)
	if err != nil {
		slog.Debug("login failed: user not found", "component", "auth", "email", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := h.Hasher.Compare(user.PasswordHash, req.Password); err != nil {
		slog.Debug("login failed: wrong password", "component", "auth", "email", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := h.Tokens.Issue(user)
	if err != nil {
		slog.Error("login: failed to issue token", "component", "auth", "email", req.Email, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	slog.Debug("login succeeded", "component", "auth", "user_id", user.ID, "email", user.Email)

	logAudit(h.Audit, models.AuditActionUserLoggedIn, user.ID, uintPtr(user.ID), "user", map[string]any{
		"email": user.Email,
	})

	setSessionCookie(c, token)
	protobufResponse(c, http.StatusOK, &authv1.LoginResponse{
		User:  userToProto(user),
		Token: token,
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	user, err := h.Repo.FindByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	protobufResponse(c, http.StatusOK, &authv1.MeResponse{User: userToProto(user)})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("session", "", -1, "/", "", false, true)
	protobufResponse(c, http.StatusOK, &authv1.LogoutResponse{Message: "logged out"})
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	users, err := h.Repo.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	pb := make([]*issuesv1.User, len(users))
	for i := range users {
		u := users[i]
		role := issuesv1.Role_ROLE_EMPLOYEE
		if u.Role == models.RoleManager || u.Role == models.RoleRoot {
			role = issuesv1.Role_ROLE_MANAGER
		}
		pb[i] = &issuesv1.User{
			Id:    strconv.FormatUint(uint64(u.ID), 10),
			Name:  u.Name,
			Email: u.Email,
			Role:  role,
		}
	}
	protobufResponse(c, http.StatusOK, &issuesv1.GetUsersResponse{Users: pb})
}
