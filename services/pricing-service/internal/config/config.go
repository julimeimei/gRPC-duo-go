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
	defaultGRPCAddr                = ":9090"
	defaultDatabaseURL             = "postgres://grpc_duo_app:grpc_duo_dev_password@localhost:5432/grpc_duo?sslmode=disable"
	defaultDatabaseConnectTimeout  = 5 * time.Second
	defaultDatabaseQueryTimeout    = 2 * time.Second
	defaultDatabaseMaxOpenConns    = 5
	defaultDatabaseMaxIdleConns    = 5
	defaultDatabaseConnMaxLifetime = 30 * time.Minute
	defaultReadinessCheckInterval  = 10 * time.Second
	defaultLogLevel                = slog.LevelInfo
)

type Config struct {
	GRPCAddr                string
	DatabaseURL             string
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
	DatabaseMaxOpenConns    int
	DatabaseMaxIdleConns    int
	DatabaseConnMaxLifetime time.Duration
	ReadinessCheckInterval  time.Duration
	LogLevel                slog.Level
}

func Load() (Config, error) {
	grpcAddr := strings.TrimSpace(os.Getenv("PRICING_GRPC_ADDR"))
	if grpcAddr == "" {
		grpcAddr = defaultGRPCAddr
	}

	databaseURL := strings.TrimSpace(os.Getenv("PRICING_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = defaultDatabaseURL
	}

	connectTimeout, err := parsePositiveDuration("PRICING_DATABASE_CONNECT_TIMEOUT", os.Getenv("PRICING_DATABASE_CONNECT_TIMEOUT"), defaultDatabaseConnectTimeout)
	if err != nil {
		return Config{}, err
	}

	queryTimeout, err := parsePositiveDuration("PRICING_DATABASE_QUERY_TIMEOUT", os.Getenv("PRICING_DATABASE_QUERY_TIMEOUT"), defaultDatabaseQueryTimeout)
	if err != nil {
		return Config{}, err
	}

	maxOpenConns, err := parsePositiveInt("PRICING_DATABASE_MAX_OPEN_CONNS", os.Getenv("PRICING_DATABASE_MAX_OPEN_CONNS"), defaultDatabaseMaxOpenConns)
	if err != nil {
		return Config{}, err
	}

	maxIdleConns, err := parsePositiveInt("PRICING_DATABASE_MAX_IDLE_CONNS", os.Getenv("PRICING_DATABASE_MAX_IDLE_CONNS"), defaultDatabaseMaxIdleConns)
	if err != nil {
		return Config{}, err
	}
	if maxIdleConns > maxOpenConns {
		return Config{}, fmt.Errorf("PRICING_DATABASE_MAX_IDLE_CONNS must be less than or equal to PRICING_DATABASE_MAX_OPEN_CONNS")
	}

	connMaxLifetime, err := parsePositiveDuration("PRICING_DATABASE_CONN_MAX_LIFETIME", os.Getenv("PRICING_DATABASE_CONN_MAX_LIFETIME"), defaultDatabaseConnMaxLifetime)
	if err != nil {
		return Config{}, err
	}

	readinessCheckInterval, err := parsePositiveDuration("PRICING_READINESS_CHECK_INTERVAL", os.Getenv("PRICING_READINESS_CHECK_INTERVAL"), defaultReadinessCheckInterval)
	if err != nil {
		return Config{}, err
	}

	logLevel, err := parseLogLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		GRPCAddr:                grpcAddr,
		DatabaseURL:             databaseURL,
		DatabaseConnectTimeout:  connectTimeout,
		DatabaseQueryTimeout:    queryTimeout,
		DatabaseMaxOpenConns:    maxOpenConns,
		DatabaseMaxIdleConns:    maxIdleConns,
		DatabaseConnMaxLifetime: connMaxLifetime,
		ReadinessCheckInterval:  readinessCheckInterval,
		LogLevel:                logLevel,
	}, nil
}

func parsePositiveDuration(name, value string, fallback time.Duration) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, value, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}

	return duration, nil
}

func parsePositiveInt(name, value string, fallback int) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, value, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
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
