package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Init configures the default slog logger with a JSON handler.
// Log level is read from the LOG_LEVEL env var (debug, info, warn, error).
// Defaults to info if unset or unrecognised.
func Init() {
	var level slog.Level
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}
