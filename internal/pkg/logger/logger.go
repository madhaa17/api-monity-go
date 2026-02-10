package logger

import (
	"log/slog"
	"os"
)

// New returns a configured slog.Logger and sets it as the default.
// - development: human-readable text, LevelDebug
// - production: JSON to stdout, LevelInfo (Docker-parseable)
func New(env string) *slog.Logger {
	var handler slog.Handler
	switch env {
	case "production":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	default:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	log := slog.New(handler)
	slog.SetDefault(log)
	return log
}
