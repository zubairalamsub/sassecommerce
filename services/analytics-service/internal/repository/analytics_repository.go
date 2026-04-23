package repository

import (
	"context"
	"time"

	"github.com/ecommerce/analytics-service/internal/models"
	"gorm.io/gorm"
)

type AnalyticsRepository interface {
	// Sales events
	RecordSalesEvent(ctx context.Context, event *models.SalesEvent) error
	GetSalesReport(ctx context.Context, tenantID string, from, to time.Time) (*models.SalesReportResponse, error)
	GetDailySales(ctx context.Context, tenantID string, from, to time.Time) ([]models.DailySales, error)

	// Customer events
	RecordCustomerEvent(ctx context.Context, event *models.CustomerEvent) error
	GetCustomerInsights(ctx context.Context, tenantID string, from, to time.Time) (*models.CustomerInsightsResponse, error)

	// Product events
	RecordProductEvent(ctx context.Context, event *models.ProductEvent) error
	GetProductPerformance(ctx context.Context, tenantID string, from, to time.Time, limit int) (*models.ProductPerformanceResponse, error)

	// Custom reports
	CreateReport(ctx context.Context, report *models.CustomReport) error
	GetReport(ctx context.Context, id string) (*models.CustomReport, error)
	ListReports(ctx context.Context, tenantID string, page, pageSize int) ([]models.CustomReport, int64, error)
	UpdateReport(ctx context.Context, report *models.CustomReport) error
}

type analyticsRepository struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) RecordSalesEvent(ctx context.Context, event *models.SalesEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *analyticsRepository) GetSalesReport(ctx context.Context, tenantID string, from, to time.Time) (*models.SalesReportResponse, error) {
	var totalRevenue float64
	var totalOrders int64

	query := r.db.WithContext(ctx).Model(&models.SalesEvent{}).Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, from, to)

	if err := query.Count(&totalOrders).Error; err != nil {
		return nil, err
	}

	if err := query.Select("COALESCE(SUM(amount), 0)").Row().Scan(&totalRevenue); err != nil {
		return nil, err
	}

	avgOrder := 0.0
	if totalOrders > 0 {
		avgOrder = totalRevenue / float64(totalOrders)
	}

	// Top products from product events
	var topProducts []models.ProductSalesSummary
	r.db.WithContext(ctx).Model(&models.ProductEvent{}).
		Select("product_id, SUM(quantity) as total_sold, SUM(revenue) as total_revenue").
		Where("tenant_id = ? AND event_type = 'product_sold' AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Group("product_id").
		Order("total_revenue DESC").
		Limit(10).
		Find(&topProducts)

	// Revenue by channel
	revenueByChannel := make(map[string]float64)
	type channelResult struct {
		Channel string
		Total   float64
	}
	var channels []channelResult
	r.db.WithContext(ctx).Model(&models.SalesEvent{}).
		Select("channel, SUM(amount) as total").
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Group("channel").
		Find(&channels)
	for _, ch := range channels {
		revenueByChannel[ch.Channel] = ch.Total
	}

	return &models.SalesReportResponse{
		TenantID:         tenantID,
		TotalRevenue:     totalRevenue,
		TotalOrders:      totalOrders,
		AverageOrder:     avgOrder,
		TopProducts:      topProducts,
		RevenueByChannel: revenueByChannel,
	}, nil
}

func (r *analyticsRepository) GetDailySales(ctx context.Context, tenantID string, from, to time.Time) ([]models.DailySales, error) {
	var results []models.DailySales

	r.db.WithContext(ctx).Model(&models.SalesEvent{}).
		Select("DATE(created_at) as date, SUM(amount) as revenue, COUNT(*) as orders").
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&results)

	return results, nil
}

