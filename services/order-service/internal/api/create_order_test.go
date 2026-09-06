package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/ecommerce/order-service/internal/domain/commands"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"go.uber.org/zap"
)

// setupCreateRouter registers CreateOrder the way the real router does: on the
// public group, with no auth middleware. Guests reach this route.
func setupCreateRouter(store *inMemoryEventStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	cmdHandler := commands.NewCommandHandler(store, logger)
	h := NewCommandHandler(cmdHandler, store, logger, "", "")

	router := gin.New()
	router.POST("/api/v1/orders", h.CreateOrder)
	return router
}

func eventTypesFor(store *inMemoryEventStore, orderID string) []events.EventType {
	evs, _ := store.GetEvents(orderID)
	out := make([]events.EventType, 0, len(evs))
	for _, e := range evs {
		out = append(out, e.GetEventType())
	}
	return out
}

func createdOrderID(t *testing.T, body []byte) string {
	t.Helper()
	var resp CreateOrderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode response %q: %v", string(body), err)
	}
	if resp.OrderID == "" {
		t.Fatalf("response carried no order id: %s", string(body))
	}
	return resp.OrderID
}

const guestOrderBody = `{
  "tenant_id": "tenant_saajan",
  "guest_email": "buyer@example.test",
  "guest_name": "Guest Buyer",
  "guest_phone": "01700000000",
  "shipping_address": {"street":"1 St","city":"Dhaka","state":"DH","postal_code":"1200","country":"Bangladesh"},
  "billing_address": {"street":"1 St","city":"Dhaka","state":"DH","postal_code":"1200","country":"Bangladesh"},
  "items": [
    {"product_id":"prod-1","sku":"BOARD-001","name":"Strategy Board Game: Settlers","quantity":1,"unit_price":2499},
    {"product_id":"prod-2","variant_id":"red","sku":"LEGO-001","name":"Building Blocks Starter Set","quantity":2,"unit_price":1799}
  ]
}`

// A guest must be able to submit a complete order in one unauthenticated call.
// Before this, checkout created the order and then got 401 adding items,
// stranding an empty order.
func TestCreateOrder_GuestWithItems(t *testing.T) {
	store := newInMemoryEventStore()
	router := setupCreateRouter(store)

	w := doJSON(router, http.MethodPost, "/api/v1/orders", "", guestOrderBody)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
	}
	orderID := createdOrderID(t, w.Body.Bytes())

	got := eventTypesFor(store, orderID)
	want := []events.EventType{
		events.OrderCreatedEvent,
		events.OrderItemAddedEvent,
		events.OrderItemAddedEvent,
	}
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("events[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// The order must not be left empty: the items have to reach the aggregate with
// their quantities and prices intact so the total is right.
func TestCreateOrder_GuestItemsCarryQuantityAndPrice(t *testing.T) {
	store := newInMemoryEventStore()
	router := setupCreateRouter(store)

	w := doJSON(router, http.MethodPost, "/api/v1/orders", "", guestOrderBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
	}
	orderID := createdOrderID(t, w.Body.Bytes())

	evs, _ := store.GetEvents(orderID)
	var total float64
	seen := map[string]events.OrderItemAdded{}
	for _, e := range evs {
		if added, ok := e.(events.OrderItemAdded); ok {
			seen[added.SKU] = added
			total += added.TotalPrice
		}
	}

	if len(seen) != 2 {
		t.Fatalf("item events = %v, want 2", seen)
	}
	if board := seen["BOARD-001"]; board.Quantity != 1 || board.UnitPrice != 2499 || board.TotalPrice != 2499 {
		t.Errorf("BOARD-001 = %+v, want qty 1 @ 2499", board)
	}
	if lego := seen["LEGO-001"]; lego.Quantity != 2 || lego.UnitPrice != 1799 || lego.TotalPrice != 3598 {
		t.Errorf("LEGO-001 = %+v, want qty 2 @ 1799 = 3598", lego)
	}
	if total != 2499+3598 {
		t.Errorf("order total = %v, want %v", total, 2499.0+3598.0)
	}
	// The variant has to survive, otherwise the wrong SKU gets picked.
	if lego := seen["LEGO-001"]; lego.VariantID != "red" {
		t.Errorf("LEGO-001 variant = %q, want red", lego.VariantID)
	}
}

// Staff/POS create an empty order and add items through the authenticated
// route, so items must stay optional.
func TestCreateOrder_WithoutItemsStillWorks(t *testing.T) {
	store := newInMemoryEventStore()
	router := setupCreateRouter(store)

	body := `{
      "tenant_id": "tenant_saajan",
      "customer_id": "cust-1",
      "shipping_address": {"street":"1 St","city":"Dhaka","state":"DH","postal_code":"1200","country":"Bangladesh"},
      "billing_address": {"street":"1 St","city":"Dhaka","state":"DH","postal_code":"1200","country":"Bangladesh"}
    }`

	w := doJSON(router, http.MethodPost, "/api/v1/orders", "", body)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
	}
	orderID := createdOrderID(t, w.Body.Bytes())
	if got := eventTypesFor(store, orderID); len(got) != 1 || got[0] != events.OrderCreatedEvent {
		t.Errorf("events = %v, want just [OrderCreated]", got)
	}
}

