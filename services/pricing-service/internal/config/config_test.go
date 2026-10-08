package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("PRICING_GRPC_ADDR", "")
	t.Setenv("PRICING_DATABASE_URL", "")
	t.Setenv("PRICING_DATABASE_CONNECT_TIMEOUT", "")
	t.Setenv("PRICING_DATABASE_QUERY_TIMEOUT", "")
	t.Setenv("PRICING_DATABASE_MAX_OPEN_CONNS", "")
	t.Setenv("PRICING_DATABASE_MAX_IDLE_CONNS", "")
	t.Setenv("PRICING_DATABASE_CONN_MAX_LIFETIME", "")
	t.Setenv("PRICING_READINESS_CHECK_INTERVAL", "")
	t.Setenv("LOG_LEVEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.GRPCAddr != defaultGRPCAddr {
		t.Fatalf("GRPCAddr = %q, want %q", cfg.GRPCAddr, defaultGRPCAddr)
	}

	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}

	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Fatalf("DatabaseURL = %q, want default", cfg.DatabaseURL)
	}

	if cfg.DatabaseConnectTimeout != defaultDatabaseConnectTimeout {
		t.Fatalf("DatabaseConnectTimeout = %v, want %v", cfg.DatabaseConnectTimeout, defaultDatabaseConnectTimeout)
	}

	if cfg.DatabaseQueryTimeout != defaultDatabaseQueryTimeout {
		t.Fatalf("DatabaseQueryTimeout = %v, want %v", cfg.DatabaseQueryTimeout, defaultDatabaseQueryTimeout)
	}

	if cfg.DatabaseMaxOpenConns != defaultDatabaseMaxOpenConns {
		t.Fatalf("DatabaseMaxOpenConns = %d, want %d", cfg.DatabaseMaxOpenConns, defaultDatabaseMaxOpenConns)
	}
}

func TestLoadReadsEnvironment(t *testing.T) {
	t.Setenv("PRICING_GRPC_ADDR", "127.0.0.1:9191")
	t.Setenv("PRICING_DATABASE_URL", "postgres://example:secret@localhost:5432/example?sslmode=disable")
	t.Setenv("PRICING_DATABASE_CONNECT_TIMEOUT", "3s")
	t.Setenv("PRICING_DATABASE_QUERY_TIMEOUT", "750ms")
	t.Setenv("PRICING_DATABASE_MAX_OPEN_CONNS", "8")
	t.Setenv("PRICING_DATABASE_MAX_IDLE_CONNS", "4")
	t.Setenv("PRICING_DATABASE_CONN_MAX_LIFETIME", "15m")
	t.Setenv("PRICING_READINESS_CHECK_INTERVAL", "2s")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.GRPCAddr != "127.0.0.1:9191" {
		t.Fatalf("GRPCAddr = %q, want %q", cfg.GRPCAddr, "127.0.0.1:9191")
	}

	if cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}

	if cfg.DatabaseURL != "postgres://example:secret@localhost:5432/example?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q, want configured value", cfg.DatabaseURL)
	}

	if cfg.DatabaseConnectTimeout != 3*time.Second {
		t.Fatalf("DatabaseConnectTimeout = %v, want 3s", cfg.DatabaseConnectTimeout)
	}

	if cfg.DatabaseQueryTimeout != 750*time.Millisecond {
		t.Fatalf("DatabaseQueryTimeout = %v, want 750ms", cfg.DatabaseQueryTimeout)
	}

	if cfg.DatabaseMaxOpenConns != 8 {
		t.Fatalf("DatabaseMaxOpenConns = %d, want 8", cfg.DatabaseMaxOpenConns)
	}

	if cfg.DatabaseMaxIdleConns != 4 {
		t.Fatalf("DatabaseMaxIdleConns = %d, want 4", cfg.DatabaseMaxIdleConns)
	}

	if cfg.DatabaseConnMaxLifetime != 15*time.Minute {
		t.Fatalf("DatabaseConnMaxLifetime = %v, want 15m", cfg.DatabaseConnMaxLifetime)
	}

	if cfg.ReadinessCheckInterval != 2*time.Second {
		t.Fatalf("ReadinessCheckInterval = %v, want 2s", cfg.ReadinessCheckInterval)
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "trace")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadRejectsInvalidDatabaseConfig(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "invalid connect timeout",
			env:  map[string]string{"PRICING_DATABASE_CONNECT_TIMEOUT": "forever"},
		},
		{
			name: "zero query timeout",
			env:  map[string]string{"PRICING_DATABASE_QUERY_TIMEOUT": "0s"},
		},
		{
			name: "invalid max open connections",
			env:  map[string]string{"PRICING_DATABASE_MAX_OPEN_CONNS": "many"},
		},
		{
			name: "zero max idle connections",
			env:  map[string]string{"PRICING_DATABASE_MAX_IDLE_CONNS": "0"},
		},
		{
			name: "max idle greater than max open",
			env: map[string]string{
				"PRICING_DATABASE_MAX_OPEN_CONNS": "2",
				"PRICING_DATABASE_MAX_IDLE_CONNS": "3",
			},
		},
		{
			name: "invalid connection lifetime",
			env:  map[string]string{"PRICING_DATABASE_CONN_MAX_LIFETIME": "-1s"},
		},
		{
			name: "invalid readiness interval",
			env:  map[string]string{"PRICING_READINESS_CHECK_INTERVAL": "0s"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			if _, err := Load(); err == nil {
				t.Fatal("Load() error = nil, want error")
			}
		})
	}
}
