package logging

import (
	"log/slog"
	"os"
	"strings"
)

func New() *slog.Logger {
	format := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
	env := strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))

	if format == "json" || env == "production" || env == "prod" {
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
}
