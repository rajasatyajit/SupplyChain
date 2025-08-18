package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		out  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
	}
	for _, tt := range tests {
		if got := parseLevel(tt.in); got != tt.out {
			t.Errorf("parseLevel(%q)=%v want %v", tt.in, got, tt.out)
		}
	}
}

func TestInitAndWithContext(t *testing.T) {
	// Ensure Init does not panic and sets default logger
	Init("debug", "text")
	if defaultLogger == nil {
		t.Fatalf("defaultLogger not initialized")
	}

	// WithContext should return a non-nil logger and be safe to use
	ctx := context.WithValue(context.Background(), "request_id", "req-123")
	ctx = context.WithValue(ctx, "trace_id", "trace-abc")
	l := WithContext(ctx)
	if l == nil {
		t.Fatalf("WithContext returned nil")
	}

	// Exercise logging methods to ensure they don't panic
	Info("info message", "k", "v")
	Warn("warn message")
	Error("error message")
	Debug("debug message")
}
