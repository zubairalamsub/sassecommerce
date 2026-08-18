package saga

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/ecommerce/order-service/internal/domain/aggregates"
	"github.com/yourusername/ecommerce/order-service/internal/domain/commands"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------- test doubles

// memEventStore is an in-memory eventstore.EventStore. failSaveOn lets a test
// reject a specific write (used to force a non-optional saga step to fail).
type memEventStore struct {
	mu         sync.Mutex
	streams    map[string][]events.Event
	failSaveOn func([]events.Event) bool
}

func newMemEventStore() *memEventStore {
	return &memEventStore{streams: make(map[string][]events.Event)}
}

func (s *memEventStore) Save(aggregateID string, evs []events.Event, expectedVersion int) error {
	if len(evs) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failSaveOn != nil && s.failSaveOn(evs) {
		return errors.New("event store rejected save")
	}

	// Mirror PostgresEventStore's optimistic concurrency check:
	// COALESCE(MAX(version), -1) must equal expectedVersion.
	currentVersion := -1
	for _, e := range s.streams[aggregateID] {
		if v := e.GetVersion(); v > currentVersion {
			currentVersion = v
		}
	}
	if currentVersion != expectedVersion {
		return fmt.Errorf("concurrency conflict: aggregate version mismatch (current %d, expected %d)",
			currentVersion, expectedVersion)
	}

	s.streams[aggregateID] = append(s.streams[aggregateID], evs...)
	return nil
}

func (s *memEventStore) GetEvents(aggregateID string) ([]events.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]events.Event(nil), s.streams[aggregateID]...), nil
}

func (s *memEventStore) GetEventsByType(eventType events.EventType, limit int) ([]events.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]events.Event, 0)
	for _, stream := range s.streams {
		for _, e := range stream {
			if e.GetEventType() == eventType {
				out = append(out, e)
			}
		}
	}
	return out, nil
}

func (s *memEventStore) GetAllEvents(offset, limit int) ([]events.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]events.Event, 0)
	for _, stream := range s.streams {
		out = append(out, stream...)
	}
	return out, nil
}

// eventTypes lists the event types persisted for an aggregate, in order.
func (s *memEventStore) eventTypes(aggregateID string) []string {
	evs, _ := s.GetEvents(aggregateID)
	out := make([]string, 0, len(evs))
	for _, e := range evs {
		out = append(out, string(e.GetEventType()))
	}
	return out
}

// recordedCall captures one outbound HTTP call made by a saga step.
type recordedCall struct {
	Service string // "inventory" or "payment"
	Kind    string // "create", "cancel" or "refund"
	Path    string
	Auth    string
	Tenant  string
	Body    map[string]interface{}
}

// fakeServices stands in for the inventory and payment microservices. Status
// codes and bodies are per-test knobs; every call is recorded in order so a test
// can assert both what was called and the sequence.
type fakeServices struct {
	mu    sync.Mutex
	calls []recordedCall

	invCreateStatus int
	invCreateBody   string
	invCancelStatus int

	payCreateStatus int
	payCreateBody   string
	payRefundStatus int

	reservationSeq int

	inventory *httptest.Server
	payment   *httptest.Server
}

func newFakeServices(t *testing.T) *fakeServices {
	t.Helper()
	f := &fakeServices{
		invCreateStatus: http.StatusCreated,
		invCancelStatus: http.StatusOK,
		payCreateStatus: http.StatusCreated,
		payCreateBody:   `{"id":"pay-1","status":"succeeded"}`,
		payRefundStatus: http.StatusOK,
	}

	f.inventory = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/cancel") {
			f.record(r, "inventory", "cancel")
			f.reply(w, f.invCancelStatus, `{"status":"cancelled"}`)
			return
		}
		f.record(r, "inventory", "create")
		body := f.invCreateBody
		if body == "" {
			f.mu.Lock()
			f.reservationSeq++
			seq := f.reservationSeq
			f.mu.Unlock()
			body = `{"id":"res-` + itoa(seq) + `"}`
		}
		f.reply(w, f.invCreateStatus, body)
	}))
	t.Cleanup(f.inventory.Close)

	f.payment = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/refund") {
			f.record(r, "payment", "refund")
			f.reply(w, f.payRefundStatus, `{"status":"refunded"}`)
			return
		}
		f.record(r, "payment", "create")
		f.reply(w, f.payCreateStatus, f.payCreateBody)
	}))
	t.Cleanup(f.payment.Close)

	return f
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

