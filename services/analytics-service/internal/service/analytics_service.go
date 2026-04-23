package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecommerce/analytics-service/internal/models"
	"github.com/ecommerce/analytics-service/internal/repository"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type AnalyticsService interface {
	// Reports
	GetSalesReport(ctx context.Context, req *models.SalesReportRequest) (*models.SalesReportResponse, error)
	GetCustomerInsights(ctx context.Context, req *models.CustomerInsightsRequest) (*models.CustomerInsightsResponse, error)
	GetProductPerformance(ctx context.Context, req *models.ProductPerformanceRequest) (*models.ProductPerformanceResponse, error)

	// Custom reports
	CreateReport(ctx context.Context, req *models.CreateReportRequest) (*models.CustomReportResponse, error)
	GetReport(ctx context.Context, id string) (*models.CustomReportResponse, error)
	ListReports(ctx context.Context, tenantID string, page, pageSize int) ([]models.CustomReportResponse, int64, error)

	// Event ingestion (from Kafka)
	RecordSale(ctx context.Context, tenantID, orderID, userID, vendorID string, amount float64, channel string) error
	RecordCustomerActivity(ctx context.Context, tenantID, userID, eventType, orderID string, amount float64) error
	RecordProductActivity(ctx context.Context, tenantID, productID, eventType string, quantity int, revenue float64) error
}

type analyticsService struct {
	repo   repository.AnalyticsRepository
	writer *kafka.Writer
	logger *logrus.Logger
}

func NewAnalyticsService(repo repository.AnalyticsRepository, writer *kafka.Writer, logger *logrus.Logger) AnalyticsService {
	return &analyticsService{
		repo:   repo,
		writer: writer,
		logger: logger,
	}
}

func (s *analyticsService) GetSalesReport(ctx context.Context, req *models.SalesReportRequest) (*models.SalesReportResponse, error) {
	from, to := parseDateRange(req.DateFrom, req.DateTo)

	report, err := s.repo.GetSalesReport(ctx, req.TenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sales report: %w", err)
	}

	// Add daily sales breakdown
	dailySales, err := s.repo.GetDailySales(ctx, req.TenantID, from, to)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get daily sales breakdown")
	} else {
		report.DailySales = dailySales
	}

	report.Period = req.Period
	if report.Period == "" {
		report.Period = "daily"
	}

	return report, nil
}

func (s *analyticsService) GetCustomerInsights(ctx context.Context, req *models.CustomerInsightsRequest) (*models.CustomerInsightsResponse, error) {
	from, to := parseDateRange(req.DateFrom, req.DateTo)

	insights, err := s.repo.GetCustomerInsights(ctx, req.TenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to generate customer insights: %w", err)
	}

	return insights, nil
}

func (s *analyticsService) GetProductPerformance(ctx context.Context, req *models.ProductPerformanceRequest) (*models.ProductPerformanceResponse, error) {
	from, to := parseDateRange(req.DateFrom, req.DateTo)

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	performance, err := s.repo.GetProductPerformance(ctx, req.TenantID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate product performance: %w", err)
	}

	return performance, nil
}

func (s *analyticsService) CreateReport(ctx context.Context, req *models.CreateReportRequest) (*models.CustomReportResponse, error) {
	dateFrom, err := time.Parse("2006-01-02", req.DateFrom)
	if err != nil {
		return nil, fmt.Errorf("invalid date_from format, use YYYY-MM-DD")
	}
	dateTo, err := time.Parse("2006-01-02", req.DateTo)
	if err != nil {
		return nil, fmt.Errorf("invalid date_to format, use YYYY-MM-DD")
	}

	if dateTo.Before(dateFrom) {
		return nil, fmt.Errorf("date_to must be after date_from")
	}

	validTypes := map[string]bool{"sales": true, "customers": true, "products": true}
	if !validTypes[req.ReportType] {
		return nil, fmt.Errorf("invalid report_type, must be one of: sales, customers, products")
	}

	filtersJSON := "{}"
	if req.Filters != nil {
		b, _ := json.Marshal(req.Filters)
		filtersJSON = string(b)
	}

	report := &models.CustomReport{
		ID:         uuid.New().String(),
		TenantID:   req.TenantID,
		Name:       req.Name,
		ReportType: req.ReportType,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Filters:    filtersJSON,
		Status:     "pending",
	}

	if err := s.repo.CreateReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to create report: %w", err)
	}

	// Generate report inline
	s.generateReport(ctx, report)

	return toReportResponse(report), nil
}

