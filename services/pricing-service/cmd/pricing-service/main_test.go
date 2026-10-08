package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	pricingv1 "github.com/julimeimei/grpc-duo-go/gen/go/pricing/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestShutdownGRPCServerReturns(t *testing.T) {
	server := grpc.NewServer()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	shutdownGRPCServer(ctx, server, slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

func TestCheckDatabaseReadinessMarksServing(t *testing.T) {
	healthServer := health.NewServer()

	checkDatabaseReadiness(
		context.Background(),
		fakeReadinessProbe{},
		healthServer,
		time.Second,
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
	)

	resp, err := healthServer.Check(context.Background(), &healthv1.HealthCheckRequest{Service: pricingv1.PricingService_ServiceDesc.ServiceName})
	if err != nil {
		t.Fatalf("health Check() error = %v", err)
	}
	if resp.GetStatus() != healthv1.HealthCheckResponse_SERVING {
		t.Fatalf("health status = %v, want SERVING", resp.GetStatus())
	}
}

func TestCheckDatabaseReadinessMarksNotServing(t *testing.T) {
	healthServer := health.NewServer()

	checkDatabaseReadiness(
		context.Background(),
		fakeReadinessProbe{err: errors.New("postgres down")},
		healthServer,
		time.Second,
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
	)

	resp, err := healthServer.Check(context.Background(), &healthv1.HealthCheckRequest{Service: pricingv1.PricingService_ServiceDesc.ServiceName})
	if err != nil {
		t.Fatalf("health Check() error = %v", err)
	}
	if resp.GetStatus() != healthv1.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("health status = %v, want NOT_SERVING", resp.GetStatus())
	}
}

type fakeReadinessProbe struct {
	err error
}

func (f fakeReadinessProbe) Ping(context.Context) error {
	return f.err
}
