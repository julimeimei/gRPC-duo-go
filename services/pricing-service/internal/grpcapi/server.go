package grpcapi

import (
	"context"
	"errors"
	"log/slog"

	pricingv1 "github.com/julimeimei/grpc-duo-go/gen/go/pricing/v1"
	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/observability"
	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/pricing"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PriceService interface {
	GetPrice(ctx context.Context, productID string) (pricing.Price, error)
}

type Server struct {
	pricingv1.UnimplementedPricingServiceServer

	prices PriceService
	logger *slog.Logger
}

func NewServer(prices PriceService, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		prices: prices,
		logger: logger,
	}
}

func (s *Server) GetPrice(ctx context.Context, req *pricingv1.GetPriceRequest) (*pricingv1.GetPriceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if s.prices == nil {
		s.logger.Error("pricing service dependency is nil")
		return nil, status.Error(codes.Internal, "internal pricing error")
	}

	requestID := observability.RequestIDFromContext(ctx)
	s.logger.InfoContext(ctx, "pricing get_price started",
		slog.String("request_id", requestID),
		slog.String("product_id", req.GetProductId()),
	)

	price, err := s.prices.GetPrice(ctx, req.GetProductId())
	if err != nil {
		return nil, s.mapError(err)
	}

	s.logger.InfoContext(ctx, "pricing get_price completed",
		slog.String("request_id", requestID),
		slog.String("product_id", price.ProductID),
	)

	return &pricingv1.GetPriceResponse{
		ProductId:       price.ProductID,
		AmountCents:     price.AmountCents,
		Currency:        price.Currency,
		DiscountPercent: price.DiscountPercent,
	}, nil
}

func (s *Server) mapError(err error) error {
	switch {
	case errors.Is(err, pricing.ErrInvalidProductID):
		return status.Error(codes.InvalidArgument, "product_id is invalid")
	case errors.Is(err, pricing.ErrNotFound):
		return status.Error(codes.NotFound, "price not found")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request deadline exceeded")
	default:
		s.logger.Error("unexpected pricing service error", slog.String("error", err.Error()))
		return status.Error(codes.Internal, "internal pricing error")
	}
}
