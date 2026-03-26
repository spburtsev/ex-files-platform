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

	if err := db.AutoMigrate(&models.User{}, &models.Workspace{}, &models.WorkspaceMember{}, &models.AuditEntry{}, &models.Issue{}, &models.Document{}, &models.DocumentVersion{}); err != nil {
		log.Fatal("auto-migrate failed:", err)
	}

	ts := services.NewJWTTokenService(os.Getenv("JWT_SECRET"))
	repo := &services.GormUserRepository{DB: db}
	hasher := services.BcryptHasher{Cost: bcrypt.DefaultCost}

	auditRepo := &services.GormAuditRepository{DB: db}

	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = "localhost:9000"
	}
	minioAccessKey := os.Getenv("MINIO_ACCESS_KEY")
	if minioAccessKey == "" {
		minioAccessKey = "minioadmin"
	}
	minioSecretKey := os.Getenv("MINIO_SECRET_KEY")
	if minioSecretKey == "" {
		minioSecretKey = "minioadmin"
	}
	minioBucket := os.Getenv("MINIO_BUCKET")
	if minioBucket == "" {
		minioBucket = "documents"
	}

	storage, err := services.NewMinIOStorage(minioEndpoint, minioAccessKey, minioSecretKey, minioBucket, false)
	if err != nil {
		log.Fatal("failed to connect to MinIO:", err)
	}

	auth := &handlers.AuthHandler{Repo: repo, Tokens: ts, Hasher: hasher, Audit: auditRepo}
	wsRepo := &services.GormWorkspaceRepository{DB: db}
	ws := &handlers.WorkspaceHandler{Repo: wsRepo, UserRepo: repo, Audit: auditRepo}
	audit := &handlers.AuditHandler{Repo: auditRepo}
	docRepo := &services.GormDocumentRepository{DB: db}
	docs := &handlers.DocumentHandler{Repo: docRepo, Storage: storage, Audit: auditRepo}

	seed.Run(db, hasher)

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:4173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"X-Total-Count", "X-Page", "X-Per-Page", "X-Total-Pages"},
		AllowCredentials: true,
	}))

	issueRepo := &services.GormIssueRepository{DB: db}
	issues := &handlers.IssuesHandler{Repo: issueRepo, UserRepo: repo, Audit: auditRepo}

	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/register", auth.Register)
		authRoutes.POST("/login", auth.Login)
		authRoutes.POST("/logout", auth.Logout)
		authRoutes.GET("/me", middleware.AuthMiddleware(ts), auth.Me)
		authRoutes.GET("/users", middleware.AuthMiddleware(ts), auth.ListUsers)
	}

	workspaceRoutes := router.Group("/workspaces", middleware.AuthMiddleware(ts))
	{
		workspaceRoutes.POST("", ws.Create)
		workspaceRoutes.GET("", ws.List)
		workspaceRoutes.GET("/:id", ws.Get)
		workspaceRoutes.PUT("/:id", ws.Update)
		workspaceRoutes.DELETE("/:id", ws.Delete)
		workspaceRoutes.POST("/:id/members", ws.AddMember)
		workspaceRoutes.DELETE("/:id/members/:userId", ws.RemoveMember)
		workspaceRoutes.GET("/:id/issues", issues.ListByWorkspace)
		workspaceRoutes.POST("/:id/issues", issues.Create)
	}

	issueRoutes := router.Group("/issues", middleware.AuthMiddleware(ts))
	{
		issueRoutes.GET("/:id", issues.Get)
		issueRoutes.POST("/:id/documents", docs.Upload)
		issueRoutes.GET("/:id/documents", docs.List)
	}

	documentRoutes := router.Group("/documents", middleware.AuthMiddleware(ts))
	{
		documentRoutes.GET("/:id", docs.Get)
		documentRoutes.DELETE("/:id", docs.Delete)
		documentRoutes.POST("/:id/versions", docs.UploadVersion)
		documentRoutes.GET("/:id/versions/:versionId/download", docs.Download)
		documentRoutes.POST("/:id/submit", docs.Submit)
		documentRoutes.POST("/:id/resubmit", docs.Resubmit)
		documentRoutes.POST("/:id/approve", docs.Approve)
		documentRoutes.POST("/:id/reject", docs.Reject)
		documentRoutes.POST("/:id/request-changes", docs.RequestChanges)
		documentRoutes.PUT("/:id/reviewer", docs.AssignReviewer)
	}

	auditRoutes := router.Group("/audit", middleware.AuthMiddleware(ts))
	{
		auditRoutes.GET("", audit.List)
	}

	router.Run()
}
