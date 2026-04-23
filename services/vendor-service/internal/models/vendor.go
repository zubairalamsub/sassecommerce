package models

import "time"

// VendorStatus defines the lifecycle state of a vendor
type VendorStatus string

const (
	StatusPending   VendorStatus = "pending"
	StatusApproved  VendorStatus = "approved"
	StatusSuspended VendorStatus = "suspended"
	StatusRejected  VendorStatus = "rejected"
)

// Vendor represents a marketplace vendor/seller
type Vendor struct {
	ID              string       `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TenantID        string       `gorm:"type:varchar(36);index;not null" json:"tenant_id"`
	Name            string       `gorm:"type:varchar(255);not null" json:"name"`
	Email           string       `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Phone           string       `gorm:"type:varchar(50)" json:"phone"`
	Description     string       `gorm:"type:text" json:"description"`
	LogoURL         string       `gorm:"type:varchar(500)" json:"logo_url"`
	Address         string       `gorm:"type:text" json:"address"`
	City            string       `gorm:"type:varchar(100)" json:"city"`
	Country         string       `gorm:"type:varchar(100)" json:"country"`
	Status          VendorStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	CommissionRate  float64      `gorm:"default:10" json:"commission_rate"` // percentage
	TotalRevenue    float64      `gorm:"default:0" json:"total_revenue"`
	TotalOrders     int          `gorm:"default:0" json:"total_orders"`
	TotalProducts   int          `gorm:"default:0" json:"total_products"`
	Rating          float64      `gorm:"default:0" json:"rating"`
	SuspendReason   string       `gorm:"type:text" json:"suspend_reason,omitempty"`
	ApprovedAt      *time.Time   `json:"approved_at,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// VendorOrder tracks orders assigned to a vendor
type VendorOrder struct {
	ID         string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	VendorID   string    `gorm:"type:varchar(36);index;not null" json:"vendor_id"`
	TenantID   string    `gorm:"type:varchar(36);index;not null" json:"tenant_id"`
	OrderID    string    `gorm:"type:varchar(36);index;not null" json:"order_id"`
	Amount     float64   `gorm:"not null" json:"amount"`
	Commission float64   `gorm:"not null" json:"commission"`
	NetAmount  float64   `gorm:"not null" json:"net_amount"`
	Status     string    `gorm:"type:varchar(50);default:'pending'" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// --- Request DTOs ---

type RegisterVendorRequest struct {
	TenantID    string  `json:"tenant_id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Email       string  `json:"email" binding:"required,email"`
	Phone       string  `json:"phone"`
	Description string  `json:"description"`
	LogoURL     string  `json:"logo_url"`
	Address     string  `json:"address"`
	City        string  `json:"city"`
	Country     string  `json:"country"`
	CommissionRate float64 `json:"commission_rate"`
}

type UpdateVendorRequest struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	Address     string `json:"address"`
	City        string `json:"city"`
	Country     string `json:"country"`
}

type UpdateVendorStatusRequest struct {
	Status VendorStatus `json:"status" binding:"required"`
	Reason string       `json:"reason"`
}

// --- Response DTOs ---

type VendorResponse struct {
	ID             string       `json:"id"`
	TenantID       string       `json:"tenant_id"`
	Name           string       `json:"name"`
	Email          string       `json:"email"`
	Phone          string       `json:"phone"`
	Description    string       `json:"description"`
	LogoURL        string       `json:"logo_url"`
	Address        string       `json:"address"`
	City           string       `json:"city"`
	Country        string       `json:"country"`
	Status         VendorStatus `json:"status"`
	CommissionRate float64      `json:"commission_rate"`
	TotalRevenue   float64      `json:"total_revenue"`
	TotalOrders    int          `json:"total_orders"`
	TotalProducts  int          `json:"total_products"`
	Rating         float64      `json:"rating"`
	ApprovedAt     *time.Time   `json:"approved_at,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
}

type VendorOrderResponse struct {
	ID         string    `json:"id"`
	VendorID   string    `json:"vendor_id"`
	OrderID    string    `json:"order_id"`
	Amount     float64   `json:"amount"`
	Commission float64   `json:"commission"`
	NetAmount  float64   `json:"net_amount"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type VendorAnalyticsResponse struct {
	VendorID       string  `json:"vendor_id"`
	TotalRevenue   float64 `json:"total_revenue"`
	TotalOrders    int     `json:"total_orders"`
	TotalProducts  int     `json:"total_products"`
	CommissionPaid float64 `json:"commission_paid"`
	NetEarnings    float64 `json:"net_earnings"`
	Rating         float64 `json:"rating"`
}

// --- Kafka Event Models ---

type VendorEvent struct {
	EventID   string      `json:"event_id"`
	EventType string      `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type EventEnvelope struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
	Data      map[string]interface{} `json:"data"`
}

func (e *EventEnvelope) GetPayload() map[string]interface{} {
	if e.Payload != nil {
		return e.Payload
	}
	return e.Data
}
