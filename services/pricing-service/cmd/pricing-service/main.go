package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pricingv1 "github.com/julimeimei/grpc-duo-go/gen/go/pricing/v1"
	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/config"
	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/grpcapi"
	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/observability"
	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/postgres"
	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/pricing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("pricing-service failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := observability.NewLogger(cfg.LogLevel)

	dbCtx, cancel := context.WithTimeout(context.Background(), cfg.DatabaseConnectTimeout)
	defer cancel()

	priceRepository, err := postgres.Open(dbCtx, postgres.Config{
		DatabaseURL:     cfg.DatabaseURL,
		MaxOpenConns:    cfg.DatabaseMaxOpenConns,
		MaxIdleConns:    cfg.DatabaseMaxIdleConns,
		ConnMaxLifetime: cfg.DatabaseConnMaxLifetime,
		Logger:          logger,
	})
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer func() {
		if err := priceRepository.Close(); err != nil {
			logger.Error("close postgres pool", slog.String("error", err.Error()))
		}
	}()

	priceService, err := pricing.NewService(priceRepository, cfg.DatabaseQueryTimeout)
	if err != nil {
		return fmt.Errorf("create price service: %w", err)
	}

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.GRPCAddr, err)
	}
	defer listener.Close()

	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpcapi.LoggingUnaryInterceptor(logger)),
	)
	pricingv1.RegisterPricingServiceServer(server, grpcapi.NewServer(priceService, logger))
	healthServer := health.NewServer()
	healthv1.RegisterHealthServer(server, healthServer)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	setHealthStatus(healthServer, healthv1.HealthCheckResponse_SERVING)
	go monitorDatabaseReadiness(ctx, priceRepository, healthServer, cfg.ReadinessCheckInterval, logger)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("pricing-service grpc server started", slog.String("addr", cfg.GRPCAddr))
		if err := server.Serve(listener); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return fmt.Errorf("serve grpc: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		shutdownGRPCServer(shutdownCtx, server, logger)
		return nil
	}
}

type readinessProbe interface {
	Ping(ctx context.Context) error
}

func monitorDatabaseReadiness(ctx context.Context, probe readinessProbe, healthServer *health.Server, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	checkDatabaseReadiness(ctx, probe, healthServer, interval, logger)

	for {
		select {
		case <-ticker.C:
			checkDatabaseReadiness(ctx, probe, healthServer, interval, logger)
		case <-ctx.Done():
			setHealthStatus(healthServer, healthv1.HealthCheckResponse_NOT_SERVING)
			return
		}
	}
}

func checkDatabaseReadiness(ctx context.Context, probe readinessProbe, healthServer *health.Server, timeout time.Duration, logger *slog.Logger) {
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	err := probe.Ping(checkCtx)
	cancel()

	if err != nil {
		setHealthStatus(healthServer, healthv1.HealthCheckResponse_NOT_SERVING)
		logger.Error("postgres readiness check failed", slog.String("error", err.Error()))
		return
	}

	setHealthStatus(healthServer, healthv1.HealthCheckResponse_SERVING)
}

func setHealthStatus(healthServer *health.Server, status healthv1.HealthCheckResponse_ServingStatus) {
	healthServer.SetServingStatus("", status)
	healthServer.SetServingStatus(pricingv1.PricingService_ServiceDesc.ServiceName, status)
}

func shutdownGRPCServer(ctx context.Context, server *grpc.Server, logger *slog.Logger) {
	stopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		logger.Info("pricing-service grpc server stopped gracefully")
	case <-ctx.Done():
		logger.Warn("graceful shutdown timed out, forcing grpc server stop")
		server.Stop()
		<-stopped
	}
}
