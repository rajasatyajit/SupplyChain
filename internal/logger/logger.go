package logger

import (
	"context"
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

// Init initializes the global logger
func Init(level, format string) {
	var handler slog.Handler
	
	logLevel := parseLevel(level)
	opts := &slog.HandlerOptions{
		Level: logLevel,
		AddSource: true,
	}

	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// parseLevel converts string level to slog.Level
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithContext returns a logger with context values
func WithContext(ctx context.Context) *slog.Logger {
	return defaultLogger.With(
		"request_id", ctx.Value("request_id"),
		"trace_id", ctx.Value("trace_id"),
	)
}

// Info logs an info message
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
	os.Exit(1)
}