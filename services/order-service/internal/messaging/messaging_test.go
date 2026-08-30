package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/segmentio/kafka-go"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"github.com/yourusername/ecommerce/order-service/internal/projection"
	"go.uber.org/zap"
)

var testTime = time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)

// recordingDispatcher captures the commands an external event translates into.
type recordingDispatcher struct {
	handled []OrderCommand
	err     error
}

func (d *recordingDispatcher) Handle(cmd OrderCommand) error {
	d.handled = append(d.handled, cmd)
	return d.err
}

func newExternalConsumer(dispatcher CommandDispatcher) *ExternalEventConsumer {
	return &ExternalEventConsumer{
		dispatcher: dispatcher,
		logger:     zap.NewNop(),
		stopChan:   make(chan struct{}),
	}
}

func externalMessage(t *testing.T, envelope ExternalEventEnvelope) kafka.Message {
	t.Helper()
	value, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return kafka.Message{Value: value}
}

func TestGetPayloadPrefersPayloadOverData(t *testing.T) {
	tests := []struct {
		name     string
		envelope ExternalEventEnvelope
		wantKey  string
	}{
		{
			name:     "payload wins when both are present",
			envelope: ExternalEventEnvelope{Payload: map[string]interface{}{"from": "payload"}, Data: map[string]interface{}{"from": "data"}},
			wantKey:  "payload",
		},
		{
			name:     "falls back to data",
			envelope: ExternalEventEnvelope{Data: map[string]interface{}{"from": "data"}},
			wantKey:  "data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.envelope.GetPayload()
			if got["from"] != tt.wantKey {
				t.Errorf("GetPayload()[from] = %v, want %v", got["from"], tt.wantKey)
			}
		})
	}

	empty := ExternalEventEnvelope{}
	if empty.GetPayload() != nil {
		t.Error("GetPayload() on an empty envelope should be nil")
	}
}

