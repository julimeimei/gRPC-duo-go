//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/pricing"
)

func TestRepositoryIntegrationGetPriceWithPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("PRICING_INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set PRICING_INTEGRATION_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
	})

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("db.PingContext() error = %v", err)
	}

	runMigrations(t, ctx, db)

	const productID = "integration-widget"
	_, err = db.ExecContext(ctx, `
INSERT INTO product_prices (product_id, amount_cents, currency, discount_percent)
VALUES ($1, $2, $3, $4)
ON CONFLICT (product_id) DO UPDATE SET
    amount_cents = EXCLUDED.amount_cents,
    currency = EXCLUDED.currency,
    discount_percent = EXCLUDED.discount_percent,
    updated_at = now()
`, productID, int64(4567), "BRL", int32(5))
	if err != nil {
		t.Fatalf("insert integration price: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := db.ExecContext(cleanupCtx, `DELETE FROM product_prices WHERE product_id = $1`, productID); err != nil {
			t.Fatalf("delete integration price: %v", err)
		}
	})

	repository := NewRepository(db, slog.New(slog.NewJSONHandler(io.Discard, nil)))

	price, err := repository.GetPrice(ctx, productID)
	if err != nil {
		t.Fatalf("GetPrice() error = %v", err)
	}
	if price != (pricing.Price{ProductID: productID, AmountCents: 4567, Currency: "BRL", DiscountPercent: 5}) {
		t.Fatalf("GetPrice() = %+v, want integration price", price)
	}

	_, err = repository.GetPrice(ctx, "integration-missing")
	if !errors.Is(err, pricing.ErrNotFound) {
		t.Fatalf("GetPrice() error = %v, want %v", err, pricing.ErrNotFound)
	}
}

func runMigrations(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}

	migrationsDir := filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", "migrations", "pricing-service"))
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		t.Fatalf("find migrations: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no migrations found in %s", migrationsDir)
	}
	sort.Strings(files)

	for _, file := range files {
		statement, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s: %v", filepath.Base(file), err)
		}
		if _, err := db.ExecContext(ctx, string(statement)); err != nil {
			t.Fatalf("run migration %s: %v", filepath.Base(file), err)
		}
	}
}
