package pricing

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestServiceGetPrice(t *testing.T) {
	service := newTestService(t, fakeRepository{
		price: Price{
			ProductID:       "42",
			AmountCents:     12990,
			Currency:        "BRL",
			DiscountPercent: 10,
		},
	})

	price, err := service.GetPrice(context.Background(), "42")
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

func TestServiceGetPriceTrimsProductID(t *testing.T) {
	repository := &recordingRepository{
		price: Price{
			ProductID:       "42",
			AmountCents:     12990,
			Currency:        "BRL",
			DiscountPercent: 10,
		},
	}
	service := newTestService(t, repository)

	if _, err := service.GetPrice(context.Background(), " 42 "); err != nil {
		t.Fatalf("GetPrice() error = %v", err)
	}

	if repository.productID != "42" {
		t.Fatalf("repository productID = %q, want %q", repository.productID, "42")
	}
}

func TestServiceGetPriceRejectsInvalidProductID(t *testing.T) {
	service := newTestService(t, fakeRepository{})

	tests := []string{
		"",
		"   ",
		"../../etc/passwd",
		"product id",
		strings.Repeat("a", 65),
	}

	for _, productID := range tests {
		t.Run(productID, func(t *testing.T) {
			_, err := service.GetPrice(context.Background(), productID)
			if !errors.Is(err, ErrInvalidProductID) {
				t.Fatalf("GetPrice() error = %v, want %v", err, ErrInvalidProductID)
			}
		})
	}
}

func TestServiceGetPriceReturnsRepositoryError(t *testing.T) {
	service := newTestService(t, fakeRepository{err: ErrNotFound})

	_, err := service.GetPrice(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetPrice() error = %v, want %v", err, ErrNotFound)
	}
}

func TestServiceGetPriceReturnsContextError(t *testing.T) {
	service := newTestService(t, fakeRepository{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.GetPrice(ctx, "42")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetPrice() error = %v, want %v", err, context.Canceled)
	}
}

func TestServiceGetPriceAppliesQueryTimeout(t *testing.T) {
	repository := &deadlineRepository{
		price: Price{
			ProductID:       "42",
			AmountCents:     12990,
			Currency:        "BRL",
			DiscountPercent: 10,
		},
	}
	service := newTestService(t, repository)

	if _, err := service.GetPrice(context.Background(), "42"); err != nil {
		t.Fatalf("GetPrice() error = %v", err)
	}

	if !repository.hasDeadline {
		t.Fatal("repository context has no deadline")
	}
}

func TestServiceRejectsInvalidRepositoryPrice(t *testing.T) {
	tests := []struct {
		name  string
		price Price
	}{
		{
			name: "invalid product id",
			price: Price{
				ProductID:       "bad id",
				AmountCents:     100,
				Currency:        "BRL",
				DiscountPercent: 0,
			},
		},
		{
			name: "negative amount",
			price: Price{
				ProductID:       "42",
				AmountCents:     -1,
				Currency:        "BRL",
				DiscountPercent: 0,
			},
		},
		{
			name: "invalid currency",
			price: Price{
				ProductID:       "42",
				AmountCents:     100,
				Currency:        "BR",
				DiscountPercent: 0,
			},
		},
		{
			name: "lowercase currency",
			price: Price{
				ProductID:       "42",
				AmountCents:     100,
				Currency:        "brl",
				DiscountPercent: 0,
			},
		},
		{
			name: "invalid discount",
			price: Price{
				ProductID:       "42",
				AmountCents:     100,
				Currency:        "BRL",
				DiscountPercent: 101,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestService(t, fakeRepository{price: tt.price})

			if _, err := service.GetPrice(context.Background(), "42"); err == nil {
				t.Fatal("GetPrice() error = nil, want error")
			}
		})
	}
}

func TestNewServiceRejectsInvalidDependencies(t *testing.T) {
	if _, err := NewService(nil, time.Second); err == nil {
		t.Fatal("NewService() error = nil, want repository error")
	}

	if _, err := NewService(fakeRepository{}, 0); err == nil {
		t.Fatal("NewService() error = nil, want timeout error")
	}
}

func newTestService(t *testing.T, repository Repository) *Service {
	t.Helper()

	service, err := NewService(repository, time.Second)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	return service
}

type fakeRepository struct {
	price Price
	err   error
}

func (f fakeRepository) GetPrice(context.Context, string) (Price, error) {
	return f.price, f.err
}

type recordingRepository struct {
	price     Price
	productID string
}

func (r *recordingRepository) GetPrice(_ context.Context, productID string) (Price, error) {
	r.productID = productID
	return r.price, nil
}

type deadlineRepository struct {
	price       Price
	hasDeadline bool
}

func (r *deadlineRepository) GetPrice(ctx context.Context, _ string) (Price, error) {
	_, r.hasDeadline = ctx.Deadline()
	return r.price, nil
}
