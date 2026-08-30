package commands

import (
	"errors"
	"testing"

	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"go.uber.org/zap"
)

// fakeEventStore is an in-memory eventstore.EventStore whose Save/GetEvents can
// be made to fail, so the handler's error wrapping is exercised.
type fakeEventStore struct {
	events              map[string][]events.Event
	saveErr             error
	getErr              error
	saveCalls           int
	lastExpectedVersion int
}

func newFakeEventStore() *fakeEventStore {
	return &fakeEventStore{events: make(map[string][]events.Event)}
}

func (s *fakeEventStore) Save(aggregateID string, evs []events.Event, expectedVersion int) error {
	s.saveCalls++
	s.lastExpectedVersion = expectedVersion
	if s.saveErr != nil {
		return s.saveErr
	}
	s.events[aggregateID] = append(s.events[aggregateID], evs...)
	return nil
}

func (s *fakeEventStore) GetEvents(aggregateID string) ([]events.Event, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.events[aggregateID], nil
}

func (s *fakeEventStore) GetEventsByType(eventType events.EventType, limit int) ([]events.Event, error) {
	return nil, nil
}

func (s *fakeEventStore) GetAllEvents(offset, limit int) ([]events.Event, error) {
	return nil, nil
}

// types returns the event types persisted for an aggregate, in order.
func (s *fakeEventStore) types(aggregateID string) []events.EventType {
	out := make([]events.EventType, 0, len(s.events[aggregateID]))
	for _, e := range s.events[aggregateID] {
		out = append(out, e.GetEventType())
	}
	return out
}

// recordingProjector captures everything handed to the synchronous projector.
type recordingProjector struct {
	projected []events.EventType
	err       error
}

func (p *recordingProjector) Project(event events.Event) error {
	p.projected = append(p.projected, event.GetEventType())
	return p.err
}

type unknownCommand struct{}

func (unknownCommand) GetAggregateID() string { return "order-1" }

func testAddress() events.Address {
	return events.Address{Street: "1 St", City: "Dhaka", State: "DH", PostalCode: "1200", Country: "BD"}
}

// newHandler returns a handler wired to a fresh store and projector.
func newHandler(t *testing.T) (*CommandHandler, *fakeEventStore, *recordingProjector) {
	t.Helper()
	store := newFakeEventStore()
	projector := &recordingProjector{}
	h := NewCommandHandler(store, zap.NewNop())
	h.SetProjector(projector)
	return h, store, projector
}

// seedPending creates a pending order carrying one item.
func seedPending(t *testing.T, h *CommandHandler, orderID string) {
	t.Helper()
	if err := h.Handle(CreateOrderCommand{
		OrderID:         orderID,
		TenantID:        "tenant_saajan",
		CustomerID:      "cust-1",
		ShippingAddress: testAddress(),
		BillingAddress:  testAddress(),
	}); err != nil {
		t.Fatalf("seed create: %v", err)
	}
	if err := h.Handle(AddOrderItemCommand{
		OrderID:   orderID,
		ProductID: "prod-1",
		SKU:       "SKU-1",
		Name:      "Shirt",
		Quantity:  2,
		UnitPrice: 500,
	}); err != nil {
		t.Fatalf("seed add item: %v", err)
	}
}

func seedConfirmed(t *testing.T, h *CommandHandler, orderID string) {
	t.Helper()
	seedPending(t, h, orderID)
	if err := h.Handle(ConfirmOrderCommand{OrderID: orderID, ConfirmedBy: "staff-1"}); err != nil {
		t.Fatalf("seed confirm: %v", err)
	}
}

func seedShipped(t *testing.T, h *CommandHandler, orderID string) {
	t.Helper()
	seedConfirmed(t, h, orderID)
	if err := h.Handle(ShipOrderCommand{OrderID: orderID, TrackingNumber: "TRK-1", Carrier: "Pathao", ShippedBy: "staff-1"}); err != nil {
		t.Fatalf("seed ship: %v", err)
	}
}

