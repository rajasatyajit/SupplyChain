package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original env vars
	originalVars := map[string]string{
		"SERVER_PORT":     os.Getenv("SERVER_PORT"),
		"DATABASE_URL":    os.Getenv("DATABASE_URL"),
		"LOG_LEVEL":       os.Getenv("LOG_LEVEL"),
		"METRICS_ENABLED": os.Getenv("METRICS_ENABLED"),
	}

	// Clean up after test
	defer func() {
		for key, value := range originalVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("Default configuration", func(t *testing.T) {
		// Clear env vars
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("METRICS_ENABLED")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if cfg.Server.Port != 8080 {
			t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
		}

		if cfg.Database.URL != "" {
			t.Errorf("Expected empty database URL, got %s", cfg.Database.URL)
		}

		if cfg.Logging.Level != "info" {
			t.Errorf("Expected default log level 'info', got %s", cfg.Logging.Level)
		}

		if !cfg.Metrics.Enabled {
			t.Errorf("Expected metrics enabled by default")
		}
	})

	t.Run("Custom configuration", func(t *testing.T) {
		os.Setenv("SERVER_PORT", "9000")
		os.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("METRICS_ENABLED", "false")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if cfg.Server.Port != 9000 {
			t.Errorf("Expected port 9000, got %d", cfg.Server.Port)
		}

		if cfg.Database.URL != "postgres://test:test@localhost/test" {
			t.Errorf("Expected custom database URL, got %s", cfg.Database.URL)
		}

		if cfg.Logging.Level != "debug" {
			t.Errorf("Expected log level 'debug', got %s", cfg.Logging.Level)
		}

		if cfg.Metrics.Enabled {
			t.Errorf("Expected metrics disabled")
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "Valid configuration",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					MaxConns: 10,
				},
				Pipeline: PipelineConfig{
					WorkerCount: 4,
				},
			},
			expectError: false,
		},
		{
			name: "Invalid port",
			config: Config{
				Server: ServerConfig{
					Port: 70000,
				},
				Database: DatabaseConfig{
					MaxConns: 10,
				},
				Pipeline: PipelineConfig{
					WorkerCount: 4,
				},
			},
			expectError: true,
		},
		{
			name: "Invalid max connections",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					MaxConns: 0,
				},
				Pipeline: PipelineConfig{
					WorkerCount: 4,
				},
			},
			expectError: true,
		},
		{
			name: "Invalid worker count",
			config: Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					MaxConns: 10,
				},
				Pipeline: PipelineConfig{
					WorkerCount: 0,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestGetEnvHelpers(t *testing.T) {
	t.Run("getEnvInt", func(t *testing.T) {
		os.Setenv("TEST_INT", "42")
		defer os.Unsetenv("TEST_INT")

		result := getEnvInt("TEST_INT", 10)
		if result != 42 {
			t.Errorf("Expected 42, got %d", result)
		}

		result = getEnvInt("NONEXISTENT", 10)
		if result != 10 {
			t.Errorf("Expected default 10, got %d", result)
		}
	})

	t.Run("getEnvBool", func(t *testing.T) {
		os.Setenv("TEST_BOOL", "true")
		defer os.Unsetenv("TEST_BOOL")

		result := getEnvBool("TEST_BOOL", false)
		if !result {
			t.Errorf("Expected true, got %v", result)
		}

		result = getEnvBool("NONEXISTENT", false)
		if result {
			t.Errorf("Expected default false, got %v", result)
		}
	})

	t.Run("getEnvDuration", func(t *testing.T) {
		os.Setenv("TEST_DURATION", "5m")
		defer os.Unsetenv("TEST_DURATION")

		result := getEnvDuration("TEST_DURATION", time.Minute)
		if result != 5*time.Minute {
			t.Errorf("Expected 5m, got %v", result)
		}

		result = getEnvDuration("NONEXISTENT", time.Minute)
		if result != time.Minute {
			t.Errorf("Expected default 1m, got %v", result)
		}
	})
}
