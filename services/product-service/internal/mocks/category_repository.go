package mocks

import (
	"context"

	"github.com/ecommerce/product-service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) Create(ctx context.Context, category *models.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) GetByID(ctx context.Context, id string) (*models.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetBySlug(ctx context.Context, tenantID, slug string) (*models.Category, error) {
	args := m.Called(ctx, tenantID, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Category), args.Error(1)
}

func (m *MockCategoryRepository) List(ctx context.Context, tenantID string, offset, limit int) ([]models.Category, int64, error) {
	args := m.Called(ctx, tenantID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Category), args.Get(1).(int64), args.Error(2)
}

func (m *MockCategoryRepository) ListByParent(ctx context.Context, tenantID string, parentID *string, offset, limit int) ([]models.Category, int64, error) {
	args := m.Called(ctx, tenantID, parentID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Category), args.Get(1).(int64), args.Error(2)
}

func (m *MockCategoryRepository) Update(ctx context.Context, id string, category *models.Category) error {
	args := m.Called(ctx, id, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCategoryRepository) SlugExists(ctx context.Context, tenantID, slug string) (bool, error) {
	args := m.Called(ctx, tenantID, slug)
	return args.Bool(0), args.Error(1)
}

func (m *MockCategoryRepository) UpdateStatus(ctx context.Context, id string, status models.CategoryStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}
