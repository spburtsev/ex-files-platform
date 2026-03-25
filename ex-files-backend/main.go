package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/middleware"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/seed"
	"github.com/spburtsev/ex-files-backend/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading environment variables from shell")
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "host=localhost user=admin password=admin dbname=exfiles port=5432 sslmode=disable TimeZone=UTC"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatal("auto-migrate failed:", err)
	}

	ts := services.NewJWTTokenService(os.Getenv("JWT_SECRET"))
	repo := &services.GormUserRepository{DB: db}
	hasher := services.BcryptHasher{Cost: bcrypt.DefaultCost}

	auth := &handlers.AuthHandler{Repo: repo, Tokens: ts, Hasher: hasher}

	seed.Run(db, hasher)

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:4173"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	assignments := &handlers.AssignmentsHandler{}
	router.GET("/users", assignments.GetUsers)
	router.GET("/assignments", assignments.GetAssignments)
	router.GET("/assignments/:id", assignments.GetAssignment)

	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/register", auth.Register)
		authRoutes.POST("/login", auth.Login)
		authRoutes.POST("/logout", auth.Logout)
		authRoutes.GET("/me", middleware.AuthMiddleware(ts), auth.Me)
	}

	router.Run()
}
