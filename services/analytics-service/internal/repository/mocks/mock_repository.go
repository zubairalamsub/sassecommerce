package mocks

import (
	"context"
	"time"

	"github.com/ecommerce/analytics-service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockAnalyticsRepository struct {
	mock.Mock
}

func (m *MockAnalyticsRepository) RecordSalesEvent(ctx context.Context, event *models.SalesEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) GetSalesReport(ctx context.Context, tenantID string, from, to time.Time) (*models.SalesReportResponse, error) {
	args := m.Called(ctx, tenantID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SalesReportResponse), args.Error(1)
}

func (m *MockAnalyticsRepository) GetDailySales(ctx context.Context, tenantID string, from, to time.Time) ([]models.DailySales, error) {
	args := m.Called(ctx, tenantID, from, to)
	return args.Get(0).([]models.DailySales), args.Error(1)
}

func (m *MockAnalyticsRepository) RecordCustomerEvent(ctx context.Context, event *models.CustomerEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) GetCustomerInsights(ctx context.Context, tenantID string, from, to time.Time) (*models.CustomerInsightsResponse, error) {
	args := m.Called(ctx, tenantID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CustomerInsightsResponse), args.Error(1)
}

func (m *MockAnalyticsRepository) RecordProductEvent(ctx context.Context, event *models.ProductEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) GetProductPerformance(ctx context.Context, tenantID string, from, to time.Time, limit int) (*models.ProductPerformanceResponse, error) {
	args := m.Called(ctx, tenantID, from, to, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProductPerformanceResponse), args.Error(1)
}

func (m *MockAnalyticsRepository) CreateReport(ctx context.Context, report *models.CustomReport) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) GetReport(ctx context.Context, id string) (*models.CustomReport, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CustomReport), args.Error(1)
}

func (m *MockAnalyticsRepository) ListReports(ctx context.Context, tenantID string, page, pageSize int) ([]models.CustomReport, int64, error) {
	args := m.Called(ctx, tenantID, page, pageSize)
	return args.Get(0).([]models.CustomReport), args.Get(1).(int64), args.Error(2)
}

func (m *MockAnalyticsRepository) UpdateReport(ctx context.Context, report *models.CustomReport) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}
