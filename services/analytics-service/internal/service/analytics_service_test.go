package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ecommerce/analytics-service/internal/models"
	repoMocks "github.com/ecommerce/analytics-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService() (*analyticsService, *repoMocks.MockAnalyticsRepository) {
	mockRepo := new(repoMocks.MockAnalyticsRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := &analyticsService{
		repo:   mockRepo,
		writer: nil,
		logger: logger,
	}

	return svc, mockRepo
}

// === GetSalesReport Tests ===

func TestGetSalesReport_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	report := &models.SalesReportResponse{
		TenantID:     "tenant-1",
		TotalRevenue: 50000,
		TotalOrders:  100,
		AverageOrder: 500,
	}
	dailySales := []models.DailySales{
		{Date: "2026-04-01", Revenue: 1000, Orders: 5},
	}

	mockRepo.On("GetSalesReport", ctx, "tenant-1", mock.Anything, mock.Anything).Return(report, nil)
	mockRepo.On("GetDailySales", ctx, "tenant-1", mock.Anything, mock.Anything).Return(dailySales, nil)

	req := &models.SalesReportRequest{TenantID: "tenant-1", Period: "daily"}
	result, err := svc.GetSalesReport(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, 50000.0, result.TotalRevenue)
	assert.Equal(t, int64(100), result.TotalOrders)
	assert.Equal(t, "daily", result.Period)
	assert.Len(t, result.DailySales, 1)
}

func TestGetSalesReport_DefaultPeriod(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	report := &models.SalesReportResponse{TenantID: "tenant-1"}
	mockRepo.On("GetSalesReport", ctx, "tenant-1", mock.Anything, mock.Anything).Return(report, nil)
	mockRepo.On("GetDailySales", ctx, "tenant-1", mock.Anything, mock.Anything).Return([]models.DailySales{}, nil)

	req := &models.SalesReportRequest{TenantID: "tenant-1"}
	result, err := svc.GetSalesReport(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "daily", result.Period)
}

func TestGetSalesReport_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetSalesReport", ctx, "tenant-1", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	req := &models.SalesReportRequest{TenantID: "tenant-1"}
	result, err := svc.GetSalesReport(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to generate sales report")
}

func TestGetSalesReport_DailySalesFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	report := &models.SalesReportResponse{TenantID: "tenant-1", TotalRevenue: 1000}
	mockRepo.On("GetSalesReport", ctx, "tenant-1", mock.Anything, mock.Anything).Return(report, nil)
	mockRepo.On("GetDailySales", ctx, "tenant-1", mock.Anything, mock.Anything).Return([]models.DailySales{}, errors.New("daily error"))

	req := &models.SalesReportRequest{TenantID: "tenant-1"}
	result, err := svc.GetSalesReport(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, 1000.0, result.TotalRevenue)
	// DailySales not set on error, but no overall failure
}

// === GetCustomerInsights Tests ===

func TestGetCustomerInsights_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	insights := &models.CustomerInsightsResponse{
		TenantID:          "tenant-1",
		TotalCustomers:    500,
		NewCustomers:      50,
		ReturningCustomers: 450,
		AverageOrderValue: 75.5,
	}
	mockRepo.On("GetCustomerInsights", ctx, "tenant-1", mock.Anything, mock.Anything).Return(insights, nil)

	req := &models.CustomerInsightsRequest{TenantID: "tenant-1"}
	result, err := svc.GetCustomerInsights(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, int64(500), result.TotalCustomers)
	assert.Equal(t, int64(50), result.NewCustomers)
}

func TestGetCustomerInsights_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCustomerInsights", ctx, "tenant-1", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	req := &models.CustomerInsightsRequest{TenantID: "tenant-1"}
	result, err := svc.GetCustomerInsights(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === GetProductPerformance Tests ===

func TestGetProductPerformance_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	performance := &models.ProductPerformanceResponse{
		TenantID:      "tenant-1",
		TotalProducts: 200,
		TopSelling: []models.ProductPerformance{
			{ProductID: "p-1", TotalSold: 100, TotalRevenue: 5000},
		},
	}
	mockRepo.On("GetProductPerformance", ctx, "tenant-1", mock.Anything, mock.Anything, 10).Return(performance, nil)

	req := &models.ProductPerformanceRequest{TenantID: "tenant-1"}
	result, err := svc.GetProductPerformance(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, int64(200), result.TotalProducts)
	assert.Len(t, result.TopSelling, 1)
}

