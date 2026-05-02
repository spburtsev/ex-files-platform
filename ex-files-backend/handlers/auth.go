package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	authv1 "github.com/spburtsev/ex-files-backend/gen/auth/v1"
	issuesv1 "github.com/spburtsev/ex-files-backend/gen/issues/v1"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

type AuthHandler struct {
	Repo        services.UserRepository
	Tokens      services.TokenService
	Hasher      services.Hasher
	Audit       services.AuditRepository
	Email       services.EmailService
	Cache       services.CacheService
	ResetTokens services.ResetTokenStore
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

// Register creates a new user account.
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      application/x-protobuf
// @Param        body  body      swagRegisterRequest  true  "Registration payload"
// @Success      201   {object}  swagAuthResponse     "Protobuf: auth.v1.RegisterResponse"
// @Failure      400   {object}  swagErrorResponse
// @Failure      409   {object}  swagErrorResponse
// @Router       /auth/register [post]
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

// Login authenticates a user and sets a session cookie.
// @Summary      Log in
// @Tags         auth
// @Accept       json
// @Produce      application/x-protobuf
// @Param        body  body      swagLoginRequest  true  "Login credentials"
// @Success      200   {object}  swagAuthResponse  "Protobuf: auth.v1.LoginResponse"
// @Failure      400   {object}  swagErrorResponse
// @Failure      401   {object}  swagErrorResponse
// @Router       /auth/login [post]
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

// Me returns the current authenticated user.
// @Summary      Current user
// @Tags         auth
// @Produce      application/x-protobuf
// @Success      200  {object}  swagMeResponse     "Protobuf: auth.v1.MeResponse"
// @Failure      401  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}

	cacheKey := fmt.Sprintf("user:%d", userID)
	if h.Cache != nil {
		if cached, hit := h.Cache.Get(cacheKey); hit {
			var user models.User
			if err := json.Unmarshal(cached, &user); err == nil {
				protobufResponse(c, http.StatusOK, &authv1.MeResponse{User: userToProto(&user)})
				return
			}
		}
	}

	user, err := h.Repo.FindByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if h.Cache != nil {
		if data, err := json.Marshal(user); err == nil {
			h.Cache.Set(cacheKey, data, 30*time.Second)
		}
	}
	protobufResponse(c, http.StatusOK, &authv1.MeResponse{User: userToProto(user)})
}

// Logout clears the session cookie.
// @Summary      Log out
// @Tags         auth
// @Produce      application/x-protobuf
// @Success      200  {object}  swagMessageResponse  "Protobuf: auth.v1.LogoutResponse"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("session", "", -1, "/", "", false, true)
	protobufResponse(c, http.StatusOK, &authv1.LogoutResponse{Message: "logged out"})
}

// ListUsers returns all users.
// @Summary      List users
// @Tags         auth
// @Produce      application/x-protobuf
// @Success      200  {object}  swagGetUsersResponse  "Protobuf: issues.v1.GetUsersResponse"
// @Failure      401  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /auth/users [get]
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

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"    binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// ForgotPassword generates a reset token and emails a reset link.
// @Summary      Request password reset
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      forgotPasswordRequest  true  "Email"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  swagErrorResponse
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Always return 200 regardless of whether user exists (prevent enumeration).
	user, err := h.Repo.FindByEmail(req.Email)
	if err != nil {
		slog.Debug("forgot-password: user not found", "email", req.Email)
		c.JSON(http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"})
		return
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		slog.Error("forgot-password: failed to generate token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	tokenStr := hex.EncodeToString(tokenBytes)

	if err := h.ResetTokens.StoreResetToken(tokenStr, user.ID, 1*time.Hour); err != nil {
		slog.Error("forgot-password: failed to save token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	if h.Email != nil {
		subject := "Password Reset — Ex-Files"
		body := fmt.Sprintf(
			"<p>Hello %s,</p>"+
				"<p>You requested a password reset. Use the following token to reset your password:</p>"+
				"<p><strong>%s</strong></p>"+
				"<p>This token expires in 1 hour. If you did not request this, ignore this email.</p>",
			user.Name, tokenStr,
		)
		if err := h.Email.Send(user.Email, subject, body); err != nil {
			slog.Error("forgot-password: failed to send email", "error", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"})
}

// ResetPassword validates a reset token and updates the password.
// @Summary      Reset password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      resetPasswordRequest  true  "Token and new password"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  swagErrorResponse
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := h.ResetTokens.GetResetTokenUserID(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}

	hash, err := h.Hasher.Hash(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	if err := h.Repo.UpdatePassword(userID, hash); err != nil {
		slog.Error("reset-password: failed to update password", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	if err := h.ResetTokens.DeleteResetToken(req.Token); err != nil {
		slog.Error("reset-password: failed to delete token", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "password has been reset"})
}
