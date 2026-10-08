package product

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGetProduct(t *testing.T) {
	service, err := NewInMemoryService(DefaultProducts())
	if err != nil {
		t.Fatalf("NewInMemoryService() error = %v", err)
	}

	item, err := service.GetProduct(context.Background(), "42")
	if err != nil {
		t.Fatalf("GetProduct() error = %v", err)
	}

	if item.ID != "42" {
		t.Fatalf("ID = %q, want %q", item.ID, "42")
	}
}

func TestGetProductRejectsInvalidProductID(t *testing.T) {
	service, err := NewInMemoryService(DefaultProducts())
	if err != nil {
		t.Fatalf("NewInMemoryService() error = %v", err)
	}

	tests := []string{
		"",
		"   ",
		"../../etc/passwd",
		"product id",
		strings.Repeat("a", 65),
	}

	for _, productID := range tests {
		t.Run(productID, func(t *testing.T) {
			_, err := service.GetProduct(context.Background(), productID)
			if !errors.Is(err, ErrInvalidProductID) {
				t.Fatalf("GetProduct() error = %v, want %v", err, ErrInvalidProductID)
			}
		})
	}
}

func TestGetProductReturnsNotFound(t *testing.T) {
	service, err := NewInMemoryService(DefaultProducts())
	if err != nil {
		t.Fatalf("NewInMemoryService() error = %v", err)
	}

	_, err = service.GetProduct(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetProduct() error = %v, want %v", err, ErrNotFound)
	}
}

func TestGetProductReturnsContextError(t *testing.T) {
	service, err := NewInMemoryService(DefaultProducts())
	if err != nil {
		t.Fatalf("NewInMemoryService() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = service.GetProduct(ctx, "42")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetProduct() error = %v, want %v", err, context.Canceled)
	}
}

func TestNewInMemoryServiceRejectsInvalidSeedData(t *testing.T) {
	tests := []struct {
		name     string
		products []Product
	}{
		{
			name: "invalid id",
			products: []Product{{
				ID:   "bad id",
				Name: "Bad",
			}},
		},
		{
			name: "empty name",
			products: []Product{{
				ID:   "42",
				Name: " ",
			}},
		},
		{
			name: "duplicate id",
			products: []Product{
				{ID: "42", Name: "One"},
				{ID: "42", Name: "Two"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewInMemoryService(tt.products); err == nil {
				t.Fatal("NewInMemoryService() error = nil, want error")
			}
		})
	}
}
