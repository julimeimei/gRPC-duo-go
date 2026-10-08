package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/observability"
	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/pricing"
)

type Config struct {
	DatabaseURL     string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	Logger          *slog.Logger
}

type Repository struct {
	db     *sql.DB
	logger *slog.Logger
}

func Open(ctx context.Context, cfg Config) (*Repository, error) {
	if cfg.DatabaseURL == "" {
		return nil, errors.New("database url is required")
	}
	if cfg.MaxOpenConns <= 0 {
		return nil, errors.New("max open connections must be greater than zero")
	}
	if cfg.MaxIdleConns <= 0 {
		return nil, errors.New("max idle connections must be greater than zero")
	}
	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		return nil, errors.New("max idle connections must not exceed max open connections")
	}
	if cfg.ConnMaxLifetime <= 0 {
		return nil, errors.New("connection max lifetime must be greater than zero")
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection pool: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	repository := NewRepository(db, cfg.Logger)
	if err := repository.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return repository, nil
}

func NewRepository(db *sql.DB, loggers ...*slog.Logger) *Repository {
	logger := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}

	return &Repository{
		db:     db,
		logger: logger,
	}
}

func (r *Repository) GetPrice(ctx context.Context, productID string) (pricing.Price, error) {
	if r.db == nil {
		return pricing.Price{}, errors.New("postgres repository database is nil")
	}
	if r.logger == nil {
		r.logger = slog.Default()
	}

	const query = `
SELECT product_id, amount_cents, currency, discount_percent
FROM product_prices
WHERE product_id = $1`

	startedAt := time.Now()
	requestID := observability.RequestIDFromContext(ctx)
	r.logger.InfoContext(ctx, "postgres query started",
		slog.String("request_id", requestID),
		slog.String("operation", "product_prices.get_by_product_id"),
		slog.String("product_id", productID),
	)

	var price pricing.Price
	if err := r.db.QueryRowContext(ctx, query, productID).Scan(
		&price.ProductID,
		&price.AmountCents,
		&price.Currency,
		&price.DiscountPercent,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.InfoContext(ctx, "postgres query completed",
				slog.String("request_id", requestID),
				slog.String("operation", "product_prices.get_by_product_id"),
				slog.String("result", "not_found"),
				slog.Duration("duration", time.Since(startedAt)),
			)
			return pricing.Price{}, pricing.ErrNotFound
		}
		r.logger.ErrorContext(ctx, "postgres query failed",
			slog.String("request_id", requestID),
			slog.String("operation", "product_prices.get_by_product_id"),
			slog.Duration("duration", time.Since(startedAt)),
			slog.String("error", err.Error()),
		)
		return pricing.Price{}, fmt.Errorf("query product price: %w", err)
	}

	r.logger.InfoContext(ctx, "postgres query completed",
		slog.String("request_id", requestID),
		slog.String("operation", "product_prices.get_by_product_id"),
		slog.String("result", "found"),
		slog.Duration("duration", time.Since(startedAt)),
	)

	return price, nil
}

func (r *Repository) Ping(ctx context.Context) error {
	if r.db == nil {
		return errors.New("postgres repository database is nil")
	}

	return r.db.PingContext(ctx)
}

func (r *Repository) Close() error {
	if r.db == nil {
		return nil
	}

	return r.db.Close()
}
