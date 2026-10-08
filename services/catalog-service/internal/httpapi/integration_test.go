package httpapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	pricingv1 "github.com/julimeimei/grpc-duo-go/gen/go/pricing/v1"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/observability"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/pricingclient"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/product"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

func TestGetProductCallsPricingServiceOverGRPC(t *testing.T) {
	const requestID = "test-request-123"

	pricingServer := &recordingPricingServer{}
	conn := newBufconnPricingClient(t, pricingServer)

	products, err := product.NewInMemoryService(product.DefaultProducts())
	if err != nil {
		t.Fatalf("NewInMemoryService() error = %v", err)
	}

	prices := pricingclient.New(conn, time.Second, testLogger())
	handler := NewHandler(products, prices, prices, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/products/42", nil)
	req.Header.Set(observability.RequestIDHeader, requestID)
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if got := rec.Header().Get(observability.RequestIDHeader); got != requestID {
		t.Fatalf("response request id = %q, want %q", got, requestID)
	}

	productID, propagatedRequestID := pricingServer.snapshot()
	if productID != "42" {
		t.Fatalf("pricing request product id = %q, want %q", productID, "42")
	}
	if propagatedRequestID != requestID {
		t.Fatalf("pricing metadata request id = %q, want %q", propagatedRequestID, requestID)
	}

	var body productResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if body.ID != "42" {
		t.Fatalf("ID = %q, want %q", body.ID, "42")
	}
	if body.Name != "Wireless Mouse" {
		t.Fatalf("Name = %q, want %q", body.Name, "Wireless Mouse")
	}
	if body.Price.AmountCents != 12990 {
		t.Fatalf("AmountCents = %d, want %d", body.Price.AmountCents, 12990)
	}
	if body.Price.Currency != "BRL" {
		t.Fatalf("Currency = %q, want %q", body.Price.Currency, "BRL")
	}
	if body.Price.DiscountPercent != 10 {
		t.Fatalf("DiscountPercent = %d, want %d", body.Price.DiscountPercent, 10)
	}
}

func newBufconnPricingClient(t *testing.T, pricingServer pricingv1.PricingServiceServer) *grpc.ClientConn {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	pricingv1.RegisterPricingServiceServer(server, pricingServer)

	go func() {
		if err := server.Serve(listener); err != nil {
			t.Logf("bufconn grpc server stopped: %v", err)
		}
	}()

	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}

	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Fatalf("conn.Close() error = %v", err)
		}
	})

	return conn
}

type recordingPricingServer struct {
	pricingv1.UnimplementedPricingServiceServer

	mu        sync.Mutex
	productID string
	requestID string
}

func (s *recordingPricingServer) GetPrice(ctx context.Context, req *pricingv1.GetPriceRequest) (*pricingv1.GetPriceResponse, error) {
	requestID := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(observability.RequestIDKey)
		if len(values) > 0 {
			requestID = values[0]
		}
	}

	s.mu.Lock()
	s.productID = req.GetProductId()
	s.requestID = requestID
	s.mu.Unlock()

	return &pricingv1.GetPriceResponse{
		ProductId:       req.GetProductId(),
		AmountCents:     12990,
		Currency:        "BRL",
		DiscountPercent: 10,
	}, nil
}

func (s *recordingPricingServer) snapshot() (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.productID, s.requestID
}
