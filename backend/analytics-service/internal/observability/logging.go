package observability

import (
	"log/slog"
	"os"

	"github.com/petmatch/petmatch/internal/config"
)

func NewLogger(cfg config.AppConfig) *slog.Logger {
	opts := &slog.HandlerOptions{AddSource: cfg.Env != "prod"}
	if cfg.Env == "prod" {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts)).With(
			slog.String("service", cfg.Name),
			slog.String("version", cfg.Version),
		)
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts)).With(
		slog.String("service", cfg.Name),
		slog.String("version", cfg.Version),
	)
}
