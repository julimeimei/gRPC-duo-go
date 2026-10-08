package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/config"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/httpapi"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/observability"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/pricingclient"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/product"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("catalog-service failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := observability.NewLogger(cfg.LogLevel)

	productService, err := product.NewInMemoryService(product.DefaultProducts())
	if err != nil {
		return fmt.Errorf("create in-memory product service: %w", err)
	}

	conn, err := grpc.NewClient(
		cfg.PricingGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("create pricing-service grpc client: %w", err)
	}
	defer conn.Close()

	pricingClient := pricingclient.New(conn, cfg.PricingGRPCDeadline, logger)

	handler := httpapi.NewHandler(productService, pricingClient, pricingClient, logger)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler.Routes(),
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("catalog-service http server started", slog.String("addr", cfg.HTTPAddr))
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve http: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		logger.Info("catalog-service http server stopped gracefully")
		return nil
	}
}
