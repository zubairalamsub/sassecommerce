package eventstore

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
)

func TestPostgresEventStore_SaveAndGetEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, err := NewPostgresEventStore(db)
	require.NoError(t, err)

	aggregateID := "order-123"

	// Create test events
	testEvents := []events.Event{
		events.OrderCreated{
			BaseEvent: events.BaseEvent{
				ID:          "event-1",
				AggregateID: aggregateID,
				EventType:   events.OrderCreatedEvent,
				Timestamp:   time.Now(),
				Version:     1,
			},
			TenantID:   "tenant-123",
			CustomerID: "customer-123",
			Currency:   "BDT",
			ShippingAddr: events.Address{
				Street:     "123 Main St",
				City:       "Springfield",
				State:      "IL",
				PostalCode: "62701",
				Country:    "USA",
			},
			BillingAddr: events.Address{
				Street:     "456 Oak Ave",
				City:       "Springfield",
				State:      "IL",
				PostalCode: "62702",
				Country:    "USA",
			},
		},
		events.OrderItemAdded{
			BaseEvent: events.BaseEvent{
				ID:          "event-2",
				AggregateID: aggregateID,
				EventType:   events.OrderItemAddedEvent,
				Timestamp:   time.Now(),
				Version:     2,
			},
			ItemID:     "item-123",
			ProductID:  "product-123",
			VariantID:  "variant-123",
			SKU:        "SKU-001",
			Name:       "Test Product",
			Quantity:   2,
			UnitPrice:  99.99,
			TotalPrice: 199.98,
		},
	}

	// Save events
	err = store.Save(aggregateID, testEvents, -1)
	require.NoError(t, err)

	// Retrieve events
	retrievedEvents, err := store.GetEvents(aggregateID)
	require.NoError(t, err)
	assert.Len(t, retrievedEvents, 2)

	// Verify first event
	assert.Equal(t, events.OrderCreatedEvent, retrievedEvents[0].GetEventType())
	assert.Equal(t, aggregateID, retrievedEvents[0].GetAggregateID())

	// Verify second event
	assert.Equal(t, events.OrderItemAddedEvent, retrievedEvents[1].GetEventType())
	assert.Equal(t, aggregateID, retrievedEvents[1].GetAggregateID())
}

func TestPostgresEventStore_OptimisticConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, err := NewPostgresEventStore(db)
	require.NoError(t, err)

	aggregateID := "order-456"

	// Save first event
	event1 := events.OrderCreated{
		BaseEvent: events.BaseEvent{
			ID:          "event-1",
			AggregateID: aggregateID,
			EventType:   events.OrderCreatedEvent,
			Timestamp:   time.Now(),
			Version:     1,
		},
		TenantID:   "tenant-123",
		CustomerID: "customer-123",
		Currency:   "BDT",
	}

	err = store.Save(aggregateID, []events.Event{event1}, -1)
	require.NoError(t, err)

	// Try to save with wrong expected version
	event2 := events.OrderItemAdded{
		BaseEvent: events.BaseEvent{
			ID:          "event-2",
			AggregateID: aggregateID,
			EventType:   events.OrderItemAddedEvent,
			Timestamp:   time.Now(),
			Version:     2,
		},
		ItemID:    "item-123",
		ProductID: "product-123",
	}

	// This should fail because expected version is -1 but current is 0
	err = store.Save(aggregateID, []events.Event{event2}, -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "concurrency conflict")

	// This should succeed with correct expected version
	err = store.Save(aggregateID, []events.Event{event2}, 0)
	assert.NoError(t, err)
}

func TestPostgresEventStore_GetEventsByType(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, err := NewPostgresEventStore(db)
	require.NoError(t, err)

	// Save multiple events of different types
	for i := 0; i < 5; i++ {
		aggregateID := "order-" + string(rune(i))

		event := events.OrderCreated{
			BaseEvent: events.BaseEvent{
				ID:          "event-" + string(rune(i)),
				AggregateID: aggregateID,
				EventType:   events.OrderCreatedEvent,
				Timestamp:   time.Now(),
				Version:     1,
			},
			TenantID:   "tenant-123",
			CustomerID: "customer-" + string(rune(i)),
			Currency:   "BDT",
		}

		err = store.Save(aggregateID, []events.Event{event}, -1)
		require.NoError(t, err)
	}

	// Get events by type
	retrievedEvents, err := store.GetEventsByType(events.OrderCreatedEvent, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(retrievedEvents), 5)
}

// Helper functions

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Use environment variable or default
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/order_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping integration test: cannot open database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		t.Skipf("Skipping integration test: database not available: %v", err)
	}

	// Cleanup function
	cleanup := func() {
		// Clean up test data
		db.Exec("DROP TABLE IF EXISTS events")
		db.Close()
	}

	return db, cleanup
}
