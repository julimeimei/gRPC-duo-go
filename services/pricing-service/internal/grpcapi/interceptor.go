package grpcapi

import (
	"context"
	"log/slog"
	"time"

	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func LoggingUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		startedAt := time.Now()
		requestID := requestIDFromMetadata(ctx)
		ctx = observability.ContextWithRequestID(ctx, requestID)
		requestID = observability.RequestIDFromContext(ctx)

		logger.InfoContext(ctx, "grpc request started",
			slog.String("request_id", requestID),
			slog.String("method", info.FullMethod),
		)

		resp, err := handler(ctx, req)
		code := status.Code(err)

		logger.InfoContext(ctx, "grpc request completed",
			slog.String("request_id", requestID),
			slog.String("method", info.FullMethod),
			slog.String("code", code.String()),
			slog.Duration("duration", time.Since(startedAt)),
		)

		return resp, err
	}
}

func requestIDFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get(observability.RequestIDKey)
	if len(values) == 0 {
		return ""
	}

	return observability.NormalizeRequestID(values[0])
}
