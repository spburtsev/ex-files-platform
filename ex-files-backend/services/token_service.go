package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/spburtsev/ex-files-backend/models"
)

const tokenDuration = 8 * time.Hour

type JWTTokenService struct {
	secret string
}

func NewJWTTokenService(secret string) *JWTTokenService {
	return &JWTTokenService{secret: secret}
}

func (s *JWTTokenService) Issue(user *models.User) (string, error) {
	claims := models.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(s.secret))
}

func (s *JWTTokenService) Validate(tokenStr string) (*models.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &models.Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*models.Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
