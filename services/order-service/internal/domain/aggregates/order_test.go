package aggregates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
)

func TestNewOrder(t *testing.T) {
	orderID := "order-123"
	tenantID := "tenant-123"
	customerID := "customer-123"

	shippingAddr := events.Address{
		Street:     "House 12, Road 5, Dhanmondi",
		City:       "Dhaka",
		State:      "Dhaka",
		PostalCode: "1205",
		Country:    "BD",
	}

	billingAddr := events.Address{
		Street:     "House 8, Road 3, Gulshan",
		City:       "Dhaka",
		State:      "Dhaka",
		PostalCode: "1212",
		Country:    "BD",
	}

	order := NewOrder(orderID, tenantID, customerID, shippingAddr, billingAddr)

	assert.NotNil(t, order)
	assert.Equal(t, orderID, order.ID)
	assert.Equal(t, tenantID, order.TenantID)
	assert.Equal(t, customerID, order.CustomerID)
	assert.Equal(t, StatusPending, order.Status)
	assert.Equal(t, "BDT", order.Currency)
	assert.Equal(t, 0.0, order.TotalAmount)
	assert.Equal(t, 1, order.Version)
	assert.Len(t, order.UncommittedEvents, 1)

	// Verify event
	event := order.UncommittedEvents[0]
	assert.IsType(t, events.OrderCreated{}, event)

	orderCreated := event.(events.OrderCreated)
	assert.Equal(t, orderID, orderCreated.AggregateID)
	assert.Equal(t, tenantID, orderCreated.TenantID)
	assert.Equal(t, customerID, orderCreated.CustomerID)
}

func TestAddItem(t *testing.T) {
	order := createTestOrder()

	err := order.AddItem(
		"product-123",
		"variant-123",
		"SKU-001",
		"Test Product",
		2,
		99.99,
	)

	assert.NoError(t, err)
	assert.Len(t, order.Items, 1)
	assert.Equal(t, 199.98, order.TotalAmount)
	assert.Len(t, order.UncommittedEvents, 2) // OrderCreated + OrderItemAdded
}

func TestAddItem_InvalidQuantity(t *testing.T) {
	order := createTestOrder()

	err := order.AddItem(
		"product-123",
		"variant-123",
		"SKU-001",
		"Test Product",
		0,
		99.99,
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quantity must be greater than 0")
}

func TestAddItem_InvalidPrice(t *testing.T) {
	order := createTestOrder()

	err := order.AddItem(
		"product-123",
		"variant-123",
		"SKU-001",
		"Test Product",
		2,
		-10.0,
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unit price must be greater than 0")
}

func TestRemoveItem(t *testing.T) {
	order := createTestOrder()

	// Add item first
	order.AddItem("product-123", "variant-123", "SKU-001", "Test Product", 2, 99.99)
	itemID := ""
	for id := range order.Items {
		itemID = id
		break
	}

	// Remove item
	err := order.RemoveItem(itemID)

	assert.NoError(t, err)
	assert.Len(t, order.Items, 0)
	assert.Equal(t, 0.0, order.TotalAmount)
}

func TestConfirm(t *testing.T) {
	order := createTestOrder()
	order.AddItem("product-123", "variant-123", "SKU-001", "Test Product", 1, 99.99)

	err := order.Confirm("admin@example.com")

	assert.NoError(t, err)
	assert.Equal(t, StatusConfirmed, order.Status)
}

func TestConfirm_NoItems(t *testing.T) {
	order := createTestOrder()

	err := order.Confirm("admin@example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot confirm order with no items")
}

func TestCancel(t *testing.T) {
	order := createTestOrder()

	err := order.Cancel("Customer request", "customer@example.com")

	assert.NoError(t, err)
	assert.Equal(t, StatusCancelled, order.Status)
}

func TestShip(t *testing.T) {
	order := createTestOrder()
	order.AddItem("product-123", "variant-123", "SKU-001", "Test Product", 1, 99.99)
	order.Confirm("admin@example.com")

	err := order.Ship("TRACK123", "FedEx", "warehouse@example.com")

	assert.NoError(t, err)
	assert.Equal(t, StatusShipped, order.Status)
	assert.Equal(t, "TRACK123", order.TrackingNumber)
	assert.Equal(t, "FedEx", order.Carrier)
}

func TestShip_NotConfirmed(t *testing.T) {
	order := createTestOrder()

	err := order.Ship("TRACK123", "FedEx", "warehouse@example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only ship confirmed orders")
}

func TestDeliver(t *testing.T) {
	order := createTestOrder()
	order.AddItem("product-123", "variant-123", "SKU-001", "Test Product", 1, 99.99)
	order.Confirm("admin@example.com")
	order.Ship("TRACK123", "FedEx", "warehouse@example.com")

	err := order.Deliver("customer@example.com")

	assert.NoError(t, err)
	assert.Equal(t, StatusDelivered, order.Status)
}

func TestDeliver_NotShipped(t *testing.T) {
	order := createTestOrder()

	err := order.Deliver("customer@example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only deliver shipped orders")
}

func TestLoadFromHistory(t *testing.T) {
	// Create order and add items
	order1 := createTestOrder()
	order1.AddItem("product-123", "variant-123", "SKU-001", "Test Product", 2, 99.99)
	order1.Confirm("admin@example.com")

	// Get events
	eventHistory := order1.UncommittedEvents

	// Create new order and load from history
	order2 := &Order{
		ID:    order1.ID,
		Items: make(map[string]*OrderItem),
	}
	order2.LoadFromHistory(eventHistory)

	// Verify state
	assert.Equal(t, order1.TenantID, order2.TenantID)
	assert.Equal(t, order1.CustomerID, order2.CustomerID)
	assert.Equal(t, order1.Status, order2.Status)
	assert.Equal(t, order1.TotalAmount, order2.TotalAmount)
	assert.Len(t, order2.Items, 1)
	assert.Equal(t, order1.Version, order2.Version)
}

func TestRecordPayment(t *testing.T) {
	order := createTestOrder()

	err := order.RecordPayment("payment-123", "credit_card", "txn-123", 199.98)

	assert.NoError(t, err)
	assert.Equal(t, "payment-123", order.PaymentID)
	assert.Len(t, order.UncommittedEvents, 2) // OrderCreated + PaymentProcessed
}

func TestRecordInventoryReservation(t *testing.T) {
	order := createTestOrder()

	reservedItems := []events.ReservedItem{
		{
			ProductID: "product-123",
			VariantID: "variant-123",
			Quantity:  2,
		},
	}

	err := order.RecordInventoryReservation("reservation-123", reservedItems)

	assert.NoError(t, err)
	assert.Equal(t, "reservation-123", order.ReservationID)
	assert.Len(t, order.UncommittedEvents, 2) // OrderCreated + InventoryReserved
}

// Helper functions

func createTestOrder() *Order {
	return NewOrder(
		"order-123",
		"tenant-123",
		"customer-123",
		events.Address{
			Street:     "House 12, Road 5, Dhanmondi",
			City:       "Dhaka",
			State:      "Dhaka",
			PostalCode: "1205",
			Country:    "BD",
		},
		events.Address{
			Street:     "House 8, Road 3, Gulshan",
			City:       "Dhaka",
			State:      "Dhaka",
			PostalCode: "1212",
			Country:    "BD",
		},
	)
}
