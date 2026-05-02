package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/spburtsev/ex-files-backend/middleware"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
)

// AuthRegister implements POST /auth/register.
func (s *Server) AuthRegister(ctx context.Context, req *oapi.RegisterRequest) (oapi.AuthRegisterRes, error) {
	if _, err := s.UserRepo.FindByEmail(req.Email); err == nil {
		return &oapi.AuthRegisterConflict{Error: "email already registered"}, nil
	}

	hash, err := s.Hasher.Hash(req.Password)
	if err != nil {
		logErr("auth.register.hash", err)
		return &oapi.AuthRegisterInternalServerError{Error: "internal error"}, nil
	}

	user := models.User{
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: hash,
	}
	if err := s.UserRepo.Create(&user); err != nil {
		logErr("auth.register.create", err)
		return &oapi.AuthRegisterInternalServerError{Error: "failed to create user"}, nil
	}

	token, err := s.Tokens.Issue(&user)
	if err != nil {
		logErr("auth.register.issue", err)
		return &oapi.AuthRegisterInternalServerError{Error: "failed to issue token"}, nil
	}

	logAudit(s.Audit, models.AuditActionUserRegistered, user.ID, uintPtr(user.ID), "user", map[string]any{
		"email": user.Email,
	})

	middleware.SetSessionCookie(ctx, token)
	return &oapi.AuthResponse{
		User:  userToOAPI(&user),
		Token: token,
	}, nil
}

// AuthLogin implements POST /auth/login.
func (s *Server) AuthLogin(ctx context.Context, req *oapi.LoginRequest) (oapi.AuthLoginRes, error) {
	user, err := s.UserRepo.FindByEmail(req.Email)
	if err != nil {
		slog.Debug("login failed: user not found", "email", req.Email)
		return &oapi.AuthLoginUnauthorized{Error: "invalid credentials"}, nil
	}
	if err := s.Hasher.Compare(user.PasswordHash, req.Password); err != nil {
		slog.Debug("login failed: wrong password", "email", req.Email)
		return &oapi.AuthLoginUnauthorized{Error: "invalid credentials"}, nil
	}

	token, err := s.Tokens.Issue(user)
	if err != nil {
		logErr("auth.login.issue", err)
		return &oapi.AuthLoginInternalServerError{Error: "failed to issue token"}, nil
	}

	logAudit(s.Audit, models.AuditActionUserLoggedIn, user.ID, uintPtr(user.ID), "user", map[string]any{
		"email": user.Email,
	})

	middleware.SetSessionCookie(ctx, token)
	return &oapi.AuthResponse{
		User:  userToOAPI(user),
		Token: token,
	}, nil
}

// AuthLogout implements POST /auth/logout.
func (s *Server) AuthLogout(ctx context.Context) (oapi.AuthLogoutRes, error) {
	middleware.ClearSessionCookie(ctx)
	return &oapi.MessageResponse{Message: "logged out"}, nil
}

// AuthMe implements GET /auth/me.
func (s *Server) AuthMe(ctx context.Context) (oapi.AuthMeRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.AuthMeUnauthorized{Error: "unauthorized"}, nil
	}

	cacheKey := fmt.Sprintf("user:%d", uid)
	if s.Cache != nil {
		if cached, hit := s.Cache.Get(cacheKey); hit {
			var u models.User
			if err := json.Unmarshal(cached, &u); err == nil {
				resp := oapi.MeResponse{User: userToOAPI(&u)}
				return &resp, nil
			}
		}
	}

	user, err := s.UserRepo.FindByID(uid)
	if err != nil {
		return &oapi.AuthMeNotFound{Error: "user not found"}, nil
	}

	if s.Cache != nil {
		if data, err := json.Marshal(user); err == nil {
			s.Cache.Set(cacheKey, data, 30*time.Second)
		}
	}
	resp := oapi.MeResponse{User: userToOAPI(user)}
	return &resp, nil
}

// AuthListUsers implements GET /auth/users.
func (s *Server) AuthListUsers(ctx context.Context) (oapi.AuthListUsersRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.AuthListUsersUnauthorized{Error: "unauthorized"}, nil
	}
	users, err := s.UserRepo.ListAll()
	if err != nil {
		logErr("auth.users.list", err)
		return &oapi.AuthListUsersInternalServerError{Error: "failed to list users"}, nil
	}
	out := make([]oapi.User, len(users))
	for i := range users {
		out[i] = userToOAPI(&users[i])
	}
	return &oapi.GetUsersResponse{Users: out}, nil
}

// AuthForgotPassword implements POST /auth/forgot-password.
// Always returns 200 to prevent email enumeration.
func (s *Server) AuthForgotPassword(ctx context.Context, req *oapi.ForgotPasswordRequest) (oapi.AuthForgotPasswordRes, error) {
	const stockMessage = "if the email exists, a reset link has been sent"

	user, err := s.UserRepo.FindByEmail(req.Email)
	if err != nil {
		return &oapi.MessageResponse{Message: stockMessage}, nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		logErr("auth.forgot.rand", err)
		return &oapi.AuthForgotPasswordInternalServerError{Error: "internal error"}, nil
	}
	tokenStr := hex.EncodeToString(tokenBytes)

	if err := s.ResetTokens.StoreResetToken(tokenStr, user.ID, time.Hour); err != nil {
		logErr("auth.forgot.store", err)
		return &oapi.AuthForgotPasswordInternalServerError{Error: "internal error"}, nil
	}

	if s.Email != nil {
		subject := "Password Reset - Ex-Files"
		body := fmt.Sprintf(
			"<p>Hello %s,</p>"+
				"<p>You requested a password reset. Use the following token to reset your password:</p>"+
				"<p><strong>%s</strong></p>"+
				"<p>This token expires in 1 hour. If you did not request this, ignore this email.</p>",
			user.Name, tokenStr,
		)
		if err := s.Email.Send(user.Email, subject, body); err != nil {
			slog.Error("forgot-password: email send failed", "error", err)
		}
	}

	return &oapi.MessageResponse{Message: stockMessage}, nil
}

// AuthResetPassword implements POST /auth/reset-password.
func (s *Server) AuthResetPassword(ctx context.Context, req *oapi.ResetPasswordRequest) (oapi.AuthResetPasswordRes, error) {
	uid, err := s.ResetTokens.GetResetTokenUserID(req.Token)
	if err != nil {
		return &oapi.AuthResetPasswordBadRequest{Error: "invalid or expired token"}, nil
	}
	hash, err := s.Hasher.Hash(req.Password)
	if err != nil {
		logErr("auth.reset.hash", err)
		return &oapi.AuthResetPasswordInternalServerError{Error: "internal error"}, nil
	}
	if err := s.UserRepo.UpdatePassword(uid, hash); err != nil {
		logErr("auth.reset.update", err)
		return &oapi.AuthResetPasswordInternalServerError{Error: "internal error"}, nil
	}
	if err := s.ResetTokens.DeleteResetToken(req.Token); err != nil {
		slog.Error("reset-password: failed to delete token", "error", err)
	}
	return &oapi.MessageResponse{Message: "password has been reset"}, nil
}
