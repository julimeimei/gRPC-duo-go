package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPAddr            = ":8080"
	defaultReadTimeout         = 5 * time.Second
	defaultWriteTimeout        = 10 * time.Second
	defaultIdleTimeout         = 60 * time.Second
	defaultMaxHeaderBytes      = 1 << 20
	defaultPricingGRPCAddr     = "localhost:9090"
	defaultPricingGRPCDeadline = 2 * time.Second
	defaultLogLevel            = slog.LevelInfo
)

type Config struct {
	HTTPAddr            string
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	MaxHeaderBytes      int
	PricingGRPCAddr     string
	PricingGRPCDeadline time.Duration
	LogLevel            slog.Level
}

func Load() (Config, error) {
	readTimeout, err := parseDuration("CATALOG_READ_TIMEOUT", defaultReadTimeout)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := parseDuration("CATALOG_WRITE_TIMEOUT", defaultWriteTimeout)
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := parseDuration("CATALOG_IDLE_TIMEOUT", defaultIdleTimeout)
	if err != nil {
		return Config{}, err
	}

	pricingDeadline, err := parseDuration("CATALOG_PRICING_GRPC_DEADLINE", defaultPricingGRPCDeadline)
	if err != nil {
		return Config{}, err
	}

	maxHeaderBytes, err := parsePositiveInt("CATALOG_MAX_HEADER_BYTES", defaultMaxHeaderBytes)
	if err != nil {
		return Config{}, err
	}

	logLevel, err := parseLogLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:            getString("CATALOG_HTTP_ADDR", defaultHTTPAddr),
		ReadTimeout:         readTimeout,
		WriteTimeout:        writeTimeout,
		IdleTimeout:         idleTimeout,
		MaxHeaderBytes:      maxHeaderBytes,
		PricingGRPCAddr:     getString("CATALOG_PRICING_GRPC_ADDR", defaultPricingGRPCAddr),
		PricingGRPCDeadline: pricingDeadline,
		LogLevel:            logLevel,
	}, nil
}

func getString(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func parseDuration(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, value, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("invalid %s %q: duration must be positive", name, value)
	}

	return duration, nil
}

func parsePositiveInt(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, value, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("invalid %s %q: value must be positive", name, value)
	}

	return parsed, nil
}

func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return defaultLogLevel, nil
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid LOG_LEVEL %q", value)
	}
}
