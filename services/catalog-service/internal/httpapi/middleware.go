package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/julimeimei/grpc-duo-go/services/catalog-service/internal/observability"
)

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		requestID := observability.NormalizeRequestID(r.Header.Get(observability.RequestIDHeader))
		if requestID == "" {
			requestID = observability.NewRequestID()
		}
		ctx := observability.ContextWithRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)

		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		recorder.Header().Set(observability.RequestIDHeader, requestID)

		logger.InfoContext(ctx, "http request started",
			slog.String("request_id", requestID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)

		next.ServeHTTP(recorder, r)

		logger.InfoContext(ctx, "http request completed",
			slog.String("request_id", requestID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.statusCode),
			slog.Duration("duration", time.Since(startedAt)),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