func (f *fakeServices) record(r *http.Request, service, kind string) {
	var body map[string]interface{}
	_ = json.NewDecoder(r.Body).Decode(&body)

	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, recordedCall{
		Service: service,
		Kind:    kind,
		Path:    r.URL.Path,
		Auth:    r.Header.Get("Authorization"),
		Tenant:  r.Header.Get("X-Tenant-ID"),
		Body:    body,
	})
}

func (f *fakeServices) reply(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (f *fakeServices) recorded() []recordedCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedCall(nil), f.calls...)
}

// sequence renders the calls as "service:kind" strings for order assertions.
func (f *fakeServices) sequence() []string {
	out := make([]string, 0)
	for _, c := range f.recorded() {
		out = append(out, c.Service+":"+c.Kind)
	}
	return out
}

func (f *fakeServices) callsOfKind(kind string) []recordedCall {
	out := make([]recordedCall, 0)
	for _, c := range f.recorded() {
		if c.Kind == kind {
			out = append(out, c)
		}
	}
	return out
}

// ------------------------------------------------------------------- fixtures

const (
	testOrderID  = "order-1"
	testTenantID = "tenant-1"
)

type sagaFixture struct {
	store    *memEventStore
	handler  *commands.CommandHandler
	services *fakeServices
	order    *aggregates.Order
}

// newSagaFixture seeds a pending order with the given items in the event store
// and rebuilds the aggregate the saga will operate on.
func newSagaFixture(t *testing.T, items ...commands.AddOrderItemCommand) *sagaFixture {
	t.Helper()

	store := newMemEventStore()
	handler := commands.NewCommandHandler(store, zap.NewNop())

	require.NoError(t, handler.Handle(commands.CreateOrderCommand{
		OrderID:    testOrderID,
		TenantID:   testTenantID,
		CustomerID: "customer-1",
		ShippingAddress: events.Address{
			Street: "1 Main St", City: "Dhaka", State: "Dhaka", PostalCode: "1200", Country: "BD",
		},
		BillingAddress: events.Address{
			Street: "1 Main St", City: "Dhaka", State: "Dhaka", PostalCode: "1200", Country: "BD",
		},
	}))

	for _, item := range items {
		item.OrderID = testOrderID
		require.NoError(t, handler.Handle(item))
	}

	return &sagaFixture{
		store:    store,
		handler:  handler,
		services: newFakeServices(t),
		order:    loadOrder(t, store),
	}
}

func defaultItems() []commands.AddOrderItemCommand {
	return []commands.AddOrderItemCommand{
		{ProductID: "product-1", SKU: "SKU-1", Name: "Shirt", Quantity: 2, UnitPrice: 500},
	}
}

// loadOrder rebuilds the order aggregate from its event history, the same way
// the command handler does.
func loadOrder(t *testing.T, store *memEventStore) *aggregates.Order {
	t.Helper()
	evs, err := store.GetEvents(testOrderID)
	require.NoError(t, err)
	require.NotEmpty(t, evs, "expected seeded order events")

	order := &aggregates.Order{ID: testOrderID, Items: make(map[string]*aggregates.OrderItem)}
	order.LoadFromHistory(evs)
	return order
}

func (f *sagaFixture) newSaga(authToken string) *OrderSaga {
	return NewOrderSaga(
		testOrderID,
		f.order,
		f.handler,
		zap.NewNop(),
		f.services.inventory.URL,
		f.services.payment.URL,
		authToken,
	)
}

// statusInStore rebuilds the aggregate from persisted events and returns status.
func (f *sagaFixture) statusInStore(t *testing.T) aggregates.OrderStatus {
	t.Helper()
	return loadOrder(t, f.store).Status
}

