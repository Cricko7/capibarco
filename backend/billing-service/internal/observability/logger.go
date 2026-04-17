package observability

import (
	"log/slog"
	"os"

	"github.com/petmatch/petmatch/internal/config"
)

func NewLogger(cfg config.AppConfig) *slog.Logger {
	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level, AddSource: cfg.Development}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts)).With(
		slog.String("service", cfg.Name),
		slog.String("version", cfg.Version),
		slog.String("env", cfg.Env),
	)
}
