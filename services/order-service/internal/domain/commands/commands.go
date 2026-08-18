package commands

import "github.com/yourusername/ecommerce/order-service/internal/domain/events"

// Command represents a command to be executed
type Command interface {
	GetAggregateID() string
}

// CreateOrderCommand creates a new order
type CreateOrderCommand struct {
	OrderID          string
	TenantID         string
	CustomerID       string
	GuestEmail       string
	GuestName        string
	GuestPhone       string
	ShippingAddress  events.Address
	BillingAddress   events.Address
}

func (c CreateOrderCommand) GetAggregateID() string { return c.OrderID }

// AddOrderItemCommand adds an item to an order
type AddOrderItemCommand struct {
	OrderID    string
	ProductID  string
	VariantID  string
	SKU        string
	Name       string
	Quantity   int
	UnitPrice  float64
}

func (c AddOrderItemCommand) GetAggregateID() string { return c.OrderID }

// RemoveOrderItemCommand removes an item from an order
type RemoveOrderItemCommand struct {
	OrderID string
	ItemID  string
}

func (c RemoveOrderItemCommand) GetAggregateID() string { return c.OrderID }

// ConfirmOrderCommand confirms an order
type ConfirmOrderCommand struct {
	OrderID     string
	ConfirmedBy string
}

func (c ConfirmOrderCommand) GetAggregateID() string { return c.OrderID }

// CancelOrderCommand cancels an order
type CancelOrderCommand struct {
	OrderID     string
	Reason      string
	CancelledBy string
}

func (c CancelOrderCommand) GetAggregateID() string { return c.OrderID }

// ShipOrderCommand marks an order as shipped
type ShipOrderCommand struct {
	OrderID        string
	TrackingNumber string
	Carrier        string
	ShippedBy      string
}

func (c ShipOrderCommand) GetAggregateID() string { return c.OrderID }

// DeliverOrderCommand marks an order as delivered
type DeliverOrderCommand struct {
	OrderID    string
	ReceivedBy string
}

func (c DeliverOrderCommand) GetAggregateID() string { return c.OrderID }

// RecordInventoryReservationCommand records a successful inventory reservation.
// The order saga issues this after the inventory service accepts the reservation
// so the event is persisted, projected and published like any other state change.
type RecordInventoryReservationCommand struct {
	OrderID       string
	ReservationID string
	Items         []events.ReservedItem
}

func (c RecordInventoryReservationCommand) GetAggregateID() string { return c.OrderID }

// RecordInventoryReleaseCommand records the release of a previously held
// inventory reservation (saga compensation).
type RecordInventoryReleaseCommand struct {
	OrderID       string
	ReservationID string
	Reason        string
}

func (c RecordInventoryReleaseCommand) GetAggregateID() string { return c.OrderID }

// RecordPaymentCommand records a successful payment against the order.
type RecordPaymentCommand struct {
	OrderID       string
	PaymentID     string
	PaymentMethod string
	TransactionID string
	Amount        float64
}

func (c RecordPaymentCommand) GetAggregateID() string { return c.OrderID }

// RecordPaymentFailureCommand records a failed or reversed payment.
type RecordPaymentFailureCommand struct {
	OrderID   string
	PaymentID string
	Reason    string
}

func (c RecordPaymentFailureCommand) GetAggregateID() string { return c.OrderID }
