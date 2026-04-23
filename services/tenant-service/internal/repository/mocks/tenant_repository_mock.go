package mocks

import (
	"context"

	"github.com/ecommerce/tenant-service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockTenantRepository struct {
	mock.Mock
}

func (m *MockTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) GetByDomain(ctx context.Context, domain string) (*models.Tenant, error) {
	args := m.Called(ctx, domain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tenant), args.Error(1)
}

func (m *MockTenantRepository) List(ctx context.Context, page, pageSize int) ([]models.Tenant, int64, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]models.Tenant), args.Get(1).(int64), args.Error(2)
}

func (m *MockTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTenantRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}
