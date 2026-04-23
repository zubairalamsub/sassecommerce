package models

import (
	"time"

	"github.com/google/uuid"
)

// ShipmentStatus represents the lifecycle of a shipment
type ShipmentStatus string

const (
	StatusPending      ShipmentStatus = "pending"
	StatusLabelCreated ShipmentStatus = "label_created"
	StatusPickedUp     ShipmentStatus = "picked_up"
	StatusInTransit    ShipmentStatus = "in_transit"
	StatusOutForDelivery ShipmentStatus = "out_for_delivery"
	StatusDelivered    ShipmentStatus = "delivered"
	StatusFailed       ShipmentStatus = "failed"
	StatusReturned     ShipmentStatus = "returned"
	StatusCancelled    ShipmentStatus = "cancelled"
)

// Carrier constants (Bangladesh local carriers)
const (
	CarrierPathao     = "pathao"
	CarrierSteadfast  = "steadfast"
	CarrierRedX       = "redx"
	CarrierPaperfly   = "paperfly"
	CarrierSundarban  = "sundarban"
	CarrierSAParibahan = "sa_paribahan"
)

// Shipment represents a shipping order
type Shipment struct {
	ID        string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	TenantID  string         `gorm:"type:varchar(100);not null;index" json:"tenant_id"`
	OrderID   string         `gorm:"type:varchar(100);not null;index" json:"order_id"`

	// Carrier & tracking
	Carrier        string `gorm:"type:varchar(50);not null" json:"carrier"`
	TrackingNumber string `gorm:"type:varchar(200);index" json:"tracking_number"`
	ServiceType    string `gorm:"type:varchar(100)" json:"service_type"` // ground, express, overnight
	LabelURL       string `gorm:"type:varchar(500)" json:"label_url,omitempty"`

	// Status
	Status        ShipmentStatus `gorm:"type:varchar(50);not null;default:'pending';index" json:"status"`
	FailureReason string         `gorm:"type:varchar(500)" json:"failure_reason,omitempty"`

	// Weight & dimensions (metric)
	WeightOz     float64 `gorm:"type:decimal(10,2)" json:"weight_kg"`
	LengthIn     float64 `gorm:"type:decimal(10,2)" json:"length_cm"`
	WidthIn      float64 `gorm:"type:decimal(10,2)" json:"width_cm"`
	HeightIn     float64 `gorm:"type:decimal(10,2)" json:"height_cm"`

	// Cost
	ShippingCost float64 `gorm:"type:decimal(18,2)" json:"shipping_cost"`
	Currency     string  `gorm:"type:varchar(3);default:'BDT'" json:"currency"`
	InsuredValue float64 `gorm:"type:decimal(18,2)" json:"insured_value"`

	// Addresses
	FromName       string `gorm:"type:varchar(200)" json:"from_name"`
	FromStreet     string `gorm:"type:varchar(200)" json:"from_street"`
	FromCity       string `gorm:"type:varchar(100)" json:"from_city"`
	FromState      string `gorm:"type:varchar(100)" json:"from_state"`
	FromPostalCode string `gorm:"type:varchar(20)" json:"from_postal_code"`
	FromCountry    string `gorm:"type:varchar(100)" json:"from_country"`

	ToName       string `gorm:"type:varchar(200)" json:"to_name"`
	ToStreet     string `gorm:"type:varchar(200)" json:"to_street"`
	ToCity       string `gorm:"type:varchar(100)" json:"to_city"`
	ToState      string `gorm:"type:varchar(100)" json:"to_state"`
	ToPostalCode string `gorm:"type:varchar(20)" json:"to_postal_code"`
	ToCountry    string `gorm:"type:varchar(100)" json:"to_country"`

	// Estimated & actual delivery
	EstimatedDelivery *time.Time `json:"estimated_delivery,omitempty"`
	ActualDelivery    *time.Time `json:"actual_delivery,omitempty"`
	ShippedAt         *time.Time `json:"shipped_at,omitempty"`
	DeliveredAt       *time.Time `json:"delivered_at,omitempty"`

	// Signature
	SignatureRequired bool   `gorm:"default:false" json:"signature_required"`
	SignedBy          string `gorm:"type:varchar(200)" json:"signed_by,omitempty"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Items  []ShipmentItem  `gorm:"foreignKey:ShipmentID" json:"items,omitempty"`
	Events []ShipmentEvent `gorm:"foreignKey:ShipmentID" json:"events,omitempty"`
}

func (Shipment) TableName() string {
	return "shipments"
}

func (s *Shipment) BeforeCreate() error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

// ShipmentItem represents an item within a shipment
type ShipmentItem struct {
	ID         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ShipmentID string `gorm:"type:uuid;not null;index" json:"shipment_id"`
	ProductID  string `gorm:"type:varchar(100);not null" json:"product_id"`
	VariantID  string `gorm:"type:varchar(100)" json:"variant_id,omitempty"`
	SKU        string `gorm:"type:varchar(100)" json:"sku"`
	Name       string `gorm:"type:varchar(255)" json:"name"`
	Quantity   int    `gorm:"not null" json:"quantity"`
	WeightOz   float64 `gorm:"type:decimal(10,2)" json:"weight_oz"`

	CreatedAt time.Time `json:"created_at"`
}

func (ShipmentItem) TableName() string {
	return "shipment_items"
}

func (si *ShipmentItem) BeforeCreate() error {
	if si.ID == "" {
		si.ID = uuid.New().String()
	}
	return nil
}

// ShipmentEvent represents a tracking event in the shipment lifecycle
type ShipmentEvent struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ShipmentID  string    `gorm:"type:uuid;not null;index" json:"shipment_id"`
	Status      string    `gorm:"type:varchar(50);not null" json:"status"`
	Location    string    `gorm:"type:varchar(200)" json:"location,omitempty"`
	Description string    `gorm:"type:varchar(500)" json:"description"`
	OccurredAt  time.Time `json:"occurred_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (ShipmentEvent) TableName() string {
	return "shipment_events"
}

