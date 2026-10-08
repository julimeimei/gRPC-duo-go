package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidProductID = errors.New("invalid product id")
	ErrNotFound         = errors.New("product not found")
)

type Product struct {
	ID          string
	Name        string
	Description string
}

type InMemoryService struct {
	products map[string]Product
}

func NewInMemoryService(products []Product) (*InMemoryService, error) {
	index := make(map[string]Product, len(products))
	for _, item := range products {
		normalizedID, err := normalizeProductID(item.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid seed product id %q: %w", item.ID, err)
		}
		if strings.TrimSpace(item.Name) == "" {
			return nil, fmt.Errorf("invalid seed product name for id %q", item.ID)
		}
		if _, exists := index[normalizedID]; exists {
			return nil, fmt.Errorf("duplicate seed product id %q", normalizedID)
		}

		item.ID = normalizedID
		item.Name = strings.TrimSpace(item.Name)
		item.Description = strings.TrimSpace(item.Description)
		index[normalizedID] = item
	}

	return &InMemoryService{products: index}, nil
}

func DefaultProducts() []Product {
	return []Product{
		{
			ID:          "42",
			Name:        "Wireless Mouse",
			Description: "Ergonomic wireless mouse",
		},
		{
			ID:          "100",
			Name:        "USB-C Dock",
			Description: "Compact dock for external monitors and peripherals",
		},
		{
			ID:          "keyboard-pro",
			Name:        "Mechanical Keyboard Pro",
			Description: "Low-latency mechanical keyboard",
		},
	}
}

func (s *InMemoryService) GetProduct(ctx context.Context, productID string) (Product, error) {
	if err := ctx.Err(); err != nil {
		return Product{}, err
	}

	normalizedID, err := normalizeProductID(productID)
	if err != nil {
		return Product{}, ErrInvalidProductID
	}

	item, ok := s.products[normalizedID]
	if !ok {
		return Product{}, ErrNotFound
	}

	return item, nil
}

func normalizeProductID(productID string) (string, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" || len(productID) > 64 {
		return "", ErrInvalidProductID
	}

	for _, r := range productID {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return "", ErrInvalidProductID
		}
	}

	return productID, nil
}
