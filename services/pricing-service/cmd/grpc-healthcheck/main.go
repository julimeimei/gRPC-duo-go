package main

import (
	"context"
	"fmt"
	"os"
	"time"

	pricingv1 "github.com/julimeimei/grpc-duo-go/gen/go/pricing/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

const timeout = 2 * time.Second

func main() {
	target := "127.0.0.1:9090"
	if len(os.Args) > 1 {
		target = os.Args[1]
	}
	service := pricingv1.PricingService_ServiceDesc.ServiceName
	if len(os.Args) > 2 {
		service = os.Args[2]
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "create grpc client: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	resp, err := healthv1.NewHealthClient(conn).Check(ctx, &healthv1.HealthCheckRequest{Service: service})
	if err != nil {
		fmt.Fprintf(os.Stderr, "health check failed: %v\n", err)
		os.Exit(1)
	}
	if resp.GetStatus() != healthv1.HealthCheckResponse_SERVING {
		fmt.Fprintf(os.Stderr, "health status is %s\n", resp.GetStatus())
		os.Exit(1)
	}
}