// ------------------------------------------------------------------ happy path

func TestOrderSaga_HappyPath(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)

	err := f.newSaga("test-token").Execute()

	require.NoError(t, err)
	assert.Equal(t, []string{"inventory:create", "payment:create"}, f.services.sequence(),
		"happy path reserves inventory then takes payment, with no compensation")
	assert.Equal(t, aggregates.StatusConfirmed, f.statusInStore(t), "order should be confirmed")
}

// Each order item is reserved with its own request, and the reservation id kept
// for compensation is the one from the last call.
func TestOrderSaga_ReservesEachItemSeparately(t *testing.T) {
	f := newSagaFixture(t,
		commands.AddOrderItemCommand{ProductID: "product-1", SKU: "SKU-1", Name: "Shirt", Quantity: 1, UnitPrice: 100},
		commands.AddOrderItemCommand{ProductID: "product-2", SKU: "SKU-2", Name: "Hat", Quantity: 3, UnitPrice: 50},
	)

	require.NoError(t, f.newSaga("test-token").Execute())

	creates := f.services.callsOfKind("create")
	inventoryCreates := make([]recordedCall, 0)
	for _, c := range creates {
		if c.Service == "inventory" {
			inventoryCreates = append(inventoryCreates, c)
		}
	}
	require.Len(t, inventoryCreates, 2, "one reservation request per order item")

	productIDs := []string{
		inventoryCreates[0].Body["productId"].(string),
		inventoryCreates[1].Body["productId"].(string),
	}
	assert.ElementsMatch(t, []string{"product-1", "product-2"}, productIDs)
	assert.Equal(t, "res-2", loadOrder(t, f.store).ReservationID,
		"the last reservation id is the one recorded on the order")
}

// The reservation payload carries the tenant, order and expiry the inventory
// service expects.
func TestOrderSaga_ReservationRequestShape(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)

	require.NoError(t, f.newSaga("test-token").Execute())

	call := f.services.callsOfKind("create")[0]
	assert.Equal(t, "/api/v1/inventory/reservations", call.Path)
	assert.Equal(t, testTenantID, call.Body["tenantId"])
	assert.Equal(t, testOrderID, call.Body["orderId"])
	assert.Equal(t, "product-1", call.Body["productId"])
	assert.Equal(t, float64(2), call.Body["quantity"])
	assert.Equal(t, float64(30), call.Body["expirationMinutes"])
	assert.Equal(t, "system", call.Body["createdBy"])
}

// The payment payload carries the order total and currency from the aggregate.
func TestOrderSaga_PaymentRequestShape(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)

	require.NoError(t, f.newSaga("test-token").Execute())

	var payCreate recordedCall
	for _, c := range f.services.recorded() {
		if c.Service == "payment" && c.Kind == "create" {
			payCreate = c
		}
	}
	assert.Equal(t, "/api/v1/payments", payCreate.Path)
	assert.Equal(t, testTenantID, payCreate.Body["tenantId"])
	assert.Equal(t, "customer-1", payCreate.Body["customerId"])
	assert.Equal(t, testOrderID, payCreate.Body["orderId"])
	assert.Equal(t, float64(1000), payCreate.Body["amount"], "2 x 500 from the aggregate total")
	assert.Equal(t, "BDT", payCreate.Body["currency"])
}

// --------------------------------------------------------- credential handling

func TestOrderSaga_PropagatesAuthAndTenantHeaders(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)

	require.NoError(t, f.newSaga("test-token").Execute())

	calls := f.services.recorded()
	require.NotEmpty(t, calls)
	for _, c := range calls {
		assert.Equal(t, "Bearer test-token", c.Auth, "%s:%s must forward the caller credential", c.Service, c.Kind)
		assert.Equal(t, testTenantID, c.Tenant, "%s:%s must forward the tenant", c.Service, c.Kind)
	}
}

func TestOrderSaga_OmitsAuthHeaderWhenTokenEmpty(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)

	require.NoError(t, f.newSaga("").Execute())

	for _, c := range f.services.recorded() {
		assert.Empty(t, c.Auth, "no Authorization header should be sent when no token is configured")
		assert.Equal(t, testTenantID, c.Tenant)
	}
}