func (r *analyticsRepository) RecordCustomerEvent(ctx context.Context, event *models.CustomerEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *analyticsRepository) GetCustomerInsights(ctx context.Context, tenantID string, from, to time.Time) (*models.CustomerInsightsResponse, error) {
	var totalCustomers int64
	r.db.WithContext(ctx).Model(&models.CustomerEvent{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Distinct("user_id").
		Count(&totalCustomers)

	var newCustomers int64
	r.db.WithContext(ctx).Model(&models.CustomerEvent{}).
		Where("tenant_id = ? AND event_type = 'first_purchase' AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Count(&newCustomers)

	returningCustomers := totalCustomers - newCustomers
	if returningCustomers < 0 {
		returningCustomers = 0
	}

	var avgOrderValue float64
	r.db.WithContext(ctx).Model(&models.CustomerEvent{}).
		Where("tenant_id = ? AND event_type IN ('purchase', 'first_purchase') AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Select("COALESCE(AVG(amount), 0)").
		Row().Scan(&avgOrderValue)

	// Top customers
	var topCustomers []models.CustomerSummary
	r.db.WithContext(ctx).Model(&models.CustomerEvent{}).
		Select("user_id, COUNT(*) as total_orders, SUM(amount) as total_spent").
		Where("tenant_id = ? AND event_type IN ('purchase', 'first_purchase') AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Group("user_id").
		Order("total_spent DESC").
		Limit(10).
		Find(&topCustomers)

	// Customer segments by spend
	segments := buildCustomerSegments(topCustomers, totalCustomers)

	return &models.CustomerInsightsResponse{
		TenantID:           tenantID,
		TotalCustomers:     totalCustomers,
		NewCustomers:       newCustomers,
		ReturningCustomers: returningCustomers,
		AverageOrderValue:  avgOrderValue,
		TopCustomers:       topCustomers,
		CustomerSegments:   segments,
	}, nil
}

func buildCustomerSegments(customers []models.CustomerSummary, total int64) []models.CustomerSegment {
	if total == 0 {
		return []models.CustomerSegment{}
	}

	var high, medium, low int64
	for _, c := range customers {
		switch {
		case c.TotalSpent >= 1000:
			high++
		case c.TotalSpent >= 100:
			medium++
		default:
			low++
		}
	}

	return []models.CustomerSegment{
		{Segment: "high_value", Count: high, Percentage: float64(high) / float64(total) * 100},
		{Segment: "medium_value", Count: medium, Percentage: float64(medium) / float64(total) * 100},
		{Segment: "low_value", Count: low, Percentage: float64(low) / float64(total) * 100},
	}
}

func (r *analyticsRepository) RecordProductEvent(ctx context.Context, event *models.ProductEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *analyticsRepository) GetProductPerformance(ctx context.Context, tenantID string, from, to time.Time, limit int) (*models.ProductPerformanceResponse, error) {
	if limit <= 0 {
		limit = 10
	}

	var totalProducts int64
	r.db.WithContext(ctx).Model(&models.ProductEvent{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Distinct("product_id").
		Count(&totalProducts)

	var topSelling []models.ProductPerformance
	r.db.WithContext(ctx).Model(&models.ProductEvent{}).
		Select("product_id, SUM(quantity) as total_sold, SUM(revenue) as total_revenue").
		Where("tenant_id = ? AND event_type = 'product_sold' AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Group("product_id").
		Order("total_sold DESC").
		Limit(limit).
		Find(&topSelling)

	var lowPerforming []models.ProductPerformance
	r.db.WithContext(ctx).Model(&models.ProductEvent{}).
		Select("product_id, SUM(quantity) as total_sold, SUM(revenue) as total_revenue").
		Where("tenant_id = ? AND event_type = 'product_sold' AND created_at BETWEEN ? AND ?", tenantID, from, to).
		Group("product_id").
		Order("total_sold ASC").
		Limit(5).
		Find(&lowPerforming)

	return &models.ProductPerformanceResponse{
		TenantID:      tenantID,
		TotalProducts: totalProducts,
		TopSelling:    topSelling,
		LowPerforming: lowPerforming,
	}, nil
}

func (r *analyticsRepository) CreateReport(ctx context.Context, report *models.CustomReport) error {
	return r.db.WithContext(ctx).Create(report).Error
}

func (r *analyticsRepository) GetReport(ctx context.Context, id string) (*models.CustomReport, error) {
	var report models.CustomReport
	if err := r.db.WithContext(ctx).First(&report, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *analyticsRepository) ListReports(ctx context.Context, tenantID string, page, pageSize int) ([]models.CustomReport, int64, error) {
	var reports []models.CustomReport
	var total int64

	query := r.db.WithContext(ctx).Model(&models.CustomReport{}).Where("tenant_id = ?", tenantID)
	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&reports).Error; err != nil {
		return nil, 0, err
	}

	return reports, total, nil
}

func (r *analyticsRepository) UpdateReport(ctx context.Context, report *models.CustomReport) error {
	return r.db.WithContext(ctx).Save(report).Error
}
