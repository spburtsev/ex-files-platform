package seed

import (
	"log"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
	"gorm.io/gorm"
)

type seedUser struct {
	Name, Email, Password string
	Role                  models.Role
}

var mockUsers = []seedUser{
	{"Sergei Burtsev", "sergei.p.burtsev@gmail.com", "admin", models.RoleRoot},
	{"Alex Johnson", "a.johnson@acme.org", "password123", models.RoleEmployee},
	{"Maria Chen", "m.chen@acme.org", "password123", models.RoleEmployee},
	{"James Wilson", "j.wilson@acme.org", "password123", models.RoleEmployee},
	{"Sofia Martinez", "s.martinez@acme.org", "password123", models.RoleManager},
}

func Run(db *gorm.DB, hasher services.Hasher) {
	for _, su := range mockUsers {
		var existing models.User
		if db.Where("email = ?", su.Email).First(&existing).Error == nil {
			continue
		}
		hash, err := hasher.Hash(su.Password)
		if err != nil {
			log.Printf("seed: hash error for %s: %v", su.Email, err)
			continue
		}
		u := models.User{
			Name:         su.Name,
			Email:        su.Email,
			PasswordHash: hash,
			Role:         su.Role,
		}
		if err := db.Create(&u).Error; err != nil {
			log.Printf("seed: create error for %s: %v", su.Email, err)
		} else {
			log.Printf("seed: created %s (%s)", su.Name, su.Email)
		}
	}
}