// ------------------------------------------------------------- optional steps

// Inventory reservation is declared optional, so a failure is logged and skipped
// rather than failing the order.
func TestOrderSaga_InventoryFailureIsSkipped(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.services.invCreateStatus = http.StatusInternalServerError

	err := f.newSaga("test-token").Execute()

	require.NoError(t, err, "an optional step failure must not fail the saga")
	assert.Equal(t, aggregates.StatusConfirmed, f.statusInStore(t))
	assert.Equal(t, []string{"inventory:create", "payment:create"}, f.services.sequence(),
		"payment still runs after inventory is skipped")
	assert.Empty(t, loadOrder(t, f.store).ReservationID, "no reservation recorded when inventory failed")
}

// Payment is also optional (e.g. cash-on-delivery), so a failure is skipped.
func TestOrderSaga_PaymentFailureIsSkipped(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.services.payCreateStatus = http.StatusBadGateway

	err := f.newSaga("test-token").Execute()

	require.NoError(t, err)
	assert.Equal(t, aggregates.StatusConfirmed, f.statusInStore(t))
	assert.Empty(t, loadOrder(t, f.store).PaymentID, "no payment recorded when the payment call failed")
}

// A 2xx payment response without an id is treated as a step failure — and since
// payment is optional, the saga continues.
func TestOrderSaga_PaymentResponseWithoutIDIsSkipped(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.services.payCreateBody = `{"status":"succeeded"}`

	err := f.newSaga("test-token").Execute()

	require.NoError(t, err)
	assert.Equal(t, aggregates.StatusConfirmed, f.statusInStore(t))
	assert.Empty(t, loadOrder(t, f.store).PaymentID)
}

func TestOrderSaga_BothOptionalStepsFailStillConfirms(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.services.invCreateStatus = http.StatusInternalServerError
	f.services.payCreateStatus = http.StatusInternalServerError

	err := f.newSaga("test-token").Execute()

	require.NoError(t, err)
	assert.Equal(t, aggregates.StatusConfirmed, f.statusInStore(t))
}

// An unreachable dependency (connection error rather than an HTTP status) is
// handled the same way as a bad status for an optional step.
func TestOrderSaga_UnreachableInventoryIsSkipped(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.services.inventory.Close() // force a dial error

	err := f.newSaga("test-token").Execute()

	require.NoError(t, err)
	assert.Equal(t, aggregates.StatusConfirmed, f.statusInStore(t))
}

// --------------------------------------------------------------- compensation

// failOnConfirm makes the event store reject the OrderConfirmed write, which is
// the cleanest way to fail the one non-optional step.
func failOnConfirm(evs []events.Event) bool {
	for _, e := range evs {
		if e.GetEventType() == events.OrderConfirmedEvent {
			return true
		}
	}
	return false
}

// When the non-optional ConfirmOrder step fails, completed steps compensate in
// reverse order and the order is cancelled.
func TestOrderSaga_ConfirmFailureCompensatesInReverseOrder(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.store.failSaveOn = failOnConfirm

	err := f.newSaga("test-token").Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "saga step ConfirmOrder failed")

	assert.Equal(t, []string{
		"inventory:create",
		"payment:create",
		"payment:refund",   // compensated first — it completed last
		"inventory:cancel", // compensated second
	}, f.services.sequence(), "compensation must unwind completed steps in reverse")

	assert.Equal(t, aggregates.StatusCancelled, f.statusInStore(t),
		"compensation cancels the order")
}

// Compensation targets the specific reservation and payment that were created.
func TestOrderSaga_CompensationTargetsCreatedResources(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.store.failSaveOn = failOnConfirm

	require.Error(t, f.newSaga("test-token").Execute())

	cancels := f.services.callsOfKind("cancel")
	require.Len(t, cancels, 1)
	assert.Equal(t, "/api/v1/inventory/reservations/res-1/cancel", cancels[0].Path)
	assert.Equal(t, "system", cancels[0].Body["cancelled_by"])

	refunds := f.services.callsOfKind("refund")
	require.Len(t, refunds, 1)
	assert.Equal(t, "/api/v1/payments/pay-1/refund", refunds[0].Path)
	assert.NotEmpty(t, refunds[0].Body["reason"])
}