func (se *ShipmentEvent) BeforeCreate() error {
	if se.ID == "" {
		se.ID = uuid.New().String()
	}
	return nil
}

// === Request DTOs ===

type CreateShipmentRequest struct {
	TenantID  string                `json:"tenant_id" binding:"required"`
	OrderID   string                `json:"order_id" binding:"required"`
	Carrier   string                `json:"carrier" binding:"required"`
	ServiceType string              `json:"service_type"`
	WeightOz  float64               `json:"weight_oz"`
	LengthIn  float64               `json:"length_in"`
	WidthIn   float64               `json:"width_in"`
	HeightIn  float64               `json:"height_in"`

	FromAddress AddressRequest       `json:"from_address" binding:"required"`
	ToAddress   AddressRequest       `json:"to_address" binding:"required"`
	Items       []ShipmentItemRequest `json:"items"`

	InsuredValue      float64 `json:"insured_value"`
	SignatureRequired bool    `json:"signature_required"`
	CreatedBy         string  `json:"created_by"`
}

type AddressRequest struct {
	Name       string `json:"name" binding:"required"`
	Street     string `json:"street" binding:"required"`
	City       string `json:"city" binding:"required"`
	State      string `json:"state" binding:"required"`
	PostalCode string `json:"postal_code" binding:"required"`
	Country    string `json:"country" binding:"required"`
}

type ShipmentItemRequest struct {
	ProductID string  `json:"product_id" binding:"required"`
	VariantID string  `json:"variant_id,omitempty"`
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity" binding:"required,min=1"`
	WeightOz  float64 `json:"weight_oz"`
}

type CalculateRateRequest struct {
	TenantID    string         `json:"tenant_id" binding:"required"`
	FromAddress AddressRequest `json:"from_address" binding:"required"`
	ToAddress   AddressRequest `json:"to_address" binding:"required"`
	WeightOz    float64        `json:"weight_oz" binding:"required"`
	LengthIn    float64        `json:"length_in"`
	WidthIn     float64        `json:"width_in"`
	HeightIn    float64        `json:"height_in"`
}

type UpdateStatusRequest struct {
	Status      string `json:"status" binding:"required"`
	Location    string `json:"location"`
	Description string `json:"description"`
	SignedBy    string `json:"signed_by"`
}

// === Response DTOs ===

type ShipmentResponse struct {
	ID                string         `json:"id"`
	TenantID          string         `json:"tenant_id"`
	OrderID           string         `json:"order_id"`
	Carrier           string         `json:"carrier"`
	TrackingNumber    string         `json:"tracking_number"`
	ServiceType       string         `json:"service_type"`
	LabelURL          string         `json:"label_url,omitempty"`
	Status            ShipmentStatus `json:"status"`
	FailureReason     string         `json:"failure_reason,omitempty"`
	WeightOz          float64        `json:"weight_oz"`
	ShippingCost      float64        `json:"shipping_cost"`
	Currency          string         `json:"currency"`
	FromAddress       AddressResponse `json:"from_address"`
	ToAddress         AddressResponse `json:"to_address"`
	EstimatedDelivery *time.Time     `json:"estimated_delivery,omitempty"`
	ActualDelivery    *time.Time     `json:"actual_delivery,omitempty"`
	ShippedAt         *time.Time     `json:"shipped_at,omitempty"`
	DeliveredAt       *time.Time     `json:"delivered_at,omitempty"`
	SignatureRequired bool           `json:"signature_required"`
	SignedBy          string         `json:"signed_by,omitempty"`
	Items             []ShipmentItemResponse  `json:"items,omitempty"`
	Events            []ShipmentEventResponse `json:"events,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type AddressResponse struct {
	Name       string `json:"name"`
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type ShipmentItemResponse struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	VariantID string  `json:"variant_id,omitempty"`
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	WeightOz  float64 `json:"weight_oz"`
}

type ShipmentEventResponse struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Location    string    `json:"location,omitempty"`
	Description string    `json:"description"`
	OccurredAt  time.Time `json:"occurred_at"`
}

type CarrierRateResponse struct {
	Carrier         string  `json:"carrier"`
	ServiceType     string  `json:"service_type"`
	Rate            float64 `json:"rate"`
	Currency        string  `json:"currency"`
	EstimatedDays   int     `json:"estimated_days"`
}

type RateCalculationResponse struct {
	Rates []CarrierRateResponse `json:"rates"`
}
