package mocks

import (
	"context"

	"github.com/ecommerce/vendor-service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockVendorRepository struct {
	mock.Mock
}

func (m *MockVendorRepository) Create(ctx context.Context, vendor *models.Vendor) error {
	args := m.Called(ctx, vendor)
	return args.Error(0)
}

func (m *MockVendorRepository) GetByID(ctx context.Context, id string) (*models.Vendor, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Vendor), args.Error(1)
}

func (m *MockVendorRepository) GetByEmail(ctx context.Context, email string) (*models.Vendor, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Vendor), args.Error(1)
}

func (m *MockVendorRepository) List(ctx context.Context, tenantID string, status string, page, pageSize int) ([]models.Vendor, int64, error) {
	args := m.Called(ctx, tenantID, status, page, pageSize)
	return args.Get(0).([]models.Vendor), args.Get(1).(int64), args.Error(2)
}

func (m *MockVendorRepository) Update(ctx context.Context, vendor *models.Vendor) error {
	args := m.Called(ctx, vendor)
	return args.Error(0)
}

func (m *MockVendorRepository) CreateOrder(ctx context.Context, order *models.VendorOrder) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockVendorRepository) GetOrdersByVendor(ctx context.Context, vendorID string, page, pageSize int) ([]models.VendorOrder, int64, error) {
	args := m.Called(ctx, vendorID, page, pageSize)
	return args.Get(0).([]models.VendorOrder), args.Get(1).(int64), args.Error(2)
}

func (m *MockVendorRepository) GetVendorAnalytics(ctx context.Context, vendorID string) (*models.VendorAnalyticsResponse, error) {
	args := m.Called(ctx, vendorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VendorAnalyticsResponse), args.Error(1)
}