func TestGetProductPerformance_CustomLimit(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	performance := &models.ProductPerformanceResponse{TenantID: "tenant-1"}
	mockRepo.On("GetProductPerformance", ctx, "tenant-1", mock.Anything, mock.Anything, 5).Return(performance, nil)

	req := &models.ProductPerformanceRequest{TenantID: "tenant-1", Limit: 5}
	result, err := svc.GetProductPerformance(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetProductPerformance_LimitCapped(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	performance := &models.ProductPerformanceResponse{TenantID: "tenant-1"}
	mockRepo.On("GetProductPerformance", ctx, "tenant-1", mock.Anything, mock.Anything, 100).Return(performance, nil)

	req := &models.ProductPerformanceRequest{TenantID: "tenant-1", Limit: 500}
	result, err := svc.GetProductPerformance(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetProductPerformance_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetProductPerformance", ctx, "tenant-1", mock.Anything, mock.Anything, 10).Return(nil, errors.New("db error"))

	req := &models.ProductPerformanceRequest{TenantID: "tenant-1"}
	result, err := svc.GetProductPerformance(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === CreateReport Tests ===

func TestCreateReport_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("CreateReport", ctx, mock.AnythingOfType("*models.CustomReport")).Return(nil)

	salesReport := &models.SalesReportResponse{TenantID: "tenant-1", TotalRevenue: 1000}
	mockRepo.On("GetSalesReport", ctx, "tenant-1", mock.Anything, mock.Anything).Return(salesReport, nil)
	mockRepo.On("UpdateReport", ctx, mock.AnythingOfType("*models.CustomReport")).Return(nil)

	req := &models.CreateReportRequest{
		TenantID:   "tenant-1",
		Name:       "Monthly Sales",
		ReportType: "sales",
		DateFrom:   "2026-03-01",
		DateTo:     "2026-03-31",
	}

	result, err := svc.CreateReport(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Monthly Sales", result.Name)
	assert.Equal(t, "sales", result.ReportType)
	assert.Equal(t, "completed", result.Status)
}

func TestCreateReport_CustomersType(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("CreateReport", ctx, mock.AnythingOfType("*models.CustomReport")).Return(nil)

	insights := &models.CustomerInsightsResponse{TenantID: "tenant-1", TotalCustomers: 100}
	mockRepo.On("GetCustomerInsights", ctx, "tenant-1", mock.Anything, mock.Anything).Return(insights, nil)
	mockRepo.On("UpdateReport", ctx, mock.AnythingOfType("*models.CustomReport")).Return(nil)

	req := &models.CreateReportRequest{
		TenantID:   "tenant-1",
		Name:       "Customer Report",
		ReportType: "customers",
		DateFrom:   "2026-03-01",
		DateTo:     "2026-03-31",
	}

	result, err := svc.CreateReport(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
}

func TestCreateReport_ProductsType(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("CreateReport", ctx, mock.AnythingOfType("*models.CustomReport")).Return(nil)

	perf := &models.ProductPerformanceResponse{TenantID: "tenant-1", TotalProducts: 50}
	mockRepo.On("GetProductPerformance", ctx, "tenant-1", mock.Anything, mock.Anything, 10).Return(perf, nil)
	mockRepo.On("UpdateReport", ctx, mock.AnythingOfType("*models.CustomReport")).Return(nil)

	req := &models.CreateReportRequest{
		TenantID:   "tenant-1",
		Name:       "Product Report",
		ReportType: "products",
		DateFrom:   "2026-03-01",
		DateTo:     "2026-03-31",
	}

	result, err := svc.CreateReport(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
}

func TestCreateReport_InvalidDateFrom(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	req := &models.CreateReportRequest{
		TenantID:   "tenant-1",
		Name:       "Test",
		ReportType: "sales",
		DateFrom:   "invalid",
		DateTo:     "2026-03-31",
	}

	result, err := svc.CreateReport(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid date_from")
}

func TestCreateReport_InvalidDateTo(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	req := &models.CreateReportRequest{
		TenantID:   "tenant-1",
		Name:       "Test",
		ReportType: "sales",
		DateFrom:   "2026-03-01",
		DateTo:     "bad-date",
	}

	result, err := svc.CreateReport(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid date_to")
}

func TestCreateReport_DateToBeforeDateFrom(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	req := &models.CreateReportRequest{
		TenantID:   "tenant-1",
		Name:       "Test",
		ReportType: "sales",
		DateFrom:   "2026-04-01",
		DateTo:     "2026-03-01",
	}

	result, err := svc.CreateReport(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "date_to must be after date_from")
}

func TestCreateReport_InvalidType(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	req := &models.CreateReportRequest{
		TenantID:   "tenant-1",
		Name:       "Test",
		ReportType: "unknown",
		DateFrom:   "2026-03-01",
		DateTo:     "2026-03-31",
	}

	result, err := svc.CreateReport(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid report_type")
}

func TestCreateReport_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("CreateReport", ctx, mock.AnythingOfType("*models.CustomReport")).Return(errors.New("db error"))

	req := &models.CreateReportRequest{
		TenantID:   "tenant-1",
		Name:       "Test",
		ReportType: "sales",
		DateFrom:   "2026-03-01",
		DateTo:     "2026-03-31",
	}

	result, err := svc.CreateReport(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCreateReport_WithFilters(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("CreateReport", ctx, mock.AnythingOfType("*models.CustomReport")).Return(nil)
	salesReport := &models.SalesReportResponse{TenantID: "tenant-1"}
	mockRepo.On("GetSalesReport", ctx, "tenant-1", mock.Anything, mock.Anything).Return(salesReport, nil)
	mockRepo.On("UpdateReport", ctx, mock.AnythingOfType("*models.CustomReport")).Return(nil)

	req := &models.CreateReportRequest{
		TenantID:   "tenant-1",
		Name:       "Filtered Report",
		ReportType: "sales",
		DateFrom:   "2026-03-01",
		DateTo:     "2026-03-31",
		Filters:    map[string]string{"channel": "web"},
	}

	result, err := svc.CreateReport(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// === GetReport Tests ===

func TestGetReport_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	now := time.Now().UTC()
	report := &models.CustomReport{
		ID:          "report-1",
		TenantID:    "tenant-1",
		Name:        "Monthly Sales",
		ReportType:  "sales",
		DateFrom:    time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		DateTo:      time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		Status:      "completed",
		ResultData:  `{"total_revenue":5000}`,
		CompletedAt: &now,
	}
	mockRepo.On("GetReport", ctx, "report-1").Return(report, nil)

	result, err := svc.GetReport(ctx, "report-1")

	assert.NoError(t, err)
	assert.Equal(t, "Monthly Sales", result.Name)
	assert.Equal(t, "completed", result.Status)
	assert.NotNil(t, result.Result)
}

func TestGetReport_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetReport", ctx, "bad").Return(nil, errors.New("record not found"))

	result, err := svc.GetReport(ctx, "bad")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

// === ListReports Tests ===

func TestListReports_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	reports := []models.CustomReport{
		{ID: "r-1", TenantID: "tenant-1", Name: "Report 1", ReportType: "sales", Status: "completed"},
	}
	mockRepo.On("ListReports", ctx, "tenant-1", 1, 20).Return(reports, int64(1), nil)

	results, total, err := svc.ListReports(ctx, "tenant-1", 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
}

func TestListReports_DefaultPagination(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("ListReports", ctx, "tenant-1", 1, 20).Return([]models.CustomReport{}, int64(0), nil)

	results, total, err := svc.ListReports(ctx, "tenant-1", 0, 0)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, results, 0)
}

// === RecordSale Tests ===

func TestRecordSale_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("RecordSalesEvent", ctx, mock.AnythingOfType("*models.SalesEvent")).Return(nil)
	mockRepo.On("RecordCustomerEvent", ctx, mock.AnythingOfType("*models.CustomerEvent")).Return(nil)

	err := svc.RecordSale(ctx, "tenant-1", "order-1", "user-1", "vendor-1", 250.0, "web")

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "RecordSalesEvent", ctx, mock.AnythingOfType("*models.SalesEvent"))
	mockRepo.AssertCalled(t, "RecordCustomerEvent", ctx, mock.AnythingOfType("*models.CustomerEvent"))
}

func TestRecordSale_DefaultChannel(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("RecordSalesEvent", ctx, mock.AnythingOfType("*models.SalesEvent")).Return(nil)
	mockRepo.On("RecordCustomerEvent", ctx, mock.AnythingOfType("*models.CustomerEvent")).Return(nil)

	err := svc.RecordSale(ctx, "tenant-1", "order-1", "user-1", "", 100.0, "")

	assert.NoError(t, err)
}

func TestRecordSale_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("RecordSalesEvent", ctx, mock.AnythingOfType("*models.SalesEvent")).Return(errors.New("db error"))

	err := svc.RecordSale(ctx, "tenant-1", "order-1", "user-1", "", 100.0, "web")

	assert.Error(t, err)
}

// === RecordCustomerActivity Tests ===

func TestRecordCustomerActivity_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("RecordCustomerEvent", ctx, mock.AnythingOfType("*models.CustomerEvent")).Return(nil)

	err := svc.RecordCustomerActivity(ctx, "tenant-1", "user-1", "purchase", "order-1", 100)

	assert.NoError(t, err)
}

// === RecordProductActivity Tests ===

func TestRecordProductActivity_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("RecordProductEvent", ctx, mock.AnythingOfType("*models.ProductEvent")).Return(nil)

	err := svc.RecordProductActivity(ctx, "tenant-1", "product-1", "product_sold", 5, 250.0)

	assert.NoError(t, err)
}

func TestRecordProductActivity_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("RecordProductEvent", ctx, mock.AnythingOfType("*models.ProductEvent")).Return(errors.New("db error"))

	err := svc.RecordProductActivity(ctx, "tenant-1", "product-1", "product_sold", 5, 250.0)

	assert.Error(t, err)
}

// === parseDateRange Tests ===

func TestParseDateRange_ValidDates(t *testing.T) {
	from, to := parseDateRange("2026-01-01", "2026-01-31")

	assert.Equal(t, 2026, from.Year())
	assert.Equal(t, time.January, from.Month())
	assert.Equal(t, 1, from.Day())
	assert.Equal(t, 2026, to.Year())
	assert.Equal(t, time.January, to.Month())
	assert.Equal(t, 31, to.Day())
}

func TestParseDateRange_EmptyDates(t *testing.T) {
	from, to := parseDateRange("", "")

	// Should default to last 30 days
	assert.True(t, from.Before(to))
	assert.WithinDuration(t, time.Now().UTC(), to, 2*time.Second)
}

func TestParseDateRange_InvalidDates(t *testing.T) {
	from, to := parseDateRange("bad", "bad")

	// Should fall back to defaults
	assert.True(t, from.Before(to))
}

// === toReportResponse Tests ===

func TestToReportResponse_WithResult(t *testing.T) {
	now := time.Now().UTC()
	report := &models.CustomReport{
		ID:          "r-1",
		TenantID:    "tenant-1",
		Name:        "Test Report",
		ReportType:  "sales",
		DateFrom:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		DateTo:      time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		Filters:     `{"channel":"web"}`,
		ResultData:  `{"total_revenue":5000}`,
		Status:      "completed",
		CompletedAt: &now,
	}

	resp := toReportResponse(report)

	assert.Equal(t, "r-1", resp.ID)
	assert.Equal(t, "Test Report", resp.Name)
	assert.NotNil(t, resp.Filters)
	assert.Equal(t, "web", resp.Filters["channel"])
	assert.NotNil(t, resp.Result)
	assert.NotNil(t, resp.CompletedAt)
}

func TestToReportResponse_EmptyFilters(t *testing.T) {
	report := &models.CustomReport{
		ID:         "r-1",
		TenantID:   "tenant-1",
		Name:       "Test",
		ReportType: "sales",
		Filters:    "{}",
		Status:     "pending",
	}

	resp := toReportResponse(report)

	assert.Nil(t, resp.Filters)
	assert.Nil(t, resp.Result)
}
