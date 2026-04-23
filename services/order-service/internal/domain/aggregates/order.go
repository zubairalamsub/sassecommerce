package aggregates

import (
	"errors"
	"time"

	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
)

// OrderStatus represents the current status of an order
type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusConfirmed OrderStatus = "confirmed"
	StatusShipped   OrderStatus = "shipped"
	StatusDelivered OrderStatus = "delivered"
	StatusCancelled OrderStatus = "cancelled"
)

// OrderItem represents an item in the order
type OrderItem struct {
	ID         string
	ProductID  string
	VariantID  string
	SKU        string
	Name       string
	Quantity   int
	UnitPrice  float64
	TotalPrice float64
}

// Order is the aggregate root for order domain
type Order struct {
	// Aggregate ID
	ID string

	// Current state (built from events)
	TenantID         string
	CustomerID       string
	Status           OrderStatus
	Items            map[string]*OrderItem
	TotalAmount      float64
	Currency         string
	ShippingAddress  events.Address
	BillingAddress   events.Address
	PaymentID        string
	ReservationID    string
	TrackingNumber   string
	Carrier          string
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// Event sourcing fields
	Version        int
	UncommittedEvents []events.Event
}

// NewOrder creates a new order aggregate
func NewOrder(id, tenantID, customerID string, shippingAddr, billingAddr events.Address) *Order {
	order := &Order{
		ID:                id,
		Items:             make(map[string]*OrderItem),
		UncommittedEvents: make([]events.Event, 0),
	}

	// Raise OrderCreated event
	event := events.OrderCreated{
		BaseEvent:    events.NewBaseEvent(id, events.OrderCreatedEvent, 1),
		TenantID:     tenantID,
		CustomerID:   customerID,
		TotalAmount:  0,
		Currency:     "BDT",
		ShippingAddr: shippingAddr,
		BillingAddr:  billingAddr,
	}

	order.raise(event)
	return order
}

// AddItem adds an item to the order
func (o *Order) AddItem(productID, variantID, sku, name string, quantity int, unitPrice float64) error {
	if o.Status != StatusPending {
		return errors.New("cannot add items to non-pending order")
	}

	if quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}

	if unitPrice <= 0 {
		return errors.New("unit price must be greater than 0")
	}

	itemID := productID
	if variantID != "" {
		itemID = productID + "-" + variantID
	}

	totalPrice := float64(quantity) * unitPrice

	event := events.OrderItemAdded{
		BaseEvent:  events.NewBaseEvent(o.ID, events.OrderItemAddedEvent, o.Version+1),
		ItemID:     itemID,
		ProductID:  productID,
		VariantID:  variantID,
		SKU:        sku,
		Name:       name,
		Quantity:   quantity,
		UnitPrice:  unitPrice,
		TotalPrice: totalPrice,
	}

	o.raise(event)
	return nil
}

// RemoveItem removes an item from the order
func (o *Order) RemoveItem(itemID string) error {
	if o.Status != StatusPending {
		return errors.New("cannot remove items from non-pending order")
	}

	if _, exists := o.Items[itemID]; !exists {
		return errors.New("item not found in order")
	}

	event := events.OrderItemRemoved{
		BaseEvent: events.NewBaseEvent(o.ID, events.OrderItemRemovedEvent, o.Version+1),
		ItemID:    itemID,
	}

	o.raise(event)
	return nil
}

// Confirm confirms the order
func (o *Order) Confirm(confirmedBy string) error {
	if o.Status != StatusPending {
		return errors.New("can only confirm pending orders")
	}

	if len(o.Items) == 0 {
		return errors.New("cannot confirm order with no items")
	}

	event := events.OrderConfirmed{
		BaseEvent:   events.NewBaseEvent(o.ID, events.OrderConfirmedEvent, o.Version+1),
		ConfirmedBy: confirmedBy,
		ConfirmedAt: time.Now().UTC(),
	}

	o.raise(event)
	return nil
}

// Cancel cancels the order
func (o *Order) Cancel(reason, cancelledBy string) error {
	if o.Status == StatusCancelled {
		return errors.New("order already cancelled")
	}

	if o.Status == StatusDelivered {
		return errors.New("cannot cancel delivered order")
	}

	event := events.OrderCancelled{
		BaseEvent:   events.NewBaseEvent(o.ID, events.OrderCancelledEvent, o.Version+1),
		Reason:      reason,
		CancelledBy: cancelledBy,
		CancelledAt: time.Now().UTC(),
	}

	o.raise(event)
	return nil
}

// Ship marks the order as shipped
func (o *Order) Ship(trackingNumber, carrier, shippedBy string) error {
	if o.Status != StatusConfirmed {
		return errors.New("can only ship confirmed orders")
	}

	event := events.OrderShipped{
		BaseEvent:      events.NewBaseEvent(o.ID, events.OrderShippedEvent, o.Version+1),
		TrackingNumber: trackingNumber,
		Carrier:        carrier,
		ShippedAt:      time.Now().UTC(),
		ShippedBy:      shippedBy,
	}

	o.raise(event)
	return nil
}