// Payloads arrive as decoded JSON, so a numeric or missing field must not
// panic or leak a non-string into a command.
func TestGetPayloadString(t *testing.T) {
	payload := map[string]interface{}{
		"order_id": "order-1",
		"quantity": float64(3),
		"nested":   map[string]interface{}{"a": "b"},
	}

	tests := []struct {
		key  string
		want string
	}{
		{"order_id", "order-1"},
		{"quantity", ""},
		{"nested", ""},
		{"missing", ""},
	}

	for _, tt := range tests {
		if got := getPayloadString(payload, tt.key); got != tt.want {
			t.Errorf("getPayloadString(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestExternalCommandsExposeOrderID(t *testing.T) {
	const id = "order-7"
	cmds := []OrderCommand{
		ConfirmOrderCmd{OrderID: id},
		CancelOrderCmd{OrderID: id},
		ShipOrderCmd{OrderID: id},
		DeliverOrderCmd{OrderID: id},
	}
	for _, cmd := range cmds {
		if got := cmd.GetAggregateID(); got != id {
			t.Errorf("%T.GetAggregateID() = %q, want %q", cmd, got, id)
		}
	}
}

// A completed payment confirms the order; a failed one cancels it with the
// upstream reason preserved in the cancellation text.
func TestHandlePaymentEvent(t *testing.T) {
	tests := []struct {
		name    string
		event   ExternalEventEnvelope
		wantCmd OrderCommand
	}{
		{
			name: "PaymentCompleted confirms",
			event: ExternalEventEnvelope{
				EventType: "PaymentCompleted",
				Payload:   map[string]interface{}{"order_id": "order-1"},
			},
			wantCmd: ConfirmOrderCmd{OrderID: "order-1", ConfirmedBy: "payment-service"},
		},
		{
			name: "PaymentFailed cancels with the reason",
			event: ExternalEventEnvelope{
				EventType: "PaymentFailed",
				Payload:   map[string]interface{}{"order_id": "order-1", "reason": "card declined"},
			},
			wantCmd: CancelOrderCmd{OrderID: "order-1", Reason: "Payment failed: card declined", CancelledBy: "payment-service"},
		},
		{
			name: "PaymentFailed without a reason gets a default",
			event: ExternalEventEnvelope{
				EventType: "PaymentFailed",
				Payload:   map[string]interface{}{"order_id": "order-1"},
			},
			wantCmd: CancelOrderCmd{OrderID: "order-1", Reason: "Payment failed: Payment failed", CancelledBy: "payment-service"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := &recordingDispatcher{}
			c := newExternalConsumer(dispatcher)

			c.handleMessage(context.Background(), "payment-events", externalMessage(t, tt.event))

			if len(dispatcher.handled) != 1 {
				t.Fatalf("dispatched %d commands, want 1", len(dispatcher.handled))
			}
			if dispatcher.handled[0] != tt.wantCmd {
				t.Errorf("dispatched %#v, want %#v", dispatcher.handled[0], tt.wantCmd)
			}
		})
	}
}

func TestHandlePaymentEventIgnoresIncompleteEvents(t *testing.T) {
	tests := []struct {
		name  string
		event ExternalEventEnvelope
	}{
		{name: "no payload", event: ExternalEventEnvelope{EventType: "PaymentCompleted"}},
		{name: "no order_id", event: ExternalEventEnvelope{EventType: "PaymentCompleted", Payload: map[string]interface{}{"amount": 100.0}}},
		{name: "unrecognised type", event: ExternalEventEnvelope{EventType: "PaymentPending", Payload: map[string]interface{}{"order_id": "order-1"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := &recordingDispatcher{}
			c := newExternalConsumer(dispatcher)

			c.handleMessage(context.Background(), "payment-events", externalMessage(t, tt.event))

			if len(dispatcher.handled) != 0 {
				t.Errorf("dispatched %#v, want nothing", dispatcher.handled)
			}
		})
	}
}

// A dispatcher failure is logged, not propagated — the consumer must keep
// draining the topic rather than wedging on one bad order.
func TestHandlePaymentEventSurvivesDispatcherFailure(t *testing.T) {
	dispatcher := &recordingDispatcher{err: errors.New("aggregate rejected")}
	c := newExternalConsumer(dispatcher)

	c.handleMessage(context.Background(), "payment-events", externalMessage(t, ExternalEventEnvelope{
		EventType: "PaymentCompleted",
		Payload:   map[string]interface{}{"order_id": "order-1"},
	}))

	if len(dispatcher.handled) != 1 {
		t.Errorf("dispatched %d commands, want the attempt to still happen", len(dispatcher.handled))
	}
}

func TestHandleInventoryEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    ExternalEventEnvelope
		wantCmds []OrderCommand
	}{
		{
			name: "StockReservationFailed cancels with the reason",
			event: ExternalEventEnvelope{
				EventType: "StockReservationFailed",
				Payload:   map[string]interface{}{"order_id": "order-1", "reason": "only 2 left"},
			},
			wantCmds: []OrderCommand{CancelOrderCmd{OrderID: "order-1", Reason: "Stock reservation failed: only 2 left", CancelledBy: "inventory-service"}},
		},
		{
			name: "StockReservationFailed without a reason gets a default",
			event: ExternalEventEnvelope{
				EventType: "StockReservationFailed",
				Payload:   map[string]interface{}{"order_id": "order-1"},
			},
			wantCmds: []OrderCommand{CancelOrderCmd{OrderID: "order-1", Reason: "Stock reservation failed: Insufficient stock", CancelledBy: "inventory-service"}},
		},
		{
			// The saga drives the reservation happy path; this consumer only logs.
			name: "StockReserved dispatches nothing",
			event: ExternalEventEnvelope{
				EventType: "StockReserved",
				Payload:   map[string]interface{}{"order_id": "order-1"},
			},
			wantCmds: nil,
		},
		{
			name: "StockLevelLow is an alert, not an order change",
			event: ExternalEventEnvelope{
				EventType: "StockLevelLow",
				Payload:   map[string]interface{}{"product_id": "prod-1"},
			},
			wantCmds: nil,
		},
		{
			name: "StockReservationFailed without an order_id is dropped",
			event: ExternalEventEnvelope{
				EventType: "StockReservationFailed",
				Payload:   map[string]interface{}{"reason": "no stock"},
			},
			wantCmds: nil,
		},
		{
			name:     "no payload",
			event:    ExternalEventEnvelope{EventType: "StockReserved"},
			wantCmds: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := &recordingDispatcher{}
			c := newExternalConsumer(dispatcher)

			c.handleMessage(context.Background(), "inventory-events", externalMessage(t, tt.event))

			if len(dispatcher.handled) != len(tt.wantCmds) {
				t.Fatalf("dispatched %#v, want %#v", dispatcher.handled, tt.wantCmds)
			}
			for i, want := range tt.wantCmds {
				if dispatcher.handled[i] != want {
					t.Errorf("dispatched[%d] = %#v, want %#v", i, dispatcher.handled[i], want)
				}
			}
		})
	}
}

func TestHandleShippingEvent(t *testing.T) {
	tests := []struct {
		name    string
		event   ExternalEventEnvelope
		wantCmd OrderCommand
	}{
		{
			name: "ShipmentCreated ships with tracking",
			event: ExternalEventEnvelope{
				EventType: "ShipmentCreated",
				Payload:   map[string]interface{}{"order_id": "order-1", "tracking_number": "TRK-9", "carrier": "Pathao"},
			},
			wantCmd: ShipOrderCmd{OrderID: "order-1", TrackingNumber: "TRK-9", Carrier: "Pathao", ShippedBy: "shipping-service"},
		},
		{
			// The shipping service emits either name for the same thing.
			name: "OrderShipped is treated the same as ShipmentCreated",
			event: ExternalEventEnvelope{
				EventType: "OrderShipped",
				Payload:   map[string]interface{}{"order_id": "order-1", "tracking_number": "TRK-9", "carrier": "Pathao"},
			},
			wantCmd: ShipOrderCmd{OrderID: "order-1", TrackingNumber: "TRK-9", Carrier: "Pathao", ShippedBy: "shipping-service"},
		},
		{
			name: "ShipmentDelivered records the recipient",
			event: ExternalEventEnvelope{
				EventType: "ShipmentDelivered",
				Payload:   map[string]interface{}{"order_id": "order-1", "received_by": "front desk"},
			},
			wantCmd: DeliverOrderCmd{OrderID: "order-1", ReceivedBy: "front desk"},
		},
		{
			name: "delivery without a recipient defaults to customer",
			event: ExternalEventEnvelope{
				EventType: "OrderDelivered",
				Payload:   map[string]interface{}{"order_id": "order-1"},
			},
			wantCmd: DeliverOrderCmd{OrderID: "order-1", ReceivedBy: "customer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := &recordingDispatcher{}
			c := newExternalConsumer(dispatcher)

			c.handleMessage(context.Background(), "shipping-events", externalMessage(t, tt.event))

			if len(dispatcher.handled) != 1 {
				t.Fatalf("dispatched %d commands, want 1", len(dispatcher.handled))
			}
			if dispatcher.handled[0] != tt.wantCmd {
				t.Errorf("dispatched %#v, want %#v", dispatcher.handled[0], tt.wantCmd)
			}
		})
	}
}

func TestHandleShippingEventIgnoresIncompleteEvents(t *testing.T) {
	tests := []struct {
		name  string
		event ExternalEventEnvelope
	}{
		{name: "no payload", event: ExternalEventEnvelope{EventType: "ShipmentCreated"}},
		{name: "no order_id", event: ExternalEventEnvelope{EventType: "ShipmentCreated", Payload: map[string]interface{}{"carrier": "Pathao"}}},
		{name: "unrecognised type", event: ExternalEventEnvelope{EventType: "ShipmentDelayed", Payload: map[string]interface{}{"order_id": "order-1"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := &recordingDispatcher{}
			c := newExternalConsumer(dispatcher)

			c.handleMessage(context.Background(), "shipping-events", externalMessage(t, tt.event))

			if len(dispatcher.handled) != 0 {
				t.Errorf("dispatched %#v, want nothing", dispatcher.handled)
			}
		})
	}
}

func TestHandleMessageIgnoresMalformedAndUnknownTopics(t *testing.T) {
	t.Run("malformed JSON", func(t *testing.T) {
		dispatcher := &recordingDispatcher{}
		c := newExternalConsumer(dispatcher)

		c.handleMessage(context.Background(), "payment-events", kafka.Message{Value: []byte("{not json")})

		if len(dispatcher.handled) != 0 {
			t.Errorf("dispatched %#v on malformed input, want nothing", dispatcher.handled)
		}
	})

	t.Run("unrouted topic", func(t *testing.T) {
		dispatcher := &recordingDispatcher{}
		c := newExternalConsumer(dispatcher)

		c.handleMessage(context.Background(), "promotion-events", externalMessage(t, ExternalEventEnvelope{
			EventType: "PaymentCompleted",
			Payload:   map[string]interface{}{"order_id": "order-1"},
		}))

		if len(dispatcher.handled) != 0 {
			t.Errorf("dispatched %#v for an unrouted topic, want nothing", dispatcher.handled)
		}
	})
}

func TestNewExternalEventConsumerSubscribesToEveryUpstreamTopic(t *testing.T) {
	c := NewExternalEventConsumer([]string{"localhost:9092"}, "order-service", &recordingDispatcher{}, zap.NewNop())
	defer func() { _ = c.Stop() }()

	if len(c.readers) != 3 {
		t.Fatalf("got %d readers, want one each for payment, inventory and shipping", len(c.readers))
	}

	got := make(map[string]bool)
	for _, r := range c.readers {
		got[r.Config().Topic] = true
	}
	for _, topic := range []string{"payment-events", "inventory-events", "shipping-events"} {
		if !got[topic] {
			t.Errorf("no reader subscribed to %s", topic)
		}
	}
}

// newProjectionConsumer builds a consumer whose projection is backed by a mock
// driver, so processMessage can be exercised end to end without Postgres.
func newProjectionConsumer(t *testing.T) (*KafkaEventConsumer, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS order_read_model").WillReturnResult(sqlmock.NewResult(0, 0))

	proj, err := projection.NewOrderProjection(db, zap.NewNop())
	if err != nil {
		t.Fatalf("NewOrderProjection: %v", err)
	}

	c := &KafkaEventConsumer{
		projection: proj,
		logger:     zap.NewNop(),
		stopChan:   make(chan struct{}),
	}

	return c, mock, func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
		_ = db.Close()
	}
}

func TestDeserializeEventCoversEveryPublishedType(t *testing.T) {
	c := &KafkaEventConsumer{logger: zap.NewNop()}

	tests := []struct {
		eventType events.EventType
		want      interface{}
	}{
		{events.OrderCreatedEvent, events.OrderCreated{}},
		{events.OrderItemAddedEvent, events.OrderItemAdded{}},
		{events.OrderItemRemovedEvent, events.OrderItemRemoved{}},
		{events.OrderConfirmedEvent, events.OrderConfirmed{}},
		{events.OrderCancelledEvent, events.OrderCancelled{}},
		{events.OrderShippedEvent, events.OrderShipped{}},
		{events.OrderDeliveredEvent, events.OrderDelivered{}},
		{events.PaymentProcessedEvent, events.PaymentProcessed{}},
		{events.PaymentFailedEvent, events.PaymentFailed{}},
		{events.InventoryReservedEvent, events.InventoryReserved{}},
		{events.InventoryReleasedEvent, events.InventoryReleased{}},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			envelope := EventEnvelope{
				EventType:   string(tt.eventType),
				AggregateID: "order-1",
				Version:     3,
				Data: map[string]interface{}{
					"aggregate_id": "order-1",
					"event_type":   string(tt.eventType),
					"version":      3,
				},
			}

			event, err := c.deserializeEvent(envelope)
			if err != nil {
				t.Fatalf("deserializeEvent(%s) = %v", tt.eventType, err)
			}
			if got := event.GetAggregateID(); got != "order-1" {
				t.Errorf("aggregate id = %q, want order-1", got)
			}
			if got := event.GetVersion(); got != 3 {
				t.Errorf("version = %d, want 3", got)
			}
			// The concrete type matters: the projection switches on it.
			gotType := fmt.Sprintf("%T", event)
			wantType := fmt.Sprintf("%T", tt.want)
			if gotType != wantType {
				t.Errorf("concrete type = %s, want %s", gotType, wantType)
			}
		})
	}
}

