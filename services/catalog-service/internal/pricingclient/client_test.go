package pricingclient

import (
	"context"
	"errors"
	"testing"
	"time"

	pricingv1 "github.com/julimeimei/grpc-duo-go/gen/go/pricing/v1"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGetPrice(t *testing.T) {
	client := &Client{
		client: fakePricingClient{
			resp: &pricingv1.GetPriceResponse{
				ProductId:       "42",
				AmountCents:     12990,
				Currency:        "BRL",
				DiscountPercent: 10,
			},
		},
		deadline: time.Second,
	}

	price, err := client.GetPrice(context.Background(), "42")
	if err != nil {
		t.Fatalf("GetPrice() error = %v", err)
	}

	if price.ProductID != "42" {
		t.Fatalf("ProductID = %q, want %q", price.ProductID, "42")
	}
	if price.AmountCents != 12990 {
		t.Fatalf("AmountCents = %d, want %d", price.AmountCents, 12990)
	}
}

func TestGetPriceReturnsClientError(t *testing.T) {
	wantErr := errors.New("grpc failed")
	client := &Client{
		client: fakePricingClient{err: wantErr},
	}

	_, err := client.GetPrice(context.Background(), "42")
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetPrice() error = %v, want %v", err, wantErr)
	}
}

func TestGetPriceAppliesDeadline(t *testing.T) {
	deadlineSeen := false
	client := &Client{
		client: fakePricingClient{
			assert: func(ctx context.Context) {
				if _, ok := ctx.Deadline(); !ok {
					return
				}
				deadlineSeen = true
			},
		},
		deadline: time.Second,
	}

	_, err := client.GetPrice(context.Background(), "42")
	if err != nil {
		t.Fatalf("GetPrice() error = %v", err)
	}

	if !deadlineSeen {
		t.Fatal("context has no deadline")
	}
}

func TestGetPricePropagatesRequestIDMetadata(t *testing.T) {
	metadataSeen := false
	client := &Client{
		client: fakePricingClient{
			assert: func(ctx context.Context) {
				md, ok := metadata.FromOutgoingContext(ctx)
				if !ok {
					return
				}
				values := md.Get(observability.RequestIDKey)
				metadataSeen = len(values) == 1 && values[0] == "test-request-123"
			},
		},
	}
	ctx := observability.ContextWithRequestID(context.Background(), "test-request-123")

	if _, err := client.GetPrice(ctx, "42"); err != nil {
		t.Fatalf("GetPrice() error = %v", err)
	}

	if !metadataSeen {
		t.Fatal("request id metadata was not propagated")
	}
}

func TestCheckReadiness(t *testing.T) {
	client := &Client{
		healthClient: fakeHealthClient{
			resp: &healthv1.HealthCheckResponse{Status: healthv1.HealthCheckResponse_SERVING},
		},
		deadline: time.Second,
	}

	if err := client.CheckReadiness(context.Background()); err != nil {
		t.Fatalf("CheckReadiness() error = %v", err)
	}
}

func TestCheckReadinessReturnsClientError(t *testing.T) {
	wantErr := errors.New("grpc health failed")
	client := &Client{
		healthClient: fakeHealthClient{err: wantErr},
	}

	err := client.CheckReadiness(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("CheckReadiness() error = %v, want %v", err, wantErr)
	}
}

func TestCheckReadinessRejectsNotServingStatus(t *testing.T) {
	client := &Client{
		healthClient: fakeHealthClient{
			resp: &healthv1.HealthCheckResponse{Status: healthv1.HealthCheckResponse_NOT_SERVING},
		},
	}

	err := client.CheckReadiness(context.Background())
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("status.Code(error) = %v, want %v", status.Code(err), codes.Unavailable)
	}
}

func TestCheckReadinessAppliesDeadline(t *testing.T) {
	deadlineSeen := false
	client := &Client{
		healthClient: fakeHealthClient{
			assert: func(ctx context.Context) {
				if _, ok := ctx.Deadline(); !ok {
					return
				}
				deadlineSeen = true
			},
			resp: &healthv1.HealthCheckResponse{Status: healthv1.HealthCheckResponse_SERVING},
		},
		deadline: time.Second,
	}

	if err := client.CheckReadiness(context.Background()); err != nil {
		t.Fatalf("CheckReadiness() error = %v", err)
	}

	if !deadlineSeen {
		t.Fatal("context has no deadline")
	}
}

type fakePricingClient struct {
	resp   *pricingv1.GetPriceResponse
	err    error
	assert func(context.Context)
}

type fakeHealthClient struct {
	resp   *healthv1.HealthCheckResponse
	err    error
	assert func(context.Context)
}

func (f fakeHealthClient) Check(ctx context.Context, _ *healthv1.HealthCheckRequest, _ ...grpc.CallOption) (*healthv1.HealthCheckResponse, error) {
	if f.assert != nil {
		f.assert(ctx)
	}
	if f.err != nil {
		return nil, f.err
	}
	if f.resp != nil {
		return f.resp, nil
	}
	return &healthv1.HealthCheckResponse{}, nil
}

func (f fakeHealthClient) Watch(context.Context, *healthv1.HealthCheckRequest, ...grpc.CallOption) (healthv1.Health_WatchClient, error) {
	return nil, errors.New("watch is not implemented in tests")
}

func (f fakeHealthClient) List(context.Context, *healthv1.HealthListRequest, ...grpc.CallOption) (*healthv1.HealthListResponse, error) {
	return nil, errors.New("list is not implemented in tests")
}

func (f fakePricingClient) GetPrice(ctx context.Context, _ *pricingv1.GetPriceRequest, _ ...grpc.CallOption) (*pricingv1.GetPriceResponse, error) {
	if f.assert != nil {
		f.assert(ctx)
	}
	if f.err != nil {
		return nil, f.err
	}
	if f.resp != nil {
		return f.resp, nil
	}
	return &pricingv1.GetPriceResponse{}, nil
}
