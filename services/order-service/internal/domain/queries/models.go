package queries

import "time"

// OrderReadModel represents the read-optimized order model
type OrderReadModel struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	CustomerID       string    `json:"customer_id"`
	Status           string    `json:"status"`
	TotalAmount      float64   `json:"total_amount"`
	Currency         string    `json:"currency"`
	ShippingAddress  Address   `json:"shipping_address"`
	BillingAddress   Address   `json:"billing_address"`
	PaymentID        string    `json:"payment_id,omitempty"`
	ReservationID    string    `json:"reservation_id,omitempty"`
	TrackingNumber   string    `json:"tracking_number,omitempty"`
	Carrier          string    `json:"carrier,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Version          int       `json:"version"`
}

// OrderItemReadModel represents an order item in the read model
type OrderItemReadModel struct {
	ID         string  `json:"id"`
	OrderID    string  `json:"order_id"`
	ProductID  string  `json:"product_id"`
	VariantID  string  `json:"variant_id,omitempty"`
	SKU        string  `json:"sku"`
	Name       string  `json:"name"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
}

// OrderSummary represents a summary of an order for list views
type OrderSummary struct {
	ID          string    `json:"id"`
	CustomerID  string    `json:"customer_id"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"total_amount"`
	Currency    string    `json:"currency"`
	ItemCount   int       `json:"item_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Address represents an address
type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}
