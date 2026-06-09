package seed

import (
	"log/slog"
	"os"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
	"gorm.io/gorm"
)

type seedUser struct {
	Name, Email, Password string
	Role                  models.Role
}

var seedUsers = []seedUser{
	{"Alex Johnson", "a.johnson@acme.org", "password123", models.RoleEmployee},
	{"Maria Chen", "m.chen@acme.org", "password123", models.RoleEmployee},
	{"James Wilson", "j.wilson@acme.org", "password123", models.RoleEmployee},
	{"Sofia Martinez", "s.martinez@acme.org", "password123", models.RoleManager},
}

func Run(db *gorm.DB, hasher services.Hasher) {
	var users []seedUser
	if root, ok := rootSeedUser(); ok {
		users = append(users, root)
	}
	// Demo accounts use a well-known password; never create them unless the
	// deployment opts in explicitly (local dev, integration tests).
	if os.Getenv("SEED_DEMO_USERS") == "true" {
		users = append(users, seedUsers...)
	} else {
		slog.Info("skipping demo user seed: set SEED_DEMO_USERS=true to create demo accounts", "component", "seed")
	}

	for _, su := range users {
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

func rootSeedUser() (seedUser, bool) {
	email := os.Getenv("SEED_ROOT_EMAIL")
	password := os.Getenv("SEED_ROOT_PASSWORD")
	if email == "" || password == "" {
		slog.Warn("skipping root user seed: set SEED_ROOT_EMAIL and SEED_ROOT_PASSWORD to create one",
			"component", "seed")
		return seedUser{}, false
	}
	name := os.Getenv("SEED_ROOT_NAME")
	if name == "" {
		name = "Root"
	}
	return seedUser{Name: name, Email: email, Password: password, Role: models.RoleRoot}, true
}
