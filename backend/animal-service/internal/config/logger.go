package config

import (
	"log/slog"
	"os"
)

// NewLogger creates a JSON slog logger for production-friendly logs.
func NewLogger(cfg Config) *slog.Logger {
	level := new(slog.LevelVar)
	switch cfg.Logging.Level {
	case "debug":
		level.Set(slog.LevelDebug)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})).With(
		"service", cfg.Service.Name,
		"version", cfg.Service.Version,
		"environment", cfg.Service.Environment,
	)
}
