package mocks

import (
	"context"

	"github.com/ecommerce/cart-service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockCartRepository struct {
	mock.Mock
}

func (m *MockCartRepository) GetCart(ctx context.Context, tenantID, userID string) (*models.Cart, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Cart), args.Error(1)
}

func (m *MockCartRepository) SaveCart(ctx context.Context, cart *models.Cart) error {
	args := m.Called(ctx, cart)
	return args.Error(0)
}

func (m *MockCartRepository) DeleteCart(ctx context.Context, tenantID, userID string) error {
	args := m.Called(ctx, tenantID, userID)
	return args.Error(0)
}

func (m *MockCartRepository) GetCartsByProduct(ctx context.Context, productID string) ([]string, error) {
	args := m.Called(ctx, productID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockCartRepository) AddProductCartMapping(ctx context.Context, productID, cartKey string) error {
	args := m.Called(ctx, productID, cartKey)
	return args.Error(0)
}

func (m *MockCartRepository) RemoveProductCartMapping(ctx context.Context, productID, cartKey string) error {
	args := m.Called(ctx, productID, cartKey)
	return args.Error(0)
}
