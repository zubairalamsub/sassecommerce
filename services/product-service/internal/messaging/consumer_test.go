package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ecommerce/product-service/internal/mocks"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

func newTestConsumer(repo *mocks.MockProductRepository) *EventConsumer {
	return &EventConsumer{productRepo: repo, logger: newDiscardLogger()}
}

func marshalEnvelope(t *testing.T, envelope EventEnvelope) []byte {
	t.Helper()
	value, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return value
}

func assertUpdateStockNotCalled(t *testing.T, repo *mocks.MockProductRepository) {
	t.Helper()
	repo.AssertNotCalled(t, "UpdateStock", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// === EventEnvelope.GetPayload ===

func TestEventEnvelope_GetPayload(t *testing.T) {
	tests := []struct {
		name     string
		envelope EventEnvelope
		wantFrom interface{}
	}{
		{
			name:     "prefers payload over data",
			envelope: EventEnvelope{Payload: map[string]interface{}{"from": "payload"}, Data: map[string]interface{}{"from": "data"}},
			wantFrom: "payload",
		},
		{
			name:     "falls back to data when payload is nil",
			envelope: EventEnvelope{Data: map[string]interface{}{"from": "data"}},
			wantFrom: "data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.envelope.GetPayload()
			if got["from"] != tt.wantFrom {
				t.Errorf("GetPayload()[from] = %v, want %v", got["from"], tt.wantFrom)
			}
		})
	}

	t.Run("both nil returns nil", func(t *testing.T) {
		if got := (&EventEnvelope{}).GetPayload(); got != nil {
			t.Errorf("GetPayload() = %v, want nil", got)
		}
	})
}

// === handleMessage: event-type routing and malformed input ===

func TestHandleMessage_RoutesKnownInventoryEventTypesToUpdateStock(t *testing.T) {
	for _, eventType := range []string{"InventoryUpdated", "StockLevelChanged"} {
		t.Run(eventType, func(t *testing.T) {
			repo := new(mocks.MockProductRepository)
			repo.On("UpdateStock", mock.Anything, "tenant-1", "prod-1", 5, true).Return(nil)
			c := newTestConsumer(repo)

			value := marshalEnvelope(t, EventEnvelope{
				EventID: "evt-1", EventType: eventType,
				Payload: map[string]interface{}{"tenant_id": "tenant-1", "product_id": "prod-1", "quantity": float64(5)},
			})

			c.handleMessage(context.Background(), kafka.Message{Value: value})

			repo.AssertExpectations(t)
		})
	}
}

func TestHandleMessage_IgnoresUnrelatedEventTypes(t *testing.T) {
	repo := new(mocks.MockProductRepository)
	c := newTestConsumer(repo)

	value := marshalEnvelope(t, EventEnvelope{
		EventType: "ProductViewed",
		Payload:   map[string]interface{}{"tenant_id": "tenant-1", "product_id": "prod-1"},
	})

	c.handleMessage(context.Background(), kafka.Message{Value: value})

	assertUpdateStockNotCalled(t, repo)
}

func TestHandleMessage_MalformedJSONIsDroppedNotPanicked(t *testing.T) {
	repo := new(mocks.MockProductRepository)
	c := newTestConsumer(repo)

	// Must not panic.
	c.handleMessage(context.Background(), kafka.Message{Value: []byte("{not json")})

	assertUpdateStockNotCalled(t, repo)
}

// EventEnvelope.Version used to be a plain string here, and this test asserted
// the consequence: a numeric version made json.Unmarshal return a type error,
// so handleMessage returned before dispatching on EventType and a legitimate
// stock update was dropped in silence. That is the shape order-service's own
// publisher emits, and the shape that took out order-event consumption
// platform-wide once already.
//
// The field is now sharedkafka.EventVersion, which accepts both shapes
// producers here emit, so the assertion is inverted: the event must be
// processed. Kept rather than deleted, because a silently dropped stock update
// is exactly what it guards against.
//
// Version is never read in this consumer. It is declared only so an unexpected
// shape cannot reject the envelope.
func TestHandleMessage_NumericVersionIsStillProcessed(t *testing.T) {
	repo := new(mocks.MockProductRepository)
	c := newTestConsumer(repo)
	repo.On("UpdateStock", mock.Anything, "tenant-1", "prod-1", 5, true).Return(nil)

	raw := []byte(`{"event_id":"evt-1","event_type":"InventoryUpdated","version":1,"payload":{"tenant_id":"tenant-1","product_id":"prod-1","quantity":5}}`)

	c.handleMessage(context.Background(), kafka.Message{Value: raw})

	repo.AssertCalled(t, "UpdateStock", mock.Anything, "tenant-1", "prod-1", 5, true)
}

func TestHandleMessage_StringVersionDecodesFine(t *testing.T) {
	// Pins today's real payload shape (version as a JSON string) as the
	// contrast to the numeric-version case above: same event, only the
	// version's wire shape differs, and only one of them survives decoding.
	repo := new(mocks.MockProductRepository)
	repo.On("UpdateStock", mock.Anything, "tenant-1", "prod-1", 5, true).Return(nil)
	c := newTestConsumer(repo)

	raw := []byte(`{"event_id":"evt-1","event_type":"InventoryUpdated","version":"1.0.0","payload":{"tenant_id":"tenant-1","product_id":"prod-1","quantity":5}}`)

	c.handleMessage(context.Background(), kafka.Message{Value: raw})

	repo.AssertExpectations(t)
}

// === handleInventoryUpdated ===

func TestHandleInventoryUpdated_NilPayloadDoesNotCallRepo(t *testing.T) {
	repo := new(mocks.MockProductRepository)
	c := newTestConsumer(repo)

	c.handleInventoryUpdated(context.Background(), &EventEnvelope{EventType: "InventoryUpdated"})

	assertUpdateStockNotCalled(t, repo)
}

func TestHandleInventoryUpdated_MissingProductIDDoesNotCallRepo(t *testing.T) {
	repo := new(mocks.MockProductRepository)
	c := newTestConsumer(repo)

	c.handleInventoryUpdated(context.Background(), &EventEnvelope{
		Payload: map[string]interface{}{"tenant_id": "tenant-1", "quantity": float64(5)},
	})

	assertUpdateStockNotCalled(t, repo)
}

// Real finding: only product_id gates the update. tenant_id is read with a
// bare type assertion (payload["tenant_id"].(string)) that silently defaults
// to "" rather than being validated, unlike product_id which is checked for
// emptiness. On a multi-tenant platform where every other write path enforces
// tenant scoping, a malformed or spoofed inventory event missing tenant_id
// still reaches the repository -- just scoped to an empty tenant string,
// rather than being rejected outright. Whether that is harmless depends on
// how the repository's Mongo filter treats an empty tenant_id, which lives
// outside this package. Flagged in the QA report.
func TestHandleInventoryUpdated_MissingTenantIDStillCallsRepoWithEmptyTenant(t *testing.T) {
	repo := new(mocks.MockProductRepository)
	repo.On("UpdateStock", mock.Anything, "", "prod-1", 5, true).Return(nil)
	c := newTestConsumer(repo)

	c.handleInventoryUpdated(context.Background(), &EventEnvelope{
		Payload: map[string]interface{}{"product_id": "prod-1", "quantity": float64(5)},
	})

	repo.AssertExpectations(t)
}

func TestHandleInventoryUpdated_QuantityDerivesInStockWhenNotOverridden(t *testing.T) {
	tests := []struct {
		name     string
		quantity interface{}
		wantQty  int
		wantIn   bool
	}{
		{name: "positive quantity implies in stock", quantity: float64(5), wantQty: 5, wantIn: true},
		{name: "zero quantity implies out of stock", quantity: float64(0), wantQty: 0, wantIn: false},
		{name: "missing quantity defaults to zero and out of stock", quantity: nil, wantQty: 0, wantIn: false},
		{name: "non-numeric quantity is ignored and defaults to zero", quantity: "five", wantQty: 0, wantIn: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mocks.MockProductRepository)
			repo.On("UpdateStock", mock.Anything, "tenant-1", "prod-1", tt.wantQty, tt.wantIn).Return(nil)
			c := newTestConsumer(repo)

			payload := map[string]interface{}{"tenant_id": "tenant-1", "product_id": "prod-1"}
			if tt.quantity != nil {
				payload["quantity"] = tt.quantity
			}

			c.handleInventoryUpdated(context.Background(), &EventEnvelope{Payload: payload})

			repo.AssertExpectations(t)
		})
	}
}

