package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/observability"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/pricingclient"
	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/product"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProductService interface {
	GetProduct(ctx context.Context, productID string) (product.Product, error)
}

type PricingService interface {
	GetPrice(ctx context.Context, productID string) (pricingclient.Price, error)
}

type ReadinessChecker interface {
	CheckReadiness(ctx context.Context) error
}

type Handler struct {
	products  ProductService
	prices    PricingService
	readiness ReadinessChecker
	logger    *slog.Logger
}

func NewHandler(products ProductService, prices PricingService, readiness ReadinessChecker, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		products:  products,
		prices:    prices,
		readiness: readiness,
		logger:    logger,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /ready", h.ready)
	mux.HandleFunc("GET /products/{id}", h.getProduct)
	return loggingMiddleware(h.logger, mux)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Service: "catalog-service",
	})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if h.readiness == nil {
		h.logger.ErrorContext(r.Context(), "catalog readiness dependency is nil")
		writeJSON(w, http.StatusServiceUnavailable, readinessResponse{
			Status:       "not_ready",
			Service:      "catalog-service",
			Dependencies: map[string]string{"pricing-service": "unavailable"},
		})
		return
	}

	if err := h.readiness.CheckReadiness(r.Context()); err != nil {
		h.logger.WarnContext(r.Context(), "pricing-service readiness check failed", slog.String("error", err.Error()))
		writeJSON(w, http.StatusServiceUnavailable, readinessResponse{
			Status:       "not_ready",
			Service:      "catalog-service",
			Dependencies: map[string]string{"pricing-service": "unavailable"},
		})
		return
	}

	writeJSON(w, http.StatusOK, readinessResponse{
		Status:       "ready",
		Service:      "catalog-service",
		Dependencies: map[string]string{"pricing-service": "ready"},
	})
}

func (h *Handler) getProduct(w http.ResponseWriter, r *http.Request) {
	if h.products == nil || h.prices == nil {
		h.logger.ErrorContext(r.Context(), "catalog handler dependency is nil")
		writeError(w, http.StatusInternalServerError, "internal_error", "internal catalog error")
		return
	}

	startedAt := time.Now()
	requestID := observability.RequestIDFromContext(r.Context())
	productID := r.PathValue("id")
	h.logger.InfoContext(r.Context(), "catalog get_product started",
		slog.String("request_id", requestID),
		slog.String("product_id", productID),
	)

	item, err := h.products.GetProduct(r.Context(), productID)
	if err != nil {
		h.writeProductError(w, err)
		return
	}

	price, err := h.prices.GetPrice(r.Context(), productID)
	if err != nil {
		h.writePricingError(w, err)
		return
	}

	h.logger.InfoContext(r.Context(), "catalog get_product completed",
		slog.String("request_id", requestID),
		slog.String("product_id", productID),
		slog.Duration("duration", time.Since(startedAt)),
	)

	writeJSON(w, http.StatusOK, productResponse{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		Price: priceResponse{
			AmountCents:     price.AmountCents,
			Currency:        price.Currency,
			DiscountPercent: price.DiscountPercent,
		},
	})
}

func (h *Handler) writeProductError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, product.ErrInvalidProductID):
		writeError(w, http.StatusBadRequest, "invalid_product_id", "product id is invalid")
	case errors.Is(err, product.ErrNotFound):
		writeError(w, http.StatusNotFound, "product_not_found", "product not found")
	case errors.Is(err, context.Canceled):
		writeError(w, http.StatusRequestTimeout, "request_canceled", "request was canceled")
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "request_timeout", "request deadline exceeded")
	default:
		h.logger.Error("unexpected product service error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal_error", "internal catalog error")
	}
}

func (h *Handler) writePricingError(w http.ResponseWriter, err error) {
	switch status.Code(err) {
	case codes.InvalidArgument:
		writeError(w, http.StatusBadRequest, "invalid_product_id", "product id is invalid")
	case codes.NotFound:
		writeError(w, http.StatusNotFound, "price_not_found", "price not found")
	case codes.DeadlineExceeded:
		writeError(w, http.StatusGatewayTimeout, "pricing_timeout", "pricing service deadline exceeded")
	case codes.Unavailable:
		writeError(w, http.StatusServiceUnavailable, "pricing_unavailable", "pricing service unavailable")
	case codes.Canceled:
		writeError(w, http.StatusRequestTimeout, "request_canceled", "request was canceled")
	default:
		h.logger.Error("unexpected pricing service error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal_error", "internal catalog error")
	}
}

type productResponse struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Price       priceResponse `json:"price"`
}

type priceResponse struct {
	AmountCents     int64  `json:"amount_cents"`
	Currency        string `json:"currency"`
	DiscountPercent int32  `json:"discount_percent"`
}

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type readinessResponse struct {
	Status       string            `json:"status"`
	Service      string            `json:"service"`
	Dependencies map[string]string `json:"dependencies"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("failed to encode json response", slog.String("error", err.Error()))
	}
}

func writeError(w http.ResponseWriter, statusCode int, code, message string) {
	writeJSON(w, statusCode, errorResponse{
		Error: errorBody{
			Code:    code,
			Message: message,
		},
	})
}