// Only steps that actually completed are compensated: when inventory was
// skipped there is no reservation to cancel.
func TestOrderSaga_SkippedStepIsNotCompensated(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.services.invCreateStatus = http.StatusInternalServerError
	f.store.failSaveOn = failOnConfirm

	require.Error(t, f.newSaga("test-token").Execute())

	assert.Empty(t, f.services.callsOfKind("cancel"), "nothing was reserved, so nothing to cancel")
	require.Len(t, f.services.callsOfKind("refund"), 1, "the payment that did succeed is refunded")
	assert.Equal(t, aggregates.StatusCancelled, f.statusInStore(t))
}

// A failing compensation step aborts the unwind: the remaining steps are not
// compensated and the order is left uncancelled. The original step error is
// still what the caller sees.
func TestOrderSaga_CompensationFailureAbortsUnwind(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.store.failSaveOn = failOnConfirm
	f.services.payRefundStatus = http.StatusInternalServerError

	err := f.newSaga("test-token").Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "saga step ConfirmOrder failed",
		"the caller still sees the original failure, not the compensation error")

	assert.Equal(t, []string{"inventory:create", "payment:create", "payment:refund"}, f.services.sequence(),
		"the inventory cancel is never attempted after the refund fails")
	assert.Equal(t, aggregates.StatusPending, f.statusInStore(t),
		"the order is left pending when compensation could not complete")
}

// ------------------------------------- record-failure must not skip (regression)

func failOnEventType(t events.EventType) func([]events.Event) bool {
	return func(evs []events.Event) bool {
		for _, e := range evs {
			if e.GetEventType() == t {
				return true
			}
		}
		return false
	}
}

// Regression: ProcessPayment is an optional step, but once the payment has been
// CAPTURED a failure to record it must not be skipped as "optional" — the money
// would be stranded. It has to fail the saga and refund.
//
// Before this was fixed, the saga logged "Optional saga step failed, skipping",
// returned nil, and never issued the refund.
func TestOrderSaga_PaymentRecordFailureIsRefundedNotSkipped(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.store.failSaveOn = failOnEventType(events.PaymentProcessedEvent)

	err := f.newSaga("test-token").Execute()

	require.Error(t, err, "a captured payment that cannot be recorded must fail the saga")
	assert.Contains(t, err.Error(), "saga step ProcessPayment failed")

	assert.Equal(t, []string{
		"inventory:create",
		"payment:create",
		"payment:refund",   // the captured payment is returned
		"inventory:cancel", // and the stock hold released
	}, f.services.sequence(), "the captured payment must be refunded, not stranded")

	assert.Equal(t, aggregates.StatusCancelled, f.statusInStore(t))
}

// Same rule for inventory: once stock is held, a failure to record the
// reservation must release it rather than silently continue.
func TestOrderSaga_InventoryRecordFailureIsReleasedNotSkipped(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.store.failSaveOn = failOnEventType(events.InventoryReservedEvent)

	err := f.newSaga("test-token").Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "saga step ReserveInventory failed")

	assert.Equal(t, []string{
		"inventory:create",
		"inventory:cancel", // the hold is released
	}, f.services.sequence(), "payment is never attempted once the reservation failed to record")

	assert.Equal(t, aggregates.StatusCancelled, f.statusInStore(t))
}

// The optional-skip path must still work when the external dependency itself
// fails, because nothing was acquired and there is nothing to unwind.
func TestOrderSaga_ExternalFailureStillSkipsWithoutCompensating(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.services.payCreateStatus = http.StatusServiceUnavailable

	err := f.newSaga("test-token").Execute()

	require.NoError(t, err, "an unavailable optional dependency is still skipped")
	assert.Equal(t, []string{"inventory:create", "payment:create"}, f.services.sequence(),
		"no compensation, because nothing was captured")
	assert.Equal(t, aggregates.StatusConfirmed, f.statusInStore(t))
}