func TestHandleInventoryUpdated_ExplicitInStockOverridesQuantityDerivation(t *testing.T) {
	tests := []struct {
		name     string
		quantity float64
		inStock  bool
	}{
		{name: "in_stock=false wins over a positive quantity", quantity: 5, inStock: false},
		{name: "in_stock=true wins over a zero quantity", quantity: 0, inStock: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mocks.MockProductRepository)
			repo.On("UpdateStock", mock.Anything, "tenant-1", "prod-1", int(tt.quantity), tt.inStock).Return(nil)
			c := newTestConsumer(repo)

			c.handleInventoryUpdated(context.Background(), &EventEnvelope{Payload: map[string]interface{}{
				"tenant_id": "tenant-1", "product_id": "prod-1", "quantity": tt.quantity, "in_stock": tt.inStock,
			}})

			repo.AssertExpectations(t)
		})
	}
}

func TestHandleInventoryUpdated_FallsBackToDataWhenPayloadAbsent(t *testing.T) {
	repo := new(mocks.MockProductRepository)
	repo.On("UpdateStock", mock.Anything, "tenant-1", "prod-1", 3, true).Return(nil)
	c := newTestConsumer(repo)

	c.handleInventoryUpdated(context.Background(), &EventEnvelope{
		Data: map[string]interface{}{"tenant_id": "tenant-1", "product_id": "prod-1", "quantity": float64(3)},
	})

	repo.AssertExpectations(t)
}

