package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

var (
	// Default logger
	Log *slog.Logger
)

func init() {
	// Initialize with a default text handler
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	Log = slog.New(slog.NewTextHandler(os.Stderr, opts))
}

// Setup initializes the global logger
func Setup(format string, level string) {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: parseLevel(level),
	}

	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, opts)
	default:
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	Log = slog.New(handler)
	slog.SetDefault(Log)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Helper functions for easy access

func Debug(msg string, args ...any) {
	Log.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	Log.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	Log.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	Log.Error(msg, args...)
}

func With(args ...any) *slog.Logger {
	return Log.With(args...)
}

func FromContext(ctx context.Context) *slog.Logger {
	// TODO: Extract logger from context if middleware sets it
	return Log
}
