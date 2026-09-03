package api

// CreateOrderRequest represents a request to create an order.
// For guest checkout, customer_id may be omitted and guest contact fields used instead.
type CreateOrderRequest struct {
	TenantID        string  `json:"tenant_id" binding:"required"`
	CustomerID      string  `json:"customer_id"`
	GuestEmail      string  `json:"guest_email"`
	GuestName       string  `json:"guest_name"`
	GuestPhone      string  `json:"guest_phone"`
	ShippingAddress Address `json:"shipping_address" binding:"required"`
	BillingAddress  Address `json:"billing_address" binding:"required"`
	// Items lets a caller submit the whole order in one request. This is the
	// only way a guest can attach items: POST /orders/:id/items requires auth
	// deliberately, because an unauthenticated route keyed only by order id
	// would let anyone holding an id mutate someone else's pending order.
	// Optional — staff/POS flows still create an empty order and add items
	// through the authenticated route.
	Items []AddOrderItemRequest `json:"items"`
}

// AddOrderItemRequest represents a request to add an item to an order
type AddOrderItemRequest struct {
	ProductID string  `json:"product_id" binding:"required"`
	VariantID string  `json:"variant_id"`
	SKU       string  `json:"sku" binding:"required"`
	Name      string  `json:"name" binding:"required"`
	Quantity  int     `json:"quantity" binding:"required,min=1"`
	UnitPrice float64 `json:"unit_price" binding:"required,min=0"`
}

// RemoveOrderItemRequest represents a request to remove an item from an order
type RemoveOrderItemRequest struct {
	ItemID string `json:"item_id" binding:"required"`
}

// ConfirmOrderRequest represents a request to confirm an order
type ConfirmOrderRequest struct {
	ConfirmedBy string `json:"confirmed_by" binding:"required"`
}

// CancelOrderRequest represents a request to cancel an order
type CancelOrderRequest struct {
	Reason      string `json:"reason" binding:"required"`
	CancelledBy string `json:"cancelled_by" binding:"required"`
}

// ShipOrderRequest represents a request to ship an order
type ShipOrderRequest struct {
	TrackingNumber string `json:"tracking_number" binding:"required"`
	Carrier        string `json:"carrier" binding:"required"`
	ShippedBy      string `json:"shipped_by" binding:"required"`
}

// DeliverOrderRequest represents a request to mark an order as delivered
type DeliverOrderRequest struct {
	ReceivedBy string `json:"received_by" binding:"required"`
}

// SendReceiptRequest is the body for POST /api/v1/orders/:id/send-receipt.
// `Email` is the only required field — the rest are presentational
// overrides for the email body. When omitted, the notification-service
// consumer falls back to per-tenant defaults.
type SendReceiptRequest struct {
	Email         string `json:"email" binding:"required,email"`
	CustomerName  string `json:"customer_name,omitempty"`
	StoreName     string `json:"store_name,omitempty"`
	PaymentMethod string `json:"payment_method,omitempty"`
}

// Address represents a shipping or billing address
type Address struct {
	Street     string `json:"street" binding:"required"`
	City       string `json:"city" binding:"required"`
	State      string `json:"state" binding:"required"`
	PostalCode string `json:"postal_code" binding:"required"`
	Country    string `json:"country" binding:"required"`
}

// CreateOrderResponse represents a response after creating an order
type CreateOrderResponse struct {
	OrderID string `json:"order_id"`
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message"`
}
