package models

import "time"

// CartItem represents a single item in the cart
type CartItem struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	ImageURL  string  `json:"image_url,omitempty"`
	AddedAt   string  `json:"added_at"`
}

// Cart represents a user's shopping cart stored in Redis
type Cart struct {
	TenantID  string     `json:"tenant_id"`
	UserID    string     `json:"user_id"`
	Items     []CartItem `json:"items"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TotalAmount calculates the total cost of all items in the cart
func (c *Cart) TotalAmount() float64 {
	total := 0.0
	for _, item := range c.Items {
		total += item.Price * float64(item.Quantity)
	}
	return total
}

// TotalItems returns the total number of items (sum of quantities)
func (c *Cart) TotalItems() int {
	total := 0
	for _, item := range c.Items {
		total += item.Quantity
	}
	return total
}

// --- Request DTOs ---

type AddItemRequest struct {
	TenantID  string  `json:"tenant_id" binding:"required"`
	UserID    string  `json:"user_id" binding:"required"`
	ProductID string  `json:"product_id" binding:"required"`
	Name      string  `json:"name" binding:"required"`
	Price     float64 `json:"price" binding:"required,gt=0"`
	Quantity  int     `json:"quantity" binding:"required,min=1,max=100"`
	ImageURL  string  `json:"image_url"`
}

type UpdateItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1,max=100"`
}

// --- Response DTOs ---

type CartResponse struct {
	TenantID    string             `json:"tenant_id"`
	UserID      string             `json:"user_id"`
	Items       []CartItemResponse `json:"items"`
	TotalItems  int                `json:"total_items"`
	TotalAmount float64            `json:"total_amount"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type CartItemResponse struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	Subtotal  float64 `json:"subtotal"`
	ImageURL  string  `json:"image_url,omitempty"`
	AddedAt   string  `json:"added_at"`
}

// --- Kafka Event Models ---

type CartEvent struct {
	EventID   string      `json:"event_id"`
	EventType string      `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type CartUpdatedPayload struct {
	TenantID    string  `json:"tenant_id"`
	UserID      string  `json:"user_id"`
	TotalItems  int     `json:"total_items"`
	TotalAmount float64 `json:"total_amount"`
}

type CartAbandonedPayload struct {
	TenantID    string  `json:"tenant_id"`
	UserID      string  `json:"user_id"`
	TotalItems  int     `json:"total_items"`
	TotalAmount float64 `json:"total_amount"`
}

// EventEnvelope for consuming events from other services
type EventEnvelope struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
	Data      map[string]interface{} `json:"data"`
}

// GetPayload returns the event payload, checking both "payload" and "data" fields
func (e *EventEnvelope) GetPayload() map[string]interface{} {
	if e.Payload != nil {
		return e.Payload
	}
	return e.Data
}
