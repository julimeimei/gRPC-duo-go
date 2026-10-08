package pricing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidProductID = errors.New("invalid product id")
	ErrNotFound         = errors.New("price not found")
)

type Price struct {
	ProductID       string
	AmountCents     int64
	Currency        string
	DiscountPercent int32
}

type Repository interface {
	GetPrice(ctx context.Context, productID string) (Price, error)
}

type Service struct {
	repository   Repository
	queryTimeout time.Duration
}

func NewService(repository Repository, queryTimeout time.Duration) (*Service, error) {
	if repository == nil {
		return nil, errors.New("pricing repository is required")
	}
	if queryTimeout <= 0 {
		return nil, errors.New("pricing query timeout must be greater than zero")
	}

	return &Service{
		repository:   repository,
		queryTimeout: queryTimeout,
	}, nil
}

func (s *Service) GetPrice(ctx context.Context, productID string) (Price, error) {
	if err := ctx.Err(); err != nil {
		return Price{}, err
	}

	normalizedID, err := normalizeProductID(productID)
	if err != nil {
		return Price{}, ErrInvalidProductID
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	price, err := s.repository.GetPrice(queryCtx, normalizedID)
	if err != nil {
		return Price{}, err
	}
	if err := validatePrice(price); err != nil {
		return Price{}, fmt.Errorf("repository returned invalid price for product_id %q: %w", normalizedID, err)
	}

	return price, nil
}

func validatePrice(price Price) error {
	normalizedID, err := normalizeProductID(price.ProductID)
	if err != nil {
		return ErrInvalidProductID
	}
	if normalizedID != price.ProductID {
		return fmt.Errorf("product_id is not normalized")
	}
	if price.AmountCents < 0 {
		return fmt.Errorf("amount_cents must be greater than or equal to zero")
	}
	if !isCurrencyCode(price.Currency) {
		return fmt.Errorf("currency must be a 3-letter code")
	}
	if price.Currency != strings.ToUpper(price.Currency) {
		return fmt.Errorf("currency must be uppercase")
	}
	if price.DiscountPercent < 0 || price.DiscountPercent > 100 {
		return fmt.Errorf("discount_percent must be between 0 and 100")
	}

	return nil
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

func isCurrencyCode(value string) bool {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return false
	}

	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return false
		}
	}

	return true
}