// A bad item must not leave a half-built order sitting in "pending" — the
// stranded-order bug this whole change exists to fix. The order is cancelled
// and the caller told the request failed.
func TestCreateOrder_RejectedItemCancelsTheOrder(t *testing.T) {
	store := newInMemoryEventStore()
	router := setupCreateRouter(store)

	// Quantity 0 passes binding (min=1 only applies to the standalone
	// AddOrderItemRequest binding) but the aggregate rejects it.
	body := `{
      "tenant_id": "tenant_saajan",
      "guest_email": "buyer@example.test",
      "shipping_address": {"street":"1 St","city":"Dhaka","state":"DH","postal_code":"1200","country":"Bangladesh"},
      "billing_address": {"street":"1 St","city":"Dhaka","state":"DH","postal_code":"1200","country":"Bangladesh"},
      "items": [
        {"product_id":"prod-1","sku":"OK-001","name":"Fine","quantity":1,"unit_price":100},
        {"product_id":"prod-2","sku":"BAD-001","name":"Bad","quantity":0,"unit_price":100}
      ]
    }`

	w := doJSON(router, http.MethodPost, "/api/v1/orders", "", body)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body %s)", w.Code, w.Body.String())
	}

	// Exactly one order stream exists; it must end cancelled, not pending.
	if len(store.events) != 1 {
		t.Fatalf("expected one order stream, got %d", len(store.events))
	}
	for id := range store.events {
		got := eventTypesFor(store, id)
		if got[len(got)-1] != events.OrderCancelledEvent {
			t.Errorf("stream %s = %v, want it to end in OrderCancelled so no empty order is stranded", id, got)
		}
	}
}

func TestCreateOrder_RequiresCustomerOrGuestEmail(t *testing.T) {
	store := newInMemoryEventStore()
	router := setupCreateRouter(store)

	body := `{
      "tenant_id": "tenant_saajan",
      "shipping_address": {"street":"1 St","city":"Dhaka","state":"DH","postal_code":"1200","country":"Bangladesh"},
      "billing_address": {"street":"1 St","city":"Dhaka","state":"DH","postal_code":"1200","country":"Bangladesh"}
    }`

	w := doJSON(router, http.MethodPost, "/api/v1/orders", "", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
	}
	if len(store.events) != 0 {
		t.Errorf("an order was persisted for an identity-less request: %v", store.events)
	}
}

// Guarding the decision: /items stays behind auth on purpose. If it were
// public, anyone holding an order id could mutate someone else's pending
// order. This asserts the authenticated router still refuses an anonymous
// caller, so opening it later is a deliberate act and not an accident.
func TestAddOrderItem_StaysAuthenticatedForAnonymousCallers(t *testing.T) {
	store := newInMemoryEventStore()
	seedOrder(t, store, "order-1", "tenant_saajan")
	router := setupCommandRouter(store, &recordingPublisher{})

	// No tenant header => no authenticated identity in context.
	w := doJSON(router, http.MethodPost, "/api/v1/orders/order-1/items", "",
		`{"product_id":"prod-1","sku":"S-1","name":"N","quantity":1,"unit_price":100}`)

	if w.Code == http.StatusOK || w.Code == http.StatusCreated {
		t.Fatalf("anonymous caller added an item (status %d) — /items must stay auth-gated", w.Code)
	}
	if got := eventTypesFor(store, "order-1"); len(got) != 1 {
		t.Errorf("events = %v, want the seeded OrderCreated only", got)
	}
}
