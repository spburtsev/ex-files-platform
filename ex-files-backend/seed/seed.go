package seed

import (
	"log/slog"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
	"gorm.io/gorm"
)

type seedUser struct {
	Name, Email, Password string
	Role                  models.Role
}

var seedUsers = []seedUser{
	{"Sergei Burtsev", "sergei.p.burtsev@gmail.com", "admin", models.RoleRoot},
	{"Alex Johnson", "a.johnson@acme.org", "password123", models.RoleEmployee},
	{"Maria Chen", "m.chen@acme.org", "password123", models.RoleEmployee},
	{"James Wilson", "j.wilson@acme.org", "password123", models.RoleEmployee},
	{"Sofia Martinez", "s.martinez@acme.org", "password123", models.RoleManager},
}

func Run(db *gorm.DB, hasher services.Hasher) {
	// Seed users
	for _, su := range seedUsers {
		var existing models.User
		if db.Where("email = ?", su.Email).First(&existing).Error == nil {
			continue
		}
		hash, err := hasher.Hash(su.Password)
		if err != nil {
			slog.Error("hash error", "component", "seed", "email", su.Email, "error", err)
			continue
		}
		u := models.User{
			Name:         su.Name,
			Email:        su.Email,
			PasswordHash: hash,
			Role:         su.Role,
		}
		if err := db.Create(&u).Error; err != nil {
			slog.Error("create error", "component", "seed", "email", su.Email, "error", err)
		} else {
			slog.Info("created user", "component", "seed", "name", su.Name, "email", su.Email)
		}
	}
}
