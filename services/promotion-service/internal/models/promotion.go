package models

import "time"

// DiscountType defines how a discount is applied
type DiscountType string

const (
	DiscountPercentage DiscountType = "percentage"
	DiscountFixedAmount DiscountType = "fixed_amount"
	DiscountFreeShipping DiscountType = "free_shipping"
)

// PromotionStatus defines the lifecycle state
type PromotionStatus string

const (
	StatusDraft    PromotionStatus = "draft"
	StatusActive   PromotionStatus = "active"
	StatusExpired  PromotionStatus = "expired"
	StatusDisabled PromotionStatus = "disabled"
)

// Promotion represents a promotional campaign
type Promotion struct {
	ID              string          `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TenantID        string          `gorm:"type:varchar(36);index;not null" json:"tenant_id"`
	Name            string          `gorm:"type:varchar(255);not null" json:"name"`
	Description     string          `gorm:"type:text" json:"description"`
	DiscountType    DiscountType    `gorm:"type:varchar(50);not null" json:"discount_type"`
	DiscountValue   float64         `gorm:"not null" json:"discount_value"`
	MinOrderAmount  float64         `gorm:"default:0" json:"min_order_amount"`
	MaxDiscount     float64         `gorm:"default:0" json:"max_discount"`
	Status          PromotionStatus `gorm:"type:varchar(20);default:'draft'" json:"status"`
	StartDate       time.Time       `gorm:"not null" json:"start_date"`
	EndDate         time.Time       `gorm:"not null" json:"end_date"`
	ApplicableCategories []string   `gorm:"-" json:"applicable_categories,omitempty"`
	ApplicableProducts   []string   `gorm:"-" json:"applicable_products,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// Coupon represents a redeemable coupon code
type Coupon struct {
	ID            string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TenantID      string    `gorm:"type:varchar(36);index;not null" json:"tenant_id"`
	PromotionID   string    `gorm:"type:varchar(36);index;not null" json:"promotion_id"`
	Code          string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	MaxUses       int       `gorm:"default:0" json:"max_uses"`        // 0 = unlimited
	UsedCount     int       `gorm:"default:0" json:"used_count"`
	MaxUsesPerUser int      `gorm:"default:1" json:"max_uses_per_user"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CouponUsage tracks which users have used which coupons
type CouponUsage struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	CouponID  string    `gorm:"type:varchar(36);index;not null" json:"coupon_id"`
	UserID    string    `gorm:"type:varchar(36);index;not null" json:"user_id"`
	OrderID   string    `gorm:"type:varchar(36)" json:"order_id"`
	Discount  float64   `gorm:"not null" json:"discount"`
	UsedAt    time.Time `json:"used_at"`
}

// LoyaltyAccount holds a user's loyalty point balance
type LoyaltyAccount struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TenantID  string    `gorm:"type:varchar(36);index;not null" json:"tenant_id"`
	UserID    string    `gorm:"type:varchar(36);uniqueIndex;not null" json:"user_id"`
	Points    int       `gorm:"default:0" json:"points"`
	TierLevel string    `gorm:"type:varchar(20);default:'bronze'" json:"tier_level"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TransactionType for loyalty points
type TransactionType string

const (
	TransactionEarn   TransactionType = "earn"
	TransactionRedeem TransactionType = "redeem"
)

// LoyaltyTransaction records point changes
type LoyaltyTransaction struct {
	ID          string          `gorm:"primaryKey;type:varchar(36)" json:"id"`
	AccountID   string          `gorm:"type:varchar(36);index;not null" json:"account_id"`
	TenantID    string          `gorm:"type:varchar(36);index;not null" json:"tenant_id"`
	UserID      string          `gorm:"type:varchar(36);index;not null" json:"user_id"`
	Type        TransactionType `gorm:"type:varchar(20);not null" json:"type"`
	Points      int             `gorm:"not null" json:"points"`
	OrderID     string          `gorm:"type:varchar(36)" json:"order_id"`
	Description string          `gorm:"type:varchar(255)" json:"description"`
	CreatedAt   time.Time       `json:"created_at"`
}

// --- Request DTOs ---

type CreatePromotionRequest struct {
	TenantID       string       `json:"tenant_id" binding:"required"`
	Name           string       `json:"name" binding:"required"`
	Description    string       `json:"description"`
	DiscountType   DiscountType `json:"discount_type" binding:"required"`
	DiscountValue  float64      `json:"discount_value" binding:"required,gt=0"`
	MinOrderAmount float64      `json:"min_order_amount"`
	MaxDiscount    float64      `json:"max_discount"`
	StartDate      time.Time    `json:"start_date" binding:"required"`
	EndDate        time.Time    `json:"end_date" binding:"required"`
}

type CreateCouponRequest struct {
	TenantID       string `json:"tenant_id" binding:"required"`
	PromotionID    string `json:"promotion_id" binding:"required"`
	Code           string `json:"code" binding:"required,min=3,max=50"`
	MaxUses        int    `json:"max_uses"`
	MaxUsesPerUser int    `json:"max_uses_per_user"`
}

type ValidateCouponRequest struct {
	TenantID   string  `json:"tenant_id" binding:"required"`
	UserID     string  `json:"user_id" binding:"required"`
	OrderTotal float64 `json:"order_total" binding:"required,gt=0"`
}

type ApplyCouponRequest struct {
	TenantID   string  `json:"tenant_id" binding:"required"`
	UserID     string  `json:"user_id" binding:"required"`
	OrderID    string  `json:"order_id" binding:"required"`
	OrderTotal float64 `json:"order_total" binding:"required,gt=0"`
	Code       string  `json:"code" binding:"required"`
}

type LoyaltyPointsRequest struct {
	TenantID    string          `json:"tenant_id" binding:"required"`
	UserID      string          `json:"user_id" binding:"required"`
	Type        TransactionType `json:"type" binding:"required"`
	Points      int             `json:"points" binding:"required,gt=0"`
	OrderID     string          `json:"order_id"`
	Description string          `json:"description"`
}

// --- Response DTOs ---

type PromotionResponse struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	DiscountType   DiscountType    `json:"discount_type"`
	DiscountValue  float64         `json:"discount_value"`
	MinOrderAmount float64         `json:"min_order_amount"`
	MaxDiscount    float64         `json:"max_discount"`
	Status         PromotionStatus `json:"status"`
	StartDate      time.Time       `json:"start_date"`
	EndDate        time.Time       `json:"end_date"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CouponResponse struct {
	ID             string  `json:"id"`
	TenantID       string  `json:"tenant_id"`
	PromotionID    string  `json:"promotion_id"`
	Code           string  `json:"code"`
	MaxUses        int     `json:"max_uses"`
	UsedCount      int     `json:"used_count"`
	MaxUsesPerUser int     `json:"max_uses_per_user"`
	IsActive       bool    `json:"is_active"`
	DiscountType   DiscountType `json:"discount_type"`
	DiscountValue  float64 `json:"discount_value"`
}

type ValidateCouponResponse struct {
	Valid       bool         `json:"valid"`
	Code        string       `json:"code"`
	DiscountType DiscountType `json:"discount_type,omitempty"`
	DiscountValue float64    `json:"discount_value,omitempty"`
	DiscountAmount float64   `json:"discount_amount,omitempty"`
	Message     string       `json:"message,omitempty"`
}

type LoyaltyAccountResponse struct {
	UserID    string `json:"user_id"`
	TenantID  string `json:"tenant_id"`
	Points    int    `json:"points"`
	TierLevel string `json:"tier_level"`
}

// --- Kafka Event Models ---

type PromotionEvent struct {
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
