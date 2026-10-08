package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/observability"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/pricingclient"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/product"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetProduct(t *testing.T) {
	handler := NewHandler(
		fakeProductService{item: product.Product{ID: "42", Name: "Wireless Mouse", Description: "Ergonomic wireless mouse"}},
		fakePricingService{price: pricingclient.Price{ProductID: "42", AmountCents: 12990, Currency: "BRL", DiscountPercent: 10}},
		fakeReadinessChecker{},
		testLogger(),
	)

	req := httptest.NewRequest(http.MethodGet, "/products/42", nil)
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body productResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if body.ID != "42" {
		t.Fatalf("ID = %q, want %q", body.ID, "42")
	}
	if body.Price.AmountCents != 12990 {
		t.Fatalf("AmountCents = %d, want %d", body.Price.AmountCents, 12990)
	}
}

func TestGetProductMapsProductErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid product id", err: product.ErrInvalidProductID, wantStatus: http.StatusBadRequest, wantCode: "invalid_product_id"},
		{name: "product not found", err: product.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "product_not_found"},
		{name: "unexpected", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(
				fakeProductService{err: tt.err},
				fakePricingService{},
				fakeReadinessChecker{},
				testLogger(),
			)

			req := httptest.NewRequest(http.MethodGet, "/products/42", nil)
			rec := httptest.NewRecorder()

			handler.Routes().ServeHTTP(rec, req)

			assertError(t, rec, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestGetProductMapsPricingErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid argument", err: status.Error(codes.InvalidArgument, "bad id"), wantStatus: http.StatusBadRequest, wantCode: "invalid_product_id"},
		{name: "not found", err: status.Error(codes.NotFound, "missing price"), wantStatus: http.StatusNotFound, wantCode: "price_not_found"},
		{name: "deadline", err: status.Error(codes.DeadlineExceeded, "slow"), wantStatus: http.StatusGatewayTimeout, wantCode: "pricing_timeout"},
		{name: "unavailable", err: status.Error(codes.Unavailable, "down"), wantStatus: http.StatusServiceUnavailable, wantCode: "pricing_unavailable"},
		{name: "unexpected", err: status.Error(codes.Internal, "boom"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(
				fakeProductService{item: product.Product{ID: "42", Name: "Wireless Mouse"}},
				fakePricingService{err: tt.err},
				fakeReadinessChecker{},
				testLogger(),
			)

			req := httptest.NewRequest(http.MethodGet, "/products/42", nil)
			rec := httptest.NewRecorder()

			handler.Routes().ServeHTTP(rec, req)

			assertError(t, rec, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestGetProductRequiresDependencies(t *testing.T) {
	handler := NewHandler(nil, nil, nil, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/products/42", nil)
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	assertError(t, rec, http.StatusInternalServerError, "internal_error")
}

func TestHealth(t *testing.T) {
	handler := NewHandler(nil, nil, nil, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("status body = %q, want ok", body.Status)
	}
}

func TestRoutesPreservesRequestIDHeader(t *testing.T) {
	handler := NewHandler(nil, nil, nil, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(observability.RequestIDHeader, "test-request-123")
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if got := rec.Header().Get(observability.RequestIDHeader); got != "test-request-123" {
		t.Fatalf("response request id = %q, want %q", got, "test-request-123")
	}
}

func TestRoutesGeneratesRequestIDWhenHeaderIsInvalid(t *testing.T) {
	handler := NewHandler(nil, nil, nil, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(observability.RequestIDHeader, "../../bad id")
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	requestID := rec.Header().Get(observability.RequestIDHeader)
	if requestID == "" {
		t.Fatal("response request id is empty")
	}
	if requestID == "../../bad id" {
		t.Fatal("invalid request id was preserved")
	}
}

func TestReady(t *testing.T) {
	handler := NewHandler(nil, nil, fakeReadinessChecker{}, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body readinessResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if body.Status != "ready" {
		t.Fatalf("status body = %q, want ready", body.Status)
	}
	if body.Dependencies["pricing-service"] != "ready" {
		t.Fatalf("pricing dependency = %q, want ready", body.Dependencies["pricing-service"])
	}
}

func TestReadyReturnsUnavailableWhenPricingIsUnavailable(t *testing.T) {
	handler := NewHandler(nil, nil, fakeReadinessChecker{err: errors.New("pricing down")}, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}

	var body readinessResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if body.Status != "not_ready" {
		t.Fatalf("status body = %q, want not_ready", body.Status)
	}
	if body.Dependencies["pricing-service"] != "unavailable" {
		t.Fatalf("pricing dependency = %q, want unavailable", body.Dependencies["pricing-service"])
	}
}

func TestReadyRequiresDependency(t *testing.T) {
	handler := NewHandler(nil, nil, nil, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, wantStatus, rec.Body.String())
	}

	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if body.Error.Code != wantCode {
		t.Fatalf("error code = %q, want %q", body.Error.Code, wantCode)
	}
}

type fakeProductService struct {
	item product.Product
	err  error
}

func (f fakeProductService) GetProduct(context.Context, string) (product.Product, error) {
	return f.item, f.err
}

type fakePricingService struct {
	price pricingclient.Price
	err   error
}

func (f fakePricingService) GetPrice(context.Context, string) (pricingclient.Price, error) {
	return f.price, f.err
}

type fakeReadinessChecker struct {
	err error
}

func (f fakeReadinessChecker) CheckReadiness(context.Context) error {
	return f.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
