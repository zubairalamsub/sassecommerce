package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of domain event
type EventType string

const (
	OrderCreatedEvent       EventType = "OrderCreated"
	OrderConfirmedEvent     EventType = "OrderConfirmed"
	OrderCancelledEvent     EventType = "OrderCancelled"
	OrderShippedEvent       EventType = "OrderShipped"
	OrderDeliveredEvent     EventType = "OrderDelivered"
	OrderItemAddedEvent     EventType = "OrderItemAdded"
	OrderItemRemovedEvent   EventType = "OrderItemRemoved"
	PaymentProcessedEvent   EventType = "PaymentProcessed"
	PaymentFailedEvent      EventType = "PaymentFailed"
	InventoryReservedEvent  EventType = "InventoryReserved"
	InventoryReleasedEvent  EventType = "InventoryReleased"
)

// Event is the base interface for all domain events
type Event interface {
	GetID() string
	GetAggregateID() string
	GetEventType() EventType
	GetTimestamp() time.Time
	GetVersion() int
	GetData() interface{}
}

// BaseEvent contains common fields for all events
type BaseEvent struct {
	ID          string    `json:"id"`
	AggregateID string    `json:"aggregate_id"`
	EventType   EventType `json:"event_type"`
	Timestamp   time.Time `json:"timestamp"`
	Version     int       `json:"version"`
}

func (e BaseEvent) GetID() string           { return e.ID }
func (e BaseEvent) GetAggregateID() string  { return e.AggregateID }
func (e BaseEvent) GetEventType() EventType { return e.EventType }
func (e BaseEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e BaseEvent) GetVersion() int         { return e.Version }

// OrderCreated event
type OrderCreated struct {
	BaseEvent
	TenantID      string  `json:"tenant_id"`
	CustomerID    string  `json:"customer_id"`
	TotalAmount   float64 `json:"total_amount"`
	Currency      string  `json:"currency"`
	ShippingAddr  Address `json:"shipping_address"`
	BillingAddr   Address `json:"billing_address"`
}

func (e OrderCreated) GetData() interface{} { return e }

// OrderConfirmed event
type OrderConfirmed struct {
	BaseEvent
	ConfirmedBy string    `json:"confirmed_by"`
	ConfirmedAt time.Time `json:"confirmed_at"`
}

func (e OrderConfirmed) GetData() interface{} { return e }

// OrderCancelled event
type OrderCancelled struct {
	BaseEvent
	Reason      string    `json:"reason"`
	CancelledBy string    `json:"cancelled_by"`
	CancelledAt time.Time `json:"cancelled_at"`
}

func (e OrderCancelled) GetData() interface{} { return e }

// OrderShipped event
type OrderShipped struct {
	BaseEvent
	TrackingNumber string    `json:"tracking_number"`
	Carrier        string    `json:"carrier"`
	ShippedAt      time.Time `json:"shipped_at"`
	ShippedBy      string    `json:"shipped_by"`
}

func (e OrderShipped) GetData() interface{} { return e }

// OrderDelivered event
type OrderDelivered struct {
	BaseEvent
	DeliveredAt time.Time `json:"delivered_at"`
	ReceivedBy  string    `json:"received_by"`
}

func (e OrderDelivered) GetData() interface{} { return e }

// OrderItemAdded event
type OrderItemAdded struct {
	BaseEvent
	ItemID     string  `json:"item_id"`
	ProductID  string  `json:"product_id"`
	VariantID  string  `json:"variant_id,omitempty"`
	SKU        string  `json:"sku"`
	Name       string  `json:"name"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
}

func (e OrderItemAdded) GetData() interface{} { return e }

// OrderItemRemoved event
type OrderItemRemoved struct {
	BaseEvent
	ItemID string `json:"item_id"`
}

func (e OrderItemRemoved) GetData() interface{} { return e }

// PaymentProcessed event
type PaymentProcessed struct {
	BaseEvent
	PaymentID       string    `json:"payment_id"`
	Amount          float64   `json:"amount"`
	PaymentMethod   string    `json:"payment_method"`
	TransactionID   string    `json:"transaction_id"`
	ProcessedAt     time.Time `json:"processed_at"`
}

func (e PaymentProcessed) GetData() interface{} { return e }

// PaymentFailed event
type PaymentFailed struct {
	BaseEvent
	PaymentID string    `json:"payment_id"`
	Reason    string    `json:"reason"`
	FailedAt  time.Time `json:"failed_at"`
}

func (e PaymentFailed) GetData() interface{} { return e }

// InventoryReserved event
type InventoryReserved struct {
	BaseEvent
	ReservationID string          `json:"reservation_id"`
	Items         []ReservedItem  `json:"items"`
	ReservedAt    time.Time       `json:"reserved_at"`
}

func (e InventoryReserved) GetData() interface{} { return e }

// InventoryReleased event
type InventoryReleased struct {
	BaseEvent
	ReservationID string    `json:"reservation_id"`
	Reason        string    `json:"reason"`
	ReleasedAt    time.Time `json:"released_at"`
}

func (e InventoryReleased) GetData() interface{} { return e }

// Supporting types
type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type ReservedItem struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id,omitempty"`
	Quantity  int    `json:"quantity"`
}

// NewBaseEvent creates a new base event
func NewBaseEvent(aggregateID string, eventType EventType, version int) BaseEvent {
	return BaseEvent{
		ID:          uuid.New().String(),
		AggregateID: aggregateID,
		EventType:   eventType,
		Timestamp:   time.Now().UTC(),
		Version:     version,
	}
}

// Serialize serializes an event to JSON
func Serialize(event Event) ([]byte, error) {
	return json.Marshal(event)
}

// Deserialize deserializes an event from JSON
func Deserialize(eventType EventType, data []byte) (Event, error) {
	var event Event

	switch eventType {
	case OrderCreatedEvent:
		var e OrderCreated
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		event = e
	case OrderConfirmedEvent:
		var e OrderConfirmed
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		event = e
	case OrderCancelledEvent:
		var e OrderCancelled
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		event = e
	case OrderShippedEvent:
		var e OrderShipped
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		event = e
	case OrderDeliveredEvent:
		var e OrderDelivered
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		event = e
	case OrderItemAddedEvent:
		var e OrderItemAdded
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		event = e
	case OrderItemRemovedEvent:
		var e OrderItemRemoved
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		event = e
	case PaymentProcessedEvent:
		var e PaymentProcessed
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		event = e
	case PaymentFailedEvent:
		var e PaymentFailed
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		event = e
	case InventoryReservedEvent:
		var e InventoryReserved
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		event = e
	case InventoryReleasedEvent:
		var e InventoryReleased
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		event = e
	}

	return event, nil
}
