package pricingclient

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pricingv1 "github.com/julimeimei/grpc-duo-go/gen/go/pricing/v1"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Price struct {
	ProductID       string
	AmountCents     int64
	Currency        string
	DiscountPercent int32
}

type Client struct {
	client       pricingv1.PricingServiceClient
	healthClient healthv1.HealthClient
	deadline     time.Duration
	logger       *slog.Logger
}

func New(conn grpc.ClientConnInterface, deadline time.Duration, loggers ...*slog.Logger) *Client {
	logger := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}

	return &Client{
		client:       pricingv1.NewPricingServiceClient(conn),
		healthClient: healthv1.NewHealthClient(conn),
		deadline:     deadline,
		logger:       logger,
	}
}

func (c *Client) GetPrice(ctx context.Context, productID string) (Price, error) {
	ctx, cancel := c.prepareContext(ctx)
	defer cancel()

	startedAt := time.Now()
	requestID := observability.RequestIDFromContext(ctx)
	logger := c.safeLogger()
	logger.InfoContext(ctx, "pricing grpc call started",
		slog.String("request_id", requestID),
		slog.String("method", "/pricing.v1.PricingService/GetPrice"),
		slog.String("product_id", productID),
	)

	resp, err := c.client.GetPrice(ctx, &pricingv1.GetPriceRequest{
		ProductId: productID,
	})
	if err != nil {
		logger.WarnContext(ctx, "pricing grpc call failed",
			slog.String("request_id", requestID),
			slog.String("method", "/pricing.v1.PricingService/GetPrice"),
			slog.String("code", status.Code(err).String()),
			slog.Duration("duration", time.Since(startedAt)),
		)
		return Price{}, err
	}

	logger.InfoContext(ctx, "pricing grpc call completed",
		slog.String("request_id", requestID),
		slog.String("method", "/pricing.v1.PricingService/GetPrice"),
		slog.String("code", codes.OK.String()),
		slog.Duration("duration", time.Since(startedAt)),
	)

	return Price{
		ProductID:       resp.GetProductId(),
		AmountCents:     resp.GetAmountCents(),
		Currency:        resp.GetCurrency(),
		DiscountPercent: resp.GetDiscountPercent(),
	}, nil
}

func (c *Client) CheckReadiness(ctx context.Context) error {
	ctx, cancel := c.prepareContext(ctx)
	defer cancel()

	resp, err := c.healthClient.Check(ctx, &healthv1.HealthCheckRequest{
		Service: pricingv1.PricingService_ServiceDesc.ServiceName,
	})
	if err != nil {
		return err
	}
	if resp.GetStatus() != healthv1.HealthCheckResponse_SERVING {
		return status.Error(codes.Unavailable, fmt.Sprintf("pricing health status is %s", resp.GetStatus()))
	}

	return nil
}

func (c *Client) safeLogger() *slog.Logger {
	if c.logger == nil {
		return slog.Default()
	}

	return c.logger
}

func (c *Client) prepareContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.deadline > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.deadline)
		ctx = appendRequestIDMetadata(ctx)
		return ctx, cancel
	}

	ctx = appendRequestIDMetadata(ctx)
	return ctx, func() {}
}

func appendRequestIDMetadata(ctx context.Context) context.Context {
	requestID := observability.RequestIDFromContext(ctx)
	if requestID == "" {
		return ctx
	}

	return metadata.AppendToOutgoingContext(ctx, observability.RequestIDKey, requestID)
}