func TestGetAggregateIDReturnsOrderID(t *testing.T) {
	const id = "order-42"
	cmds := []Command{
		CreateOrderCommand{OrderID: id},
		AddOrderItemCommand{OrderID: id},
		RemoveOrderItemCommand{OrderID: id},
		ConfirmOrderCommand{OrderID: id},
		CancelOrderCommand{OrderID: id},
		ShipOrderCommand{OrderID: id},
		DeliverOrderCommand{OrderID: id},
		RecordInventoryReservationCommand{OrderID: id},
		RecordInventoryReleaseCommand{OrderID: id},
		RecordPaymentCommand{OrderID: id},
		RecordPaymentFailureCommand{OrderID: id},
	}
	for _, cmd := range cmds {
		if got := cmd.GetAggregateID(); got != id {
			t.Errorf("%T.GetAggregateID() = %q, want %q", cmd, got, id)
		}
	}
	if len(cmds) != 11 {
		t.Errorf("expected 11 command types, covered %d", len(cmds))
	}
}

func TestHandleUnknownCommandType(t *testing.T) {
	h, store, _ := newHandler(t)

	err := h.Handle(unknownCommand{})

	if err == nil || err.Error() != "unknown command type" {
		t.Fatalf("Handle(unknown) = %v, want unknown command type", err)
	}
	if store.saveCalls != 0 {
		t.Errorf("unknown command persisted %d events, want 0", store.saveCalls)
	}
}

func TestHandleCreateOrderPersistsAndProjects(t *testing.T) {
	h, store, projector := newHandler(t)

	err := h.Handle(CreateOrderCommand{
		OrderID:         "order-1",
		TenantID:        "tenant_saajan",
		CustomerID:      "cust-1",
		ShippingAddress: testAddress(),
		BillingAddress:  testAddress(),
	})
	if err != nil {
		t.Fatalf("Handle(CreateOrderCommand) = %v", err)
	}

	if got := store.types("order-1"); len(got) != 1 || got[0] != events.OrderCreatedEvent {
		t.Errorf("persisted events = %v, want [OrderCreated]", got)
	}
	if len(projector.projected) != 1 || projector.projected[0] != events.OrderCreatedEvent {
		t.Errorf("projected events = %v, want [OrderCreated]", projector.projected)
	}
	// A brand-new aggregate must assert "no prior stream" via expectedVersion -1.
	if store.lastExpectedVersion != -1 {
		t.Errorf("expectedVersion = %d, want -1 for a new aggregate", store.lastExpectedVersion)
	}
}

func TestHandleCreateOrderWrapsSaveError(t *testing.T) {
	h, store, projector := newHandler(t)
	store.saveErr = errors.New("db down")

	err := h.Handle(CreateOrderCommand{OrderID: "order-1", TenantID: "t", CustomerID: "c"})

	if err == nil {
		t.Fatal("expected an error when the event store rejects the save")
	}
	if !errors.Is(err, store.saveErr) {
		t.Errorf("error %v does not wrap the store error", err)
	}
	if len(projector.projected) != 0 {
		t.Errorf("projected %v after a failed save, want nothing", projector.projected)
	}
}

func TestHandleCommandsOnMissingOrder(t *testing.T) {
	cmds := []Command{
		AddOrderItemCommand{OrderID: "ghost", ProductID: "p", Quantity: 1, UnitPrice: 1},
		RemoveOrderItemCommand{OrderID: "ghost", ItemID: "p"},
		ConfirmOrderCommand{OrderID: "ghost"},
		CancelOrderCommand{OrderID: "ghost"},
		ShipOrderCommand{OrderID: "ghost"},
		DeliverOrderCommand{OrderID: "ghost"},
		RecordInventoryReservationCommand{OrderID: "ghost"},
		RecordInventoryReleaseCommand{OrderID: "ghost"},
		RecordPaymentCommand{OrderID: "ghost"},
		RecordPaymentFailureCommand{OrderID: "ghost"},
	}

	for _, cmd := range cmds {
		h, _, _ := newHandler(t)
		err := h.Handle(cmd)
		if err == nil || err.Error() != "order not found" {
			t.Errorf("%T on a missing order = %v, want order not found", cmd, err)
		}
	}
}

func TestHandleWrapsEventStoreReadError(t *testing.T) {
	h, store, _ := newHandler(t)
	store.getErr = errors.New("read timeout")

	err := h.Handle(ConfirmOrderCommand{OrderID: "order-1"})

	if err == nil || !errors.Is(err, store.getErr) {
		t.Fatalf("Handle = %v, want an error wrapping %v", err, store.getErr)
	}
}

