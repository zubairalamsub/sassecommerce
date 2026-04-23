package mocks

import (
	"context"

	"github.com/ecommerce/product-service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) Create(ctx context.Context, product *models.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockProductRepository) GetByID(ctx context.Context, id string) (*models.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductRepository) GetBySKU(ctx context.Context, tenantID, sku string) (*models.Product, error) {
	args := m.Called(ctx, tenantID, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductRepository) List(ctx context.Context, tenantID string, offset, limit int) ([]models.Product, int64, error) {
	args := m.Called(ctx, tenantID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Product), args.Get(1).(int64), args.Error(2)
}

func (m *MockProductRepository) ListByCategory(ctx context.Context, tenantID, categoryID string, offset, limit int) ([]models.Product, int64, error) {
	args := m.Called(ctx, tenantID, categoryID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Product), args.Get(1).(int64), args.Error(2)
}

func (m *MockProductRepository) Search(ctx context.Context, tenantID, query string, offset, limit int) ([]models.Product, int64, error) {
	args := m.Called(ctx, tenantID, query, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Product), args.Get(1).(int64), args.Error(2)
}

func (m *MockProductRepository) Update(ctx context.Context, id string, product *models.Product) error {
	args := m.Called(ctx, id, product)
	return args.Error(0)
}

func (m *MockProductRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProductRepository) SKUExists(ctx context.Context, tenantID, sku string) (bool, error) {
	args := m.Called(ctx, tenantID, sku)
	return args.Bool(0), args.Error(1)
}

func (m *MockProductRepository) UpdateStatus(ctx context.Context, id string, status models.ProductStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}