// Deliver marks the order as delivered
func (o *Order) Deliver(receivedBy string) error {
	if o.Status != StatusShipped {
		return errors.New("can only deliver shipped orders")
	}

	event := events.OrderDelivered{
		BaseEvent:   events.NewBaseEvent(o.ID, events.OrderDeliveredEvent, o.Version+1),
		DeliveredAt: time.Now().UTC(),
		ReceivedBy:  receivedBy,
	}

	o.raise(event)
	return nil
}

// RecordPayment records a successful payment
func (o *Order) RecordPayment(paymentID, paymentMethod, transactionID string, amount float64) error {
	event := events.PaymentProcessed{
		BaseEvent:     events.NewBaseEvent(o.ID, events.PaymentProcessedEvent, o.Version+1),
		PaymentID:     paymentID,
		Amount:        amount,
		PaymentMethod: paymentMethod,
		TransactionID: transactionID,
		ProcessedAt:   time.Now().UTC(),
	}

	o.raise(event)
	return nil
}

// RecordPaymentFailure records a failed payment
func (o *Order) RecordPaymentFailure(paymentID, reason string) error {
	event := events.PaymentFailed{
		BaseEvent: events.NewBaseEvent(o.ID, events.PaymentFailedEvent, o.Version+1),
		PaymentID: paymentID,
		Reason:    reason,
		FailedAt:  time.Now().UTC(),
	}

	o.raise(event)
	return nil
}

// RecordInventoryReservation records inventory reservation
func (o *Order) RecordInventoryReservation(reservationID string, items []events.ReservedItem) error {
	event := events.InventoryReserved{
		BaseEvent:     events.NewBaseEvent(o.ID, events.InventoryReservedEvent, o.Version+1),
		ReservationID: reservationID,
		Items:         items,
		ReservedAt:    time.Now().UTC(),
	}

	o.raise(event)
	return nil
}

// RecordInventoryRelease records inventory release
func (o *Order) RecordInventoryRelease(reservationID, reason string) error {
	event := events.InventoryReleased{
		BaseEvent:     events.NewBaseEvent(o.ID, events.InventoryReleasedEvent, o.Version+1),
		ReservationID: reservationID,
		Reason:        reason,
		ReleasedAt:    time.Now().UTC(),
	}

	o.raise(event)
	return nil
}

// GetUncommittedEvents returns events that haven't been persisted yet
func (o *Order) GetUncommittedEvents() []events.Event {
	return o.UncommittedEvents
}

// MarkEventsAsCommitted clears uncommitted events after persistence
func (o *Order) MarkEventsAsCommitted() {
	o.UncommittedEvents = make([]events.Event, 0)
}

// LoadFromHistory rebuilds aggregate state from event history
func (o *Order) LoadFromHistory(eventHistory []events.Event) {
	for _, event := range eventHistory {
		o.apply(event)
		o.Version = event.GetVersion()
	}
}

// raise applies event and adds to uncommitted events
func (o *Order) raise(event events.Event) {
	o.apply(event)
	o.UncommittedEvents = append(o.UncommittedEvents, event)
	o.Version = event.GetVersion()
}

// apply applies event to aggregate state
func (o *Order) apply(event events.Event) {
	switch e := event.(type) {
	case events.OrderCreated:
		o.TenantID = e.TenantID
		o.CustomerID = e.CustomerID
		o.TotalAmount = e.TotalAmount
		o.Currency = e.Currency
		o.ShippingAddress = e.ShippingAddr
		o.BillingAddress = e.BillingAddr
		o.Status = StatusPending
		o.CreatedAt = e.Timestamp
		o.UpdatedAt = e.Timestamp

	case events.OrderItemAdded:
		o.Items[e.ItemID] = &OrderItem{
			ID:         e.ItemID,
			ProductID:  e.ProductID,
			VariantID:  e.VariantID,
			SKU:        e.SKU,
			Name:       e.Name,
			Quantity:   e.Quantity,
			UnitPrice:  e.UnitPrice,
			TotalPrice: e.TotalPrice,
		}
		o.recalculateTotal()
		o.UpdatedAt = e.Timestamp

	case events.OrderItemRemoved:
		delete(o.Items, e.ItemID)
		o.recalculateTotal()
		o.UpdatedAt = e.Timestamp

	case events.OrderConfirmed:
		o.Status = StatusConfirmed
		o.UpdatedAt = e.Timestamp

	case events.OrderCancelled:
		o.Status = StatusCancelled
		o.UpdatedAt = e.Timestamp

	case events.OrderShipped:
		o.Status = StatusShipped
		o.TrackingNumber = e.TrackingNumber
		o.Carrier = e.Carrier
		o.UpdatedAt = e.Timestamp

	case events.OrderDelivered:
		o.Status = StatusDelivered
		o.UpdatedAt = e.Timestamp

	case events.PaymentProcessed:
		o.PaymentID = e.PaymentID
		o.UpdatedAt = e.Timestamp

	case events.PaymentFailed:
		o.UpdatedAt = e.Timestamp

	case events.InventoryReserved:
		o.ReservationID = e.ReservationID
		o.UpdatedAt = e.Timestamp

	case events.InventoryReleased:
		o.ReservationID = ""
		o.UpdatedAt = e.Timestamp
	}
}

// recalculateTotal recalculates the total amount from items
func (o *Order) recalculateTotal() {
	total := 0.0
	for _, item := range o.Items {
		total += item.TotalPrice
	}
	o.TotalAmount = total
}