func TestHandleAddOrderItem(t *testing.T) {
	h, store, projector := newHandler(t)
	if err := h.Handle(CreateOrderCommand{OrderID: "order-1", TenantID: "t", CustomerID: "c"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	projector.projected = nil

	err := h.Handle(AddOrderItemCommand{
		OrderID:   "order-1",
		ProductID: "prod-1",
		VariantID: "red",
		SKU:       "SKU-1",
		Name:      "Shirt",
		Quantity:  3,
		UnitPrice: 250,
	})
	if err != nil {
		t.Fatalf("Handle(AddOrderItemCommand) = %v", err)
	}

	if got := store.types("order-1"); len(got) != 2 || got[1] != events.OrderItemAddedEvent {
		t.Errorf("persisted events = %v, want OrderItemAdded appended", got)
	}
	if len(projector.projected) != 1 || projector.projected[0] != events.OrderItemAddedEvent {
		t.Errorf("projected = %v, want [OrderItemAdded]", projector.projected)
	}
}

func TestHandleAddOrderItemPropagatesAggregateRejection(t *testing.T) {
	h, store, _ := newHandler(t)
	if err := h.Handle(CreateOrderCommand{OrderID: "order-1", TenantID: "t", CustomerID: "c"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	before := store.saveCalls

	// Quantity 0 is rejected by the aggregate, so nothing may be persisted.
	err := h.Handle(AddOrderItemCommand{OrderID: "order-1", ProductID: "prod-1", Quantity: 0, UnitPrice: 250})

	if err == nil || err.Error() != "quantity must be greater than 0" {
		t.Fatalf("Handle = %v, want the aggregate's quantity error", err)
	}
	if store.saveCalls != before {
		t.Errorf("a rejected command still called Save (%d -> %d)", before, store.saveCalls)
	}
}

func TestHandleRemoveOrderItem(t *testing.T) {
	h, store, projector := newHandler(t)
	seedPending(t, h, "order-1")
	projector.projected = nil

	if err := h.Handle(RemoveOrderItemCommand{OrderID: "order-1", ItemID: "prod-1"}); err != nil {
		t.Fatalf("Handle(RemoveOrderItemCommand) = %v", err)
	}

	got := store.types("order-1")
	if got[len(got)-1] != events.OrderItemRemovedEvent {
		t.Errorf("last persisted event = %v, want OrderItemRemoved", got[len(got)-1])
	}
	if len(projector.projected) != 1 || projector.projected[0] != events.OrderItemRemovedEvent {
		t.Errorf("projected = %v, want [OrderItemRemoved]", projector.projected)
	}
}

func TestHandleRemoveUnknownItem(t *testing.T) {
	h, _, _ := newHandler(t)
	seedPending(t, h, "order-1")

	err := h.Handle(RemoveOrderItemCommand{OrderID: "order-1", ItemID: "not-there"})

	if err == nil || err.Error() != "item not found in order" {
		t.Fatalf("Handle = %v, want item not found in order", err)
	}
}

func TestHandleConfirmOrder(t *testing.T) {
	h, store, projector := newHandler(t)
	seedPending(t, h, "order-1")
	projector.projected = nil

	if err := h.Handle(ConfirmOrderCommand{OrderID: "order-1", ConfirmedBy: "staff-1"}); err != nil {
		t.Fatalf("Handle(ConfirmOrderCommand) = %v", err)
	}

	got := store.types("order-1")
	if got[len(got)-1] != events.OrderConfirmedEvent {
		t.Errorf("last persisted event = %v, want OrderConfirmed", got[len(got)-1])
	}
	if len(projector.projected) != 1 || projector.projected[0] != events.OrderConfirmedEvent {
		t.Errorf("projected = %v, want [OrderConfirmed]", projector.projected)
	}
}

func TestHandleConfirmEmptyOrderIsRejected(t *testing.T) {
	h, _, _ := newHandler(t)
	if err := h.Handle(CreateOrderCommand{OrderID: "order-1", TenantID: "t", CustomerID: "c"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	err := h.Handle(ConfirmOrderCommand{OrderID: "order-1", ConfirmedBy: "staff-1"})

	if err == nil || err.Error() != "cannot confirm order with no items" {
		t.Fatalf("Handle = %v, want the empty-order rejection", err)
	}
}

func TestHandleCancelOrder(t *testing.T) {
	h, store, projector := newHandler(t)
	seedPending(t, h, "order-1")
	projector.projected = nil

	if err := h.Handle(CancelOrderCommand{OrderID: "order-1", Reason: "changed mind", CancelledBy: "cust-1"}); err != nil {
		t.Fatalf("Handle(CancelOrderCommand) = %v", err)
	}

	got := store.types("order-1")
	if got[len(got)-1] != events.OrderCancelledEvent {
		t.Errorf("last persisted event = %v, want OrderCancelled", got[len(got)-1])
	}
	if len(projector.projected) != 1 || projector.projected[0] != events.OrderCancelledEvent {
		t.Errorf("projected = %v, want [OrderCancelled]", projector.projected)
	}
}

// A paid order emits OrderRefunded alongside OrderCancelled; both must be
// persisted and both must reach the read model.
func TestHandleCancelPaidOrderAlsoRecordsRefund(t *testing.T) {
	h, store, projector := newHandler(t)
	seedPending(t, h, "order-1")
	if err := h.Handle(RecordPaymentCommand{
		OrderID:       "order-1",
		PaymentID:     "pay-1",
		PaymentMethod: "card",
		TransactionID: "txn-1",
		Amount:        1000,
	}); err != nil {
		t.Fatalf("record payment: %v", err)
	}
	projector.projected = nil

	if err := h.Handle(CancelOrderCommand{OrderID: "order-1", Reason: "fraud", CancelledBy: "staff-1"}); err != nil {
		t.Fatalf("Handle(CancelOrderCommand) = %v", err)
	}

	got := store.types("order-1")
	if len(got) < 2 || got[len(got)-2] != events.OrderCancelledEvent || got[len(got)-1] != events.OrderRefundedEvent {
		t.Errorf("tail of persisted events = %v, want [... OrderCancelled OrderRefunded]", got)
	}
	if len(projector.projected) != 2 {
		t.Errorf("projected = %v, want both OrderCancelled and OrderRefunded", projector.projected)
	}
}

func TestHandleCancelAlreadyCancelledOrder(t *testing.T) {
	h, _, _ := newHandler(t)
	seedPending(t, h, "order-1")
	if err := h.Handle(CancelOrderCommand{OrderID: "order-1", Reason: "first", CancelledBy: "c"}); err != nil {
		t.Fatalf("first cancel: %v", err)
	}

	err := h.Handle(CancelOrderCommand{OrderID: "order-1", Reason: "second", CancelledBy: "c"})

	if err == nil || err.Error() != "order already cancelled" {
		t.Fatalf("Handle = %v, want order already cancelled", err)
	}
}

// Regression: every handler must feed the synchronous projector, otherwise a
// read immediately after the write still shows the pre-command state. Ship was
// the one handler that skipped it.
func TestHandleShipOrderProjectsToReadModel(t *testing.T) {
	h, store, projector := newHandler(t)
	seedConfirmed(t, h, "order-1")
	projector.projected = nil

	if err := h.Handle(ShipOrderCommand{
		OrderID:        "order-1",
		TrackingNumber: "TRK-9",
		Carrier:        "Pathao",
		ShippedBy:      "staff-1",
	}); err != nil {
		t.Fatalf("Handle(ShipOrderCommand) = %v", err)
	}

	got := store.types("order-1")
	if got[len(got)-1] != events.OrderShippedEvent {
		t.Errorf("last persisted event = %v, want OrderShipped", got[len(got)-1])
	}
	if len(projector.projected) != 1 || projector.projected[0] != events.OrderShippedEvent {
		t.Errorf("projected = %v, want [OrderShipped] - the read model would go stale after shipping", projector.projected)
	}
}

func TestHandleShipUnconfirmedOrderIsRejected(t *testing.T) {
	h, _, _ := newHandler(t)
	seedPending(t, h, "order-1")

	err := h.Handle(ShipOrderCommand{OrderID: "order-1", TrackingNumber: "TRK-9"})

	if err == nil || err.Error() != "can only ship confirmed orders" {
		t.Fatalf("Handle = %v, want the unconfirmed rejection", err)
	}
}

func TestHandleDeliverOrder(t *testing.T) {
	h, store, projector := newHandler(t)
	seedShipped(t, h, "order-1")
	projector.projected = nil

	if err := h.Handle(DeliverOrderCommand{OrderID: "order-1", ReceivedBy: "cust-1"}); err != nil {
		t.Fatalf("Handle(DeliverOrderCommand) = %v", err)
	}

	got := store.types("order-1")
	if got[len(got)-1] != events.OrderDeliveredEvent {
		t.Errorf("last persisted event = %v, want OrderDelivered", got[len(got)-1])
	}
	if len(projector.projected) != 1 || projector.projected[0] != events.OrderDeliveredEvent {
		t.Errorf("projected = %v, want [OrderDelivered]", projector.projected)
	}
}

func TestHandleDeliverUnshippedOrderIsRejected(t *testing.T) {
	h, _, _ := newHandler(t)
	seedConfirmed(t, h, "order-1")

	err := h.Handle(DeliverOrderCommand{OrderID: "order-1", ReceivedBy: "cust-1"})

	if err == nil || err.Error() != "can only deliver shipped orders" {
		t.Fatalf("Handle = %v, want the unshipped rejection", err)
	}
}

// The saga-driven commands are the reason this package exists: each one must
// land in the event store so a replay reconstructs inventory/payment state.
func TestHandleSagaRecordingCommands(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want events.EventType
	}{
		{
			name: "inventory reservation",
			cmd: RecordInventoryReservationCommand{
				OrderID:       "order-1",
				ReservationID: "res-1",
				Items:         []events.ReservedItem{{ProductID: "prod-1", Quantity: 2}},
			},
			want: events.InventoryReservedEvent,
		},
		{
			name: "inventory release",
			cmd:  RecordInventoryReleaseCommand{OrderID: "order-1", ReservationID: "res-1", Reason: "payment failed"},
			want: events.InventoryReleasedEvent,
		},
		{
			name: "payment",
			cmd: RecordPaymentCommand{
				OrderID:       "order-1",
				PaymentID:     "pay-1",
				PaymentMethod: "card",
				TransactionID: "txn-1",
				Amount:        1000,
			},
			want: events.PaymentProcessedEvent,
		},
		{
			name: "payment failure",
			cmd:  RecordPaymentFailureCommand{OrderID: "order-1", PaymentID: "pay-1", Reason: "declined"},
			want: events.PaymentFailedEvent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, store, projector := newHandler(t)
			seedPending(t, h, "order-1")
			projector.projected = nil

			if err := h.Handle(tt.cmd); err != nil {
				t.Fatalf("Handle(%T) = %v", tt.cmd, err)
			}

			got := store.types("order-1")
			if got[len(got)-1] != tt.want {
				t.Errorf("last persisted event = %v, want %v", got[len(got)-1], tt.want)
			}
			if len(projector.projected) != 1 || projector.projected[0] != tt.want {
				t.Errorf("projected = %v, want [%v]", projector.projected, tt.want)
			}
		})
	}
}

func TestHandleSagaCommandWrapsSaveError(t *testing.T) {
	h, store, _ := newHandler(t)
	seedPending(t, h, "order-1")
	store.saveErr = errors.New("db down")

	err := h.Handle(RecordPaymentCommand{OrderID: "order-1", PaymentID: "pay-1", Amount: 10})

	if err == nil || !errors.Is(err, store.saveErr) {
		t.Fatalf("Handle = %v, want an error wrapping %v", err, store.saveErr)
	}
}

func TestHandleWithoutProjectorDoesNotPanic(t *testing.T) {
	store := newFakeEventStore()
	h := NewCommandHandler(store, zap.NewNop()) // SetProjector deliberately not called

	if err := h.Handle(CreateOrderCommand{OrderID: "order-1", TenantID: "t", CustomerID: "c"}); err != nil {
		t.Fatalf("Handle without a projector = %v", err)
	}

	if got := store.types("order-1"); len(got) != 1 {
		t.Errorf("persisted events = %v, want the OrderCreated event", got)
	}
}

// The event store is the source of truth, so a read-model failure is logged
// rather than surfaced - the command still succeeds.
func TestProjectorErrorDoesNotFailTheCommand(t *testing.T) {
	store := newFakeEventStore()
	projector := &recordingProjector{err: errors.New("read model unavailable")}
	h := NewCommandHandler(store, zap.NewNop())
	h.SetProjector(projector)

	err := h.Handle(CreateOrderCommand{OrderID: "order-1", TenantID: "t", CustomerID: "c"})

	if err != nil {
		t.Fatalf("Handle = %v, want nil despite the projector failing", err)
	}
	if len(projector.projected) != 1 {
		t.Errorf("projector saw %v, want the event to still be offered", projector.projected)
	}
	if got := store.types("order-1"); len(got) != 1 {
		t.Errorf("persisted events = %v, want the event kept in the store", got)
	}
}