func TestDeserializeEventRejectsUnknownType(t *testing.T) {
	c := &KafkaEventConsumer{logger: zap.NewNop()}

	_, err := c.deserializeEvent(EventEnvelope{EventType: "OrderTeleported"})

	if err == nil || err.Error() != "unknown event type: OrderTeleported" {
		t.Fatalf("deserializeEvent = %v, want an unknown-type error", err)
	}
}

func TestDeserializeEventRejectsMistypedFields(t *testing.T) {
	c := &KafkaEventConsumer{logger: zap.NewNop()}

	// version is an int on the event but a string on the wire here.
	_, err := c.deserializeEvent(EventEnvelope{
		EventType: string(events.OrderConfirmedEvent),
		Data:      map[string]interface{}{"version": "three"},
	})

	if err == nil {
		t.Fatal("deserializeEvent accepted a mistyped field, want an error")
	}
}

// The publisher's envelope and the consumer's decoder are two halves of the
// same wire contract; this drives one through the other.
func TestPublishedEnvelopeRoundTripsThroughTheConsumer(t *testing.T) {
	c, mock, done := newProjectionConsumer(t)
	defer done()

	original := events.OrderShipped{
		BaseEvent: events.BaseEvent{
			ID:          "evt-1",
			AggregateID: "order-1",
			EventType:   events.OrderShippedEvent,
			Timestamp:   testTime,
			Version:     6,
		},
		TrackingNumber: "TRK-9",
		Carrier:        "Pathao",
		ShippedAt:      testTime,
		ShippedBy:      "staff-1",
	}

	// Built exactly as KafkaEventPublisher.PublishBatch builds it.
	value, err := json.Marshal(EventEnvelope{
		EventID:       original.GetID(),
		EventType:     string(original.GetEventType()),
		AggregateID:   original.GetAggregateID(),
		AggregateType: "Order",
		Version:       original.GetVersion(),
		Timestamp:     original.GetTimestamp(),
		Data:          original.GetData(),
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	mock.ExpectExec("UPDATE order_read_model").
		WithArgs("shipped", "TRK-9", "Pathao", testTime, 6, "order-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := c.processMessage(context.Background(), kafka.Message{Value: value}); err != nil {
		t.Fatalf("processMessage = %v", err)
	}
}

func TestProcessMessageRejectsMalformedEnvelope(t *testing.T) {
	c, _, done := newProjectionConsumer(t)
	defer done()

	err := c.processMessage(context.Background(), kafka.Message{Value: []byte("{not json")})

	if err == nil {
		t.Fatal("processMessage accepted malformed JSON, want an error")
	}
}

func TestProcessMessageRejectsUnknownEventType(t *testing.T) {
	c, _, done := newProjectionConsumer(t)
	defer done()

	value, err := json.Marshal(EventEnvelope{EventType: "OrderTeleported", AggregateID: "order-1"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := c.processMessage(context.Background(), kafka.Message{Value: value}); err == nil {
		t.Fatal("processMessage accepted an unknown event type, want an error")
	}
}

// A projection failure must be reported so the consume loop can decide whether
// to retry rather than silently committing the offset.
func TestProcessMessageSurfacesProjectionFailure(t *testing.T) {
	c, mock, done := newProjectionConsumer(t)
	defer done()

	value, err := json.Marshal(EventEnvelope{
		EventType:   string(events.OrderConfirmedEvent),
		AggregateID: "order-1",
		Data: map[string]interface{}{
			"aggregate_id": "order-1",
			"version":      4,
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	mock.ExpectExec("UPDATE order_read_model").WillReturnError(errors.New("read model down"))

	if err := c.processMessage(context.Background(), kafka.Message{Value: value}); err == nil {
		t.Fatal("processMessage swallowed a projection failure, want an error")
	}
}

func TestNewKafkaEventConsumerConfiguresReader(t *testing.T) {
	c := NewKafkaEventConsumer([]string{"localhost:9092"}, "order-events", "order-projection", nil, zap.NewNop())
	defer func() { _ = c.Stop() }()

	cfg := c.reader.Config()
	if cfg.Topic != "order-events" {
		t.Errorf("topic = %q, want order-events", cfg.Topic)
	}
	if cfg.GroupID != "order-projection" {
		t.Errorf("group id = %q, want order-projection", cfg.GroupID)
	}
	if cfg.StartOffset != kafka.FirstOffset {
		t.Errorf("start offset = %d, want FirstOffset — a rebuilt projection must replay from the beginning", cfg.StartOffset)
	}
}

func TestPublishBatchWithNoEventsIsANoop(t *testing.T) {
	// A nil writer proves nothing touches Kafka on the empty path.
	p := &KafkaEventPublisher{logger: zap.NewNop(), topic: "order-events"}

	if err := p.PublishBatch(context.Background(), nil); err != nil {
		t.Errorf("PublishBatch(nil) = %v, want nil", err)
	}
	if err := p.PublishBatch(context.Background(), []events.Event{}); err != nil {
		t.Errorf("PublishBatch(empty) = %v, want nil", err)
	}
}

// unserialisableEvent returns data json.Marshal cannot encode, exercising the
// publisher's marshal-failure branch without a broker.
type unserialisableEvent struct{ events.BaseEvent }

func (e unserialisableEvent) GetData() interface{} { return make(chan int) }

func TestPublishBatchFailsBeforeWritingWhenAnEventCannotMarshal(t *testing.T) {
	p := &KafkaEventPublisher{logger: zap.NewNop(), topic: "order-events"}

	err := p.PublishBatch(context.Background(), []events.Event{
		unserialisableEvent{BaseEvent: events.BaseEvent{AggregateID: "order-1", EventType: events.OrderCreatedEvent}},
	})

	if err == nil {
		t.Fatal("PublishBatch accepted an unmarshalable event, want an error")
	}
}

// The publish path is async on purpose: the command handler must not block on
// broker round-trips, because Postgres is the source of truth.
func TestNewKafkaEventPublisherIsAsyncAndKeyedForOrdering(t *testing.T) {
	p := NewKafkaEventPublisher([]string{"localhost:9092"}, "order-events", zap.NewNop())

	if !p.writer.Async {
		t.Error("writer is synchronous — the command path would block on Kafka")
	}
	if p.writer.Topic != "order-events" || p.topic != "order-events" {
		t.Errorf("topic = %q/%q, want order-events", p.writer.Topic, p.topic)
	}
	if p.writer.RequiredAcks != kafka.RequireOne {
		t.Errorf("RequiredAcks = %v, want RequireOne", p.writer.RequiredAcks)
	}
	if p.writer.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", p.writer.MaxAttempts)
	}
	if p.writer.Completion == nil {
		t.Error("no Completion callback — async publish failures would be invisible")
	}
}

func TestPublishReceiptRequestedRequiresTenantAndOrder(t *testing.T) {
	tests := []struct {
		name    string
		payload ReceiptRequestedPayload
	}{
		{name: "no tenant", payload: ReceiptRequestedPayload{OrderID: "order-1"}},
		{name: "no order", payload: ReceiptRequestedPayload{TenantID: "tenant_saajan"}},
		{name: "neither", payload: ReceiptRequestedPayload{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A nil writer proves validation rejects before any Kafka call.
			p := &KafkaNotificationPublisher{logger: zap.NewNop(), topic: "order-events"}

			err := p.PublishReceiptRequested(context.Background(), tt.payload)

			if err == nil || err.Error() != "tenant_id and order_id are required" {
				t.Fatalf("PublishReceiptRequested = %v, want the validation error", err)
			}
		})
	}
}

func TestNewKafkaNotificationPublisherConfiguresWriter(t *testing.T) {
	p := NewKafkaNotificationPublisher([]string{"localhost:9092"}, "order-events", zap.NewNop())

	if p.writer.Topic != "order-events" || p.topic != "order-events" {
		t.Errorf("topic = %q/%q, want order-events", p.writer.Topic, p.topic)
	}
	// Receipts are user-visible and low volume, so they go out one at a time
	// rather than waiting for a batch to fill.
	if p.writer.BatchSize != 1 {
		t.Errorf("BatchSize = %d, want 1", p.writer.BatchSize)
	}
	if p.writer.Async {
		t.Error("receipt publishing is async — the caller could not report failure")
	}
}

func TestReceiptEnvelopeCarriesTheOrderAsAggregate(t *testing.T) {
	payload := ReceiptRequestedPayload{
		TenantID:      "tenant_saajan",
		OrderID:       "order-1",
		CustomerEmail: "buyer@example.test",
		Currency:      "BDT",
		Total:         1000,
		Items: []ReceiptRequestedItem{
			{Name: "Shirt", SKU: "SKU-1", Quantity: 2, UnitPrice: 500, TotalPrice: 1000},
		},
	}

	// Mirrors the envelope PublishReceiptRequested builds, verifying the shape
	// notification-service decodes.
	value, err := json.Marshal(EventEnvelope{
		EventType:     "ReceiptRequested",
		AggregateID:   payload.OrderID,
		AggregateType: "Order",
		Version:       1,
		Timestamp:     testTime,
		Data:          payload,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded struct {
		EventType   string                  `json:"event_type"`
		AggregateID string                  `json:"aggregate_id"`
		Data        ReceiptRequestedPayload `json:"data"`
	}
	if err := json.Unmarshal(value, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.EventType != "ReceiptRequested" || decoded.AggregateID != "order-1" {
		t.Errorf("envelope = %+v, want a ReceiptRequested keyed to order-1", decoded)
	}
	if decoded.Data.TenantID != "tenant_saajan" {
		t.Errorf("tenant = %q, want tenant_saajan — the consumer scopes the email by it", decoded.Data.TenantID)
	}
	if len(decoded.Data.Items) != 1 || decoded.Data.Items[0].SKU != "SKU-1" {
		t.Errorf("items = %+v, want the single line item to survive the round trip", decoded.Data.Items)
	}
}
