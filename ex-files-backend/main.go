package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
	"github.com/spburtsev/ex-files-backend/tracing"
)

func main() {
	if err := godotenv.Load(); err != nil {
		_ = err
	}

	logging.Init()
	slog.Info("starting ex-files-backend")

	ctx := context.Background()
	tracingShutdown, err := tracing.Init(ctx)
	if err != nil {
		slog.Error("failed to init tracing", "error", err)
		os.Exit(1)
	}
	defer func() { _ = tracingShutdown(context.Background()) }()

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "host=localhost user=admin password=admin dbname=exfiles port=5433 sslmode=disable TimeZone=UTC"
	}
	var db *gorm.DB
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Workspace{}, &models.WorkspaceMember{}, &models.Issue{}, &models.Document{}, &models.Comment{}, &models.IssueReviewer{}, &models.DocumentApproval{}); err != nil {
		slog.Error("auto-migrate failed", "error", err)
		os.Exit(1)
	}

	tokens := services.NewJWTTokenService(os.Getenv("JWT_SECRET"))
	userRepo := &services.GormUserRepository{DB: db}
	hasher := services.BcryptHasher{Cost: bcrypt.DefaultCost}
	wsRepo := &services.GormWorkspaceRepository{DB: db}
	issueRepo := &services.GormIssueRepository{DB: db}
	docRepo := &services.GormDocumentRepository{DB: db}
	approvalRepo := &services.GormDocumentApprovalRepository{DB: db}
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

	emailSvc := newEmailService()
	sseHub := services.NewSSEHub()

	// Periodic deadline reminder scheduler. Runs in the background; cancelling
	// ctx (set up just below) will stop it on shutdown.
	deadlineSched := &services.DeadlineScheduler{
		Hub:       sseHub,
		IssueRepo: issueRepo,
		Tick:      10 * time.Minute,
	}
	go deadlineSched.Run(ctx)

	rdb, err := services.NewRedisClient(envOr("REDIS_ADDR", "localhost:6380"))
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	server := &handlers.Server{
		UserRepo:      userRepo,
		Tokens:        tokens,
		Hasher:        hasher,
		Email:         emailSvc,
		Cache:         rdb,
		ResetTokens:   rdb,
		WorkspaceRepo: wsRepo,
		IssueRepo:     issueRepo,
		DocumentRepo:  docRepo,
		ApprovalRepo:  approvalRepo,
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
		func(h http.Handler) http.Handler { return otelhttp.NewHandler(h, "ex-files-backend") },
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

// newEmailService picks the EmailService implementation based on the
// EMAIL_PROVIDER env var. Defaults to Resend so existing deployments keep
// working unchanged. Set EMAIL_PROVIDER=smtp to route through any SMTP
// backend (Mailtrap sandbox in dev, etc.) using the SMTP_* env vars.
func newEmailService() services.EmailService {
	switch strings.ToLower(os.Getenv("EMAIL_PROVIDER")) {
	case "smtp":
		port, err := strconv.Atoi(envOr("SMTP_PORT", "587"))
		if err != nil {
			slog.Warn("invalid SMTP_PORT, falling back to 587", "error", err)
			port = 587
		}
		return services.NewSMTPEmailService(
			os.Getenv("SMTP_HOST"),
			port,
			os.Getenv("SMTP_USER"),
			os.Getenv("SMTP_PASSWORD"),
			envOr("SMTP_FROM", "ex-files <noreply@ex-files.local>"),
		)
	default:
		return services.NewResendEmailService(
			os.Getenv("RESEND_API_KEY"),
			envOr("RESEND_FROM", "ex-files <noreply@ex-files.dev>"),
			os.Getenv("RESEND_DEV_TRAP"),
		)
	}
}