// The consumer is fire-and-forget: nothing downstream can retry a failed
// message, so a repository error must be logged, not panicked on.
func TestHandleInventoryUpdated_RepositoryErrorIsLoggedNotPanicked(t *testing.T) {
	repo := new(mocks.MockProductRepository)
	repo.On("UpdateStock", mock.Anything, "tenant-1", "prod-1", 5, true).Return(errors.New("mongo down"))
	c := newTestConsumer(repo)

	c.handleInventoryUpdated(context.Background(), &EventEnvelope{
		Payload: map[string]interface{}{"tenant_id": "tenant-1", "product_id": "prod-1", "quantity": float64(5)},
	})

	repo.AssertExpectations(t)
}

// === NewEventConsumer / Stop ===

func TestNewEventConsumer_ConfiguresReaderForTheInventoryEventsTopic(t *testing.T) {
	repo := new(mocks.MockProductRepository)
	c := NewEventConsumer([]string{"localhost:9092"}, "product-service-group", repo, logrus.New())
	defer c.Stop()

	cfg := c.reader.Config()
	if cfg.Topic != "inventory-events" {
		t.Errorf("topic = %q, want inventory-events", cfg.Topic)
	}
	if cfg.GroupID != "product-service-group" {
		t.Errorf("group id = %q, want product-service-group", cfg.GroupID)
	}
}

func TestEventConsumer_StopOnAConsumerThatNeverStartedIsSafe(t *testing.T) {
	repo := new(mocks.MockProductRepository)
	c := NewEventConsumer([]string{"localhost:9092"}, "product-service-group", repo, logrus.New())

	// Must not panic or hang even though Start (and therefore consume) was
	// never called.
	c.Stop()
}
