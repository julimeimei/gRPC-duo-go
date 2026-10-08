package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/julimeimei/grpc-duo-go/services/pricing-service/internal/pricing"
)

func TestRepositoryGetPrice(t *testing.T) {
	db, mock, cleanup := newMockDB(t)
	defer cleanup()

	repository := NewRepository(db)

	mock.ExpectQuery(`SELECT product_id, amount_cents, currency, discount_percent\s+FROM product_prices\s+WHERE product_id = \$1`).
		WithArgs("42").
		WillReturnRows(sqlmock.NewRows([]string{"product_id", "amount_cents", "currency", "discount_percent"}).
			AddRow("42", int64(12990), "BRL", int32(10)))

	price, err := repository.GetPrice(context.Background(), "42")
	if err != nil {
		t.Fatalf("GetPrice() error = %v", err)
	}

	if price != (pricing.Price{ProductID: "42", AmountCents: 12990, Currency: "BRL", DiscountPercent: 10}) {
		t.Fatalf("GetPrice() = %+v, want product 42 price", price)
	}
}

func TestRepositoryGetPriceReturnsNotFound(t *testing.T) {
	db, mock, cleanup := newMockDB(t)
	defer cleanup()

	repository := NewRepository(db)

	mock.ExpectQuery(`SELECT product_id, amount_cents, currency, discount_percent\s+FROM product_prices\s+WHERE product_id = \$1`).
		WithArgs("999").
		WillReturnError(sql.ErrNoRows)

	_, err := repository.GetPrice(context.Background(), "999")
	if !errors.Is(err, pricing.ErrNotFound) {
		t.Fatalf("GetPrice() error = %v, want %v", err, pricing.ErrNotFound)
	}
}

func TestRepositoryGetPriceWrapsQueryError(t *testing.T) {
	db, mock, cleanup := newMockDB(t)
	defer cleanup()

	repository := NewRepository(db)

	mock.ExpectQuery(`SELECT product_id, amount_cents, currency, discount_percent\s+FROM product_prices\s+WHERE product_id = \$1`).
		WithArgs("42").
		WillReturnError(errors.New("database unavailable"))

	_, err := repository.GetPrice(context.Background(), "42")
	if err == nil {
		t.Fatal("GetPrice() error = nil, want error")
	}
	if errors.Is(err, pricing.ErrNotFound) {
		t.Fatalf("GetPrice() error = %v, did not want not found", err)
	}
}

func TestRepositoryPing(t *testing.T) {
	db, mock, cleanup := newMockDB(t)
	defer cleanup()

	repository := NewRepository(db)

	mock.ExpectPing()

	if err := repository.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestOpenRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "empty database url",
			cfg: Config{
				DatabaseURL:     "",
				MaxOpenConns:    1,
				MaxIdleConns:    1,
				ConnMaxLifetime: 1,
			},
		},
		{
			name: "invalid max open",
			cfg: Config{
				DatabaseURL:     "postgres://localhost/db",
				MaxOpenConns:    0,
				MaxIdleConns:    1,
				ConnMaxLifetime: 1,
			},
		},
		{
			name: "idle exceeds open",
			cfg: Config{
				DatabaseURL:     "postgres://localhost/db",
				MaxOpenConns:    1,
				MaxIdleConns:    2,
				ConnMaxLifetime: 1,
			},
		},
		{
			name: "invalid lifetime",
			cfg: Config{
				DatabaseURL:     "postgres://localhost/db",
				MaxOpenConns:    1,
				MaxIdleConns:    1,
				ConnMaxLifetime: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Open(context.Background(), tt.cfg); err == nil {
				t.Fatal("Open() error = nil, want error")
			}
		})
	}
}

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}

	cleanup := func() {
		mock.ExpectClose()
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet sql expectations: %v", err)
		}
	}

	return db, mock, cleanup
}
