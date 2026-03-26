package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	authv1 "github.com/spburtsev/ex-files-backend/gen/auth/v1"
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
	c.SetCookie("session", token, int((8 * time.Hour).Seconds()), "/", "", false, true)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := h.Repo.FindByEmail(req.Email); err == nil {
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

	user, err := h.Repo.FindByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := h.Hasher.Compare(user.PasswordHash, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := h.Tokens.Issue(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

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
	userID, _ := c.Get("user_id")
	user, err := h.Repo.FindByID(userID.(uint))
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
	pb := make([]*authv1.User, len(users))
	for i := range users {
		pb[i] = userToProto(&users[i])
	}
	c.JSON(http.StatusOK, gin.H{"users": pb})
}