func (s *analyticsService) generateReport(ctx context.Context, report *models.CustomReport) {
	var resultData interface{}
	var err error

	switch report.ReportType {
	case "sales":
		resultData, err = s.repo.GetSalesReport(ctx, report.TenantID, report.DateFrom, report.DateTo)
	case "customers":
		resultData, err = s.repo.GetCustomerInsights(ctx, report.TenantID, report.DateFrom, report.DateTo)
	case "products":
		resultData, err = s.repo.GetProductPerformance(ctx, report.TenantID, report.DateFrom, report.DateTo, 10)
	}

	if err != nil {
		report.Status = "failed"
		s.logger.WithError(err).Error("Failed to generate report")
	} else {
		b, _ := json.Marshal(resultData)
		report.ResultData = string(b)
		report.Status = "completed"
		now := time.Now().UTC()
		report.CompletedAt = &now
	}

	s.repo.UpdateReport(ctx, report)
}

func (s *analyticsService) GetReport(ctx context.Context, id string) (*models.CustomReportResponse, error) {
	report, err := s.repo.GetReport(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("report not found")
	}
	return toReportResponse(report), nil
}

func (s *analyticsService) ListReports(ctx context.Context, tenantID string, page, pageSize int) ([]models.CustomReportResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	reports, total, err := s.repo.ListReports(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.CustomReportResponse, len(reports))
	for i, r := range reports {
		resp := toReportResponse(&r)
		responses[i] = *resp
	}

	return responses, total, nil
}

// === Event Ingestion ===

func (s *analyticsService) RecordSale(ctx context.Context, tenantID, orderID, userID, vendorID string, amount float64, channel string) error {
	if channel == "" {
		channel = "web"
	}

	event := &models.SalesEvent{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		OrderID:  orderID,
		UserID:   userID,
		VendorID: vendorID,
		Amount:   amount,
		Channel:  channel,
		Status:   "completed",
	}

	if err := s.repo.RecordSalesEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to record sales event: %w", err)
	}

	// Also record as customer purchase event
	eventType := "purchase"
	s.RecordCustomerActivity(ctx, tenantID, userID, eventType, orderID, amount)

	return nil
}

func (s *analyticsService) RecordCustomerActivity(ctx context.Context, tenantID, userID, eventType, orderID string, amount float64) error {
	event := &models.CustomerEvent{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		UserID:    userID,
		EventType: eventType,
		OrderID:   orderID,
		Amount:    amount,
	}

	return s.repo.RecordCustomerEvent(ctx, event)
}

func (s *analyticsService) RecordProductActivity(ctx context.Context, tenantID, productID, eventType string, quantity int, revenue float64) error {
	event := &models.ProductEvent{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		ProductID: productID,
		EventType: eventType,
		Quantity:  quantity,
		Revenue:   revenue,
	}

	return s.repo.RecordProductEvent(ctx, event)
}

// === Helpers ===

func parseDateRange(fromStr, toStr string) (time.Time, time.Time) {
	now := time.Now().UTC()
	to := now

	// Default: last 30 days
	from := now.AddDate(0, 0, -30)

	if fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}
	if toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
	}

	return from, to
}

func toReportResponse(report *models.CustomReport) *models.CustomReportResponse {
	resp := &models.CustomReportResponse{
		ID:          report.ID,
		TenantID:    report.TenantID,
		Name:        report.Name,
		ReportType:  report.ReportType,
		DateFrom:    report.DateFrom,
		DateTo:      report.DateTo,
		Status:      report.Status,
		CreatedAt:   report.CreatedAt,
		CompletedAt: report.CompletedAt,
	}

	if report.Filters != "" && report.Filters != "{}" {
		var filters map[string]string
		if err := json.Unmarshal([]byte(report.Filters), &filters); err == nil {
			resp.Filters = filters
		}
	}

	if report.ResultData != "" {
		var result interface{}
		if err := json.Unmarshal([]byte(report.ResultData), &result); err == nil {
			resp.Result = result
		}
	}

	return resp
}
