package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/logging"
	"github.com/spburtsev/ex-files-backend/middleware"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
	"github.com/spburtsev/ex-files-backend/seed"
	"github.com/spburtsev/ex-files-backend/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
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

	tokens := services.NewJWTTokenService(os.Getenv("JWT_SECRET"))
	userRepo := &services.GormUserRepository{DB: db}
	hasher := services.BcryptHasher{Cost: bcrypt.DefaultCost}
	auditRepo := &services.GormAuditRepository{DB: db}
	wsRepo := &services.GormWorkspaceRepository{DB: db}
	issueRepo := &services.GormIssueRepository{DB: db}
	docRepo := &services.GormDocumentRepository{DB: db}
	commentRepo := &services.GormCommentRepository{DB: db}

	minioEndpoint := envOr("MINIO_ENDPOINT", "localhost:9002")
	minioAccessKey := envOr("MINIO_ACCESS_KEY", "minioadmin")
	minioSecretKey := envOr("MINIO_SECRET_KEY", "minioadmin")
	minioBucket := envOr("MINIO_BUCKET", "documents")
	storage, err := services.NewMinIOStorage(minioEndpoint, minioAccessKey, minioSecretKey, minioBucket, false)
	if err != nil {
		slog.Error("failed to connect to MinIO", "error", err)
		os.Exit(1)
	}

	resendKey := os.Getenv("RESEND_API_KEY")
	resendFrom := envOr("RESEND_FROM", "ex-files <noreply@ex-files.dev>")
	emailSvc := services.NewResendEmailService(resendKey, resendFrom)
	sseHub := services.NewSSEHub()

	rdb, err := services.NewRedisClient(envOr("REDIS_ADDR", "localhost:6380"))
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	server := &handlers.Server{
		UserRepo:      userRepo,
		Tokens:        tokens,
		Hasher:        hasher,
		Audit:         auditRepo,
		Email:         emailSvc,
		Cache:         rdb,
		ResetTokens:   rdb,
		WorkspaceRepo: wsRepo,
		IssueRepo:     issueRepo,
		DocumentRepo:  docRepo,
		CommentRepo:   commentRepo,
		Storage:       storage,
		Hub:           sseHub,
		DB:            db,
	}

	seed.Run(db, hasher)

	ogenServer, err := oapi.NewServer(server, server)
	if err != nil {
		slog.Error("failed to construct ogen server", "error", err)
		os.Exit(1)
	}

	sse := &handlers.SSEHandler{Hub: sseHub}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.Handle("/events", middleware.RequireAuth(tokens)(sse))
	mux.Handle("/", ogenServer)

	corsOrigins := envOr("CORS_ORIGINS", "http://localhost:5173,http://localhost:4173")
	slog.Debug("CORS configuration", "origins", strings.Split(corsOrigins, ","))
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(corsOrigins, ","),
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"X-Total-Count", "X-Page", "X-Per-Page", "X-Total-Pages"},
		AllowCredentials: true,
	})

	root := middleware.Chain(mux,
		corsHandler.Handler,
		middleware.Recovery(),
		middleware.RequestLogger(),
		middleware.WithCookieJar,
	)

	port := envOr("PORT", "8080")
	slog.Info("listening", "addr", ":"+port)
	if err := http.ListenAndServe(":"+port, root); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
