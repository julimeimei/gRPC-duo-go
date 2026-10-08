package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoadUsesDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, defaultHTTPAddr)
	}
	if cfg.ReadTimeout != defaultReadTimeout {
		t.Fatalf("ReadTimeout = %v, want %v", cfg.ReadTimeout, defaultReadTimeout)
	}
	if cfg.WriteTimeout != defaultWriteTimeout {
		t.Fatalf("WriteTimeout = %v, want %v", cfg.WriteTimeout, defaultWriteTimeout)
	}
	if cfg.IdleTimeout != defaultIdleTimeout {
		t.Fatalf("IdleTimeout = %v, want %v", cfg.IdleTimeout, defaultIdleTimeout)
	}
	if cfg.MaxHeaderBytes != defaultMaxHeaderBytes {
		t.Fatalf("MaxHeaderBytes = %d, want %d", cfg.MaxHeaderBytes, defaultMaxHeaderBytes)
	}
	if cfg.PricingGRPCAddr != defaultPricingGRPCAddr {
		t.Fatalf("PricingGRPCAddr = %q, want %q", cfg.PricingGRPCAddr, defaultPricingGRPCAddr)
	}
	if cfg.PricingGRPCDeadline != defaultPricingGRPCDeadline {
		t.Fatalf("PricingGRPCDeadline = %v, want %v", cfg.PricingGRPCDeadline, defaultPricingGRPCDeadline)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
}

func TestLoadReadsEnvironment(t *testing.T) {
	clearEnv(t)
	t.Setenv("CATALOG_HTTP_ADDR", "127.0.0.1:8181")
	t.Setenv("CATALOG_READ_TIMEOUT", "2s")
	t.Setenv("CATALOG_WRITE_TIMEOUT", "3s")
	t.Setenv("CATALOG_IDLE_TIMEOUT", "4s")
	t.Setenv("CATALOG_MAX_HEADER_BYTES", "32768")
	t.Setenv("CATALOG_PRICING_GRPC_ADDR", "127.0.0.1:9191")
	t.Setenv("CATALOG_PRICING_GRPC_DEADLINE", "500ms")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != "127.0.0.1:8181" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "127.0.0.1:8181")
	}
	if cfg.ReadTimeout != 2*time.Second {
		t.Fatalf("ReadTimeout = %v, want %v", cfg.ReadTimeout, 2*time.Second)
	}
	if cfg.WriteTimeout != 3*time.Second {
		t.Fatalf("WriteTimeout = %v, want %v", cfg.WriteTimeout, 3*time.Second)
	}
	if cfg.IdleTimeout != 4*time.Second {
		t.Fatalf("IdleTimeout = %v, want %v", cfg.IdleTimeout, 4*time.Second)
	}
	if cfg.MaxHeaderBytes != 32768 {
		t.Fatalf("MaxHeaderBytes = %d, want %d", cfg.MaxHeaderBytes, 32768)
	}
	if cfg.PricingGRPCAddr != "127.0.0.1:9191" {
		t.Fatalf("PricingGRPCAddr = %q, want %q", cfg.PricingGRPCAddr, "127.0.0.1:9191")
	}
	if cfg.PricingGRPCDeadline != 500*time.Millisecond {
		t.Fatalf("PricingGRPCDeadline = %v, want %v", cfg.PricingGRPCDeadline, 500*time.Millisecond)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	clearEnv(t)
	t.Setenv("CATALOG_READ_TIMEOUT", "forever")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadRejectsNonPositiveDuration(t *testing.T) {
	clearEnv(t)
	t.Setenv("CATALOG_PRICING_GRPC_DEADLINE", "0s")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadRejectsInvalidMaxHeaderBytes(t *testing.T) {
	tests := []string{"zero", "0", "-1"}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("CATALOG_MAX_HEADER_BYTES", value)

			if _, err := Load(); err == nil {
				t.Fatal("Load() error = nil, want error")
			}
		})
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	clearEnv(t)
	t.Setenv("LOG_LEVEL", "trace")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func clearEnv(t *testing.T) {
	t.Helper()

	for _, name := range []string{
		"CATALOG_HTTP_ADDR",
		"CATALOG_READ_TIMEOUT",
		"CATALOG_WRITE_TIMEOUT",
		"CATALOG_IDLE_TIMEOUT",
		"CATALOG_MAX_HEADER_BYTES",
		"CATALOG_PRICING_GRPC_ADDR",
		"CATALOG_PRICING_GRPC_DEADLINE",
		"LOG_LEVEL",
	} {
		t.Setenv(name, "")
	}
}
