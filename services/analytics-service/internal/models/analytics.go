package models

import (
	"encoding/json"
	"time"
)

// === Database Models ===

type SalesEvent struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID  string    `json:"tenant_id" gorm:"type:varchar(36);index"`
	OrderID   string    `json:"order_id" gorm:"type:varchar(36);index"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);index"`
	VendorID  string    `json:"vendor_id" gorm:"type:varchar(36);index"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency" gorm:"type:varchar(10);default:'BDT'"`
	Status    string    `json:"status" gorm:"type:varchar(20)"`
	Channel   string    `json:"channel" gorm:"type:varchar(20);default:'web'"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type CustomerEvent struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID  string    `json:"tenant_id" gorm:"type:varchar(36);index"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);index"`
	EventType string    `json:"event_type" gorm:"type:varchar(50)"`
	OrderID   string    `json:"order_id" gorm:"type:varchar(36)"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type ProductEvent struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID  string    `json:"tenant_id" gorm:"type:varchar(36);index"`
	ProductID string    `json:"product_id" gorm:"type:varchar(36);index"`
	EventType string    `json:"event_type" gorm:"type:varchar(50)"`
	Quantity  int       `json:"quantity"`
	Revenue   float64   `json:"revenue"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type CustomReport struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID    string    `json:"tenant_id" gorm:"type:varchar(36);index"`
	Name        string    `json:"name" gorm:"type:varchar(200)"`
	ReportType  string    `json:"report_type" gorm:"type:varchar(50)"`
	DateFrom    time.Time `json:"date_from"`
	DateTo      time.Time `json:"date_to"`
	Filters     string    `json:"filters" gorm:"type:text"`
	ResultData  string    `json:"result_data" gorm:"type:text"`
	Status      string    `json:"status" gorm:"type:varchar(20);default:'pending'"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	CompletedAt *time.Time `json:"completed_at"`
}

// === Response DTOs ===

type SalesReportResponse struct {
	TenantID       string              `json:"tenant_id"`
	Period         string              `json:"period"`
	TotalRevenue   float64             `json:"total_revenue"`
	TotalOrders    int64               `json:"total_orders"`
	AverageOrder   float64             `json:"average_order_value"`
	TopProducts    []ProductSalesSummary `json:"top_products,omitempty"`
	DailySales     []DailySales        `json:"daily_sales,omitempty"`
	RevenueByChannel map[string]float64 `json:"revenue_by_channel,omitempty"`
}

type DailySales struct {
	Date     string  `json:"date"`
	Revenue  float64 `json:"revenue"`
	Orders   int64   `json:"orders"`
}

type ProductSalesSummary struct {
	ProductID    string  `json:"product_id"`
	TotalSold    int     `json:"total_sold"`
	TotalRevenue float64 `json:"total_revenue"`
}

type CustomerInsightsResponse struct {
	TenantID          string             `json:"tenant_id"`
	TotalCustomers    int64              `json:"total_customers"`
	NewCustomers      int64              `json:"new_customers"`
	ReturningCustomers int64             `json:"returning_customers"`
	AverageOrderValue float64            `json:"average_order_value"`
	TopCustomers      []CustomerSummary  `json:"top_customers,omitempty"`
	CustomerSegments  []CustomerSegment  `json:"customer_segments,omitempty"`
}

type CustomerSummary struct {
	UserID      string  `json:"user_id"`
	TotalOrders int64   `json:"total_orders"`
	TotalSpent  float64 `json:"total_spent"`
}

type CustomerSegment struct {
	Segment   string `json:"segment"`
	Count     int64  `json:"count"`
	Percentage float64 `json:"percentage"`
}

type ProductPerformanceResponse struct {
	TenantID      string                  `json:"tenant_id"`
	TotalProducts int64                   `json:"total_products"`
	TopSelling    []ProductPerformance    `json:"top_selling"`
	LowPerforming []ProductPerformance    `json:"low_performing,omitempty"`
	CategoryBreakdown []CategoryPerformance `json:"category_breakdown,omitempty"`
}

type ProductPerformance struct {
	ProductID    string  `json:"product_id"`
	TotalSold    int     `json:"total_sold"`
	TotalRevenue float64 `json:"total_revenue"`
	ViewCount    int     `json:"view_count"`
}

type CategoryPerformance struct {
	Category     string  `json:"category"`
	TotalSold    int     `json:"total_sold"`
	TotalRevenue float64 `json:"total_revenue"`
}

// === Request DTOs ===

type SalesReportRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
	Period   string `json:"period"` // daily, weekly, monthly
}

type CustomerInsightsRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
}

type ProductPerformanceRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
	Limit    int    `json:"limit"`
}

type CreateReportRequest struct {
	TenantID   string            `json:"tenant_id" binding:"required"`
	Name       string            `json:"name" binding:"required"`
	ReportType string            `json:"report_type" binding:"required"`
	DateFrom   string            `json:"date_from" binding:"required"`
	DateTo     string            `json:"date_to" binding:"required"`
	Filters    map[string]string `json:"filters"`
}

type CustomReportResponse struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id"`
	Name        string            `json:"name"`
	ReportType  string            `json:"report_type"`
	DateFrom    time.Time         `json:"date_from"`
	DateTo      time.Time         `json:"date_to"`
	Filters     map[string]string `json:"filters,omitempty"`
	Result      interface{}       `json:"result,omitempty"`
	Status      string            `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
}

// === Kafka Event Envelope ===

type EventEnvelope struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string           `json:"source"`
	Payload  json.RawMessage  `json:"payload,omitempty"`
	Data     json.RawMessage  `json:"data,omitempty"`
}

func (e *EventEnvelope) GetPayload() map[string]interface{} {
	raw := e.Payload
	if len(raw) == 0 {
		raw = e.Data
	}
	if len(raw) == 0 {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result
}
