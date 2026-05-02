package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/logging"
	"github.com/spburtsev/ex-files-backend/middleware"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/seed"
	"github.com/spburtsev/ex-files-backend/services"
)

// @title           ex-files Platform API
// @version         1.0
// @description     Document management and review platform. Successful responses use protobuf binary encoding (application/x-protobuf). Error responses are JSON. Schemas below show JSON-equivalent structure of each protobuf message.
// @host            localhost:8080
// @BasePath        /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
//
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name session
func main() {
	if err := godotenv.Load(); err != nil {
		// .env is optional; will be logged after logger init
		_ = err
	}

	logging.Init()
	slog.Info("starting ex-files-backend")

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "host=localhost user=admin password=admin dbname=exfiles port=5433 sslmode=disable TimeZone=UTC"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Workspace{}, &models.WorkspaceMember{}, &models.AuditEntry{}, &models.Issue{}, &models.Document{}, &models.DocumentVersion{}, &models.Comment{}); err != nil {
		slog.Error("auto-migrate failed", "error", err)
		os.Exit(1)
	}

	ts := services.NewJWTTokenService(os.Getenv("JWT_SECRET"))
	repo := &services.GormUserRepository{DB: db}
	hasher := services.BcryptHasher{Cost: bcrypt.DefaultCost}

	auditRepo := &services.GormAuditRepository{DB: db}

	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = "localhost:9002"
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
		slog.Error("failed to connect to MinIO", "error", err)
		os.Exit(1)
	}

	resendKey := os.Getenv("RESEND_API_KEY")
	resendFrom := os.Getenv("RESEND_FROM")
	if resendFrom == "" {
		resendFrom = "ex-files <noreply@ex-files.dev>"
	}
	emailSvc := services.NewResendEmailService(resendKey, resendFrom)
	sseHub := services.NewSSEHub()

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6380"
	}
	rdb, err := services.NewRedisClient(redisAddr)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	auth := &handlers.AuthHandler{Repo: repo, Tokens: ts, Hasher: hasher, Audit: auditRepo, Email: emailSvc, Cache: rdb, ResetTokens: rdb}
	wsRepo := &services.GormWorkspaceRepository{DB: db}
	ws := &handlers.WorkspaceHandler{Repo: wsRepo, UserRepo: repo, Audit: auditRepo}
	audit := &handlers.AuditHandler{Repo: auditRepo, DB: db}
	docRepo := &services.GormDocumentRepository{DB: db}
	docs := &handlers.DocumentHandler{Repo: docRepo, Storage: storage, Audit: auditRepo, UserRepo: repo, Email: emailSvc, Hub: sseHub}
	sse := &handlers.SSEHandler{Hub: sseHub}
	commentRepo := &services.GormCommentRepository{DB: db}
	comments := &handlers.CommentHandler{Repo: commentRepo, Audit: auditRepo, Hub: sseHub}
	verify := &handlers.VerifyHandler{Repo: docRepo}

	seed.Run(db, hasher)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger())

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:5173,http://localhost:4173"
	}
	slog.Debug("CORS configuration", "origins", strings.Split(corsOrigins, ","))
	router.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(corsOrigins, ","),
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
		authRoutes.POST("/forgot-password", auth.ForgotPassword)
		authRoutes.POST("/reset-password", auth.ResetPassword)
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
		workspaceRoutes.GET("/:id/assignable-members", ws.AssignableMembers)
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
		documentRoutes.GET("/:id/versions/:versionId/file", docs.File)
		documentRoutes.POST("/:id/submit", docs.Submit)
		documentRoutes.POST("/:id/resubmit", docs.Resubmit)
		documentRoutes.POST("/:id/approve", docs.Approve)
		documentRoutes.POST("/:id/reject", docs.Reject)
		documentRoutes.POST("/:id/request-changes", docs.RequestChanges)
		documentRoutes.PUT("/:id/reviewer", docs.AssignReviewer)
		documentRoutes.POST("/:id/comments", comments.Create)
		documentRoutes.GET("/:id/comments", comments.List)
	}

	auditRoutes := router.Group("/audit", middleware.AuthMiddleware(ts))
	{
		auditRoutes.GET("", audit.List)
		auditRoutes.GET("/stats", audit.Stats)
	}
	router.GET("/events", middleware.AuthMiddleware(ts), sse.Stream)
	router.GET("/verify", verify.Verify)

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