// ------------------------------------------------------------ step primitives

func TestSagaSteps_GetName(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	s := f.newSaga("test-token")

	assert.Equal(t, "ReserveInventory", s.NewReserveInventoryStep().GetName())
	assert.Equal(t, "ProcessPayment", s.NewProcessPaymentStep().GetName())
	assert.Equal(t, "ConfirmOrder", s.NewConfirmOrderStep().GetName())
}

// Compensating a step that never ran is a no-op, not an error or a stray call.
func TestReserveInventoryStep_CompensateWithoutReservationIsNoop(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	step := f.newSaga("test-token").NewReserveInventoryStep()

	require.NoError(t, step.Compensate())
	assert.Empty(t, f.services.recorded(), "no HTTP call should be made")
}

func TestProcessPaymentStep_CompensateWithoutPaymentIsNoop(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	step := f.newSaga("test-token").NewProcessPaymentStep()

	require.NoError(t, step.Compensate())
	assert.Empty(t, f.services.recorded(), "no HTTP call should be made")
}

// ConfirmOrderStep has no compensation of its own — the saga's cancel handles it.
func TestConfirmOrderStep_CompensateIsNoop(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	step := f.newSaga("test-token").NewConfirmOrderStep()

	assert.NoError(t, step.Compensate())
}

// A non-2xx from the inventory service surfaces the status code in the error.
func TestReserveInventoryStep_ErrorIncludesStatusCode(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.services.invCreateStatus = http.StatusConflict

	err := f.newSaga("test-token").NewReserveInventoryStep().Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reserve inventory")
	assert.Contains(t, err.Error(), "409")
}

func TestProcessPaymentStep_ErrorIncludesStatusCode(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.services.payCreateStatus = http.StatusPaymentRequired

	err := f.newSaga("test-token").NewProcessPaymentStep().Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to process payment")
	assert.Contains(t, err.Error(), "402")
}

// An order with no items cannot be confirmed, so the aggregate rule surfaces
// through the step.
func TestConfirmOrderStep_FailsForOrderWithoutItems(t *testing.T) {
	f := newSagaFixture(t) // no items

	err := f.newSaga("test-token").NewConfirmOrderStep().Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to confirm order")
}

// ------------------------------------------------------ event-sourcing surface

// Regression test: the saga's inventory and payment recordings must go through
// the command handler so they are persisted to the event store (and from there
// projected to the read model and published). They were previously recorded on
// the local aggregate and then dropped by MarkEventsAsCommitted, which lost
// reservation and payment state on replay.
func TestOrderSaga_InventoryAndPaymentEventsArePersisted(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)

	require.NoError(t, f.newSaga("test-token").Execute())

	assert.Equal(t, []string{
		"OrderCreated",
		"OrderItemAdded",
		"InventoryReserved",
		"PaymentProcessed",
		"OrderConfirmed",
	}, f.store.eventTypes(testOrderID))

	// A replayed aggregate carries the reservation and payment.
	replayed := loadOrder(t, f.store)
	assert.Equal(t, "res-1", replayed.ReservationID)
	assert.Equal(t, "pay-1", replayed.PaymentID)
}

// Compensation must also persist its events, and because the payment is now on
// the replayed aggregate, cancelling emits the OrderRefunded audit event that
// Order.Cancel documents.
func TestOrderSaga_CompensationEventsArePersisted(t *testing.T) {
	f := newSagaFixture(t, defaultItems()...)
	f.store.failSaveOn = failOnConfirm

	require.Error(t, f.newSaga("test-token").Execute())

	persisted := f.store.eventTypes(testOrderID)
	assert.Equal(t, []string{
		"OrderCreated",
		"OrderItemAdded",
		"InventoryReserved",
		"PaymentProcessed",
		"PaymentFailed",     // payment compensated first
		"InventoryReleased", // then inventory
		"OrderCancelled",
		"OrderRefunded", // emitted because the replayed order has a PaymentID
	}, persisted)

	assert.Empty(t, loadOrder(t, f.store).ReservationID,
		"releasing the reservation clears it on the aggregate")
}
