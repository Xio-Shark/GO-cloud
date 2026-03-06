package bootstrap

import (
	"log/slog"
	"os"
)

func SetupLogger(cfg Config) *slog.Logger {
	level := parseLogLevel(cfg.LogLevel)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler).With("service", cfg.ServiceName)
	slog.SetDefault(logger)
	return logger
}

func parseLogLevel(raw string) slog.Level {
	switch raw {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
