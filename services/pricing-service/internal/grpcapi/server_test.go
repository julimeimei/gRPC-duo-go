package grpcapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	pricingv1 "github.com/julimeimei/grpc-duo-go/gen/go/pricing/v1"
	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/observability"
	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/pricing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGetPrice(t *testing.T) {
	server := NewServer(fakePriceService{
		price: pricing.Price{
			ProductID:       "42",
			AmountCents:     12990,
			Currency:        "BRL",
			DiscountPercent: 10,
		},
	}, testLogger())

	resp, err := server.GetPrice(context.Background(), &pricingv1.GetPriceRequest{ProductId: "42"})
	if err != nil {
		t.Fatalf("GetPrice() error = %v", err)
	}

	if resp.GetProductId() != "42" {
		t.Fatalf("ProductId = %q, want %q", resp.GetProductId(), "42")
	}

	if resp.GetAmountCents() != 12990 {
		t.Fatalf("AmountCents = %d, want %d", resp.GetAmountCents(), 12990)
	}
}

func TestGetPriceMapsExpectedErrors(t *testing.T) {
	tests := []struct {
		name    string
		req     *pricingv1.GetPriceRequest
		want    codes.Code
		service PriceService
	}{
		{
			name:    "nil request",
			req:     nil,
			want:    codes.InvalidArgument,
			service: fakePriceService{},
		},
		{
			name:    "invalid product id",
			req:     &pricingv1.GetPriceRequest{ProductId: ""},
			want:    codes.InvalidArgument,
			service: fakePriceService{err: pricing.ErrInvalidProductID},
		},
		{
			name:    "not found",
			req:     &pricingv1.GetPriceRequest{ProductId: "999"},
			want:    codes.NotFound,
			service: fakePriceService{err: pricing.ErrNotFound},
		},
		{
			name:    "deadline exceeded",
			req:     &pricingv1.GetPriceRequest{ProductId: "42"},
			want:    codes.DeadlineExceeded,
			service: fakePriceService{err: context.DeadlineExceeded},
		},
		{
			name:    "unexpected",
			req:     &pricingv1.GetPriceRequest{ProductId: "42"},
			want:    codes.Internal,
			service: fakePriceService{err: errors.New("repository exploded")},
		},
		{
			name:    "nil dependency",
			req:     &pricingv1.GetPriceRequest{ProductId: "42"},
			want:    codes.Internal,
			service: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.service, testLogger())

			_, err := server.GetPrice(context.Background(), tt.req)
			if err == nil {
				t.Fatal("GetPrice() error = nil, want error")
			}

			if got := status.Code(err); got != tt.want {
				t.Fatalf("status.Code(error) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoggingUnaryInterceptorAddsRequestIDToContext(t *testing.T) {
	interceptor := LoggingUnaryInterceptor(testLogger())
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(observability.RequestIDKey, "test-request-123"))

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/pricing.v1.PricingService/GetPrice"}, func(ctx context.Context, _ any) (any, error) {
		if got := observability.RequestIDFromContext(ctx); got != "test-request-123" {
			t.Fatalf("request id from context = %q, want %q", got, "test-request-123")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor() error = %v", err)
	}
}

type fakePriceService struct {
	price pricing.Price
	err   error
}

func (f fakePriceService) GetPrice(context.Context, string) (pricing.Price, error) {
	return f.price, f.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
