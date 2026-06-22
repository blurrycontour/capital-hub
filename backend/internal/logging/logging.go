// Package logging configures the application's structured logger.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New builds a slog.Logger. In dev it uses a human-readable text handler;
// in prod it emits JSON. The level is parsed from the given string.
func New(level string, dev bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var handler slog.Handler
	if dev {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
