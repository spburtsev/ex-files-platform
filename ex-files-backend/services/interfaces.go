package services

import "github.com/spburtsev/ex-files-backend/models"

type UserRepository interface {
	FindByEmail(email string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
	Create(user *models.User) error
}

type TokenService interface {
	Issue(user *models.User) (string, error)
	Validate(tokenStr string) (*models.Claims, error)
}

type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}
