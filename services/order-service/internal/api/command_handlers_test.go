package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yourusername/ecommerce/order-service/internal/domain/aggregates"
	"github.com/yourusername/ecommerce/order-service/internal/domain/commands"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"github.com/yourusername/ecommerce/order-service/internal/messaging"
	"go.uber.org/zap"
)

// inMemoryEventStore is a minimal eventstore.EventStore for handler tests.
type inMemoryEventStore struct {
	mu     sync.Mutex
	events map[string][]events.Event
}

func newInMemoryEventStore() *inMemoryEventStore {
	return &inMemoryEventStore{events: make(map[string][]events.Event)}
}

func (s *inMemoryEventStore) Save(aggregateID string, evs []events.Event, expectedVersion int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[aggregateID] = append(s.events[aggregateID], evs...)
	return nil
}

func (s *inMemoryEventStore) GetEvents(aggregateID string) ([]events.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.events[aggregateID], nil
}

func (s *inMemoryEventStore) GetEventsByType(eventType events.EventType, limit int) ([]events.Event, error) {
	return nil, nil
}

func (s *inMemoryEventStore) GetAllEvents(offset, limit int) ([]events.Event, error) {
	return nil, nil
}

// recordingPublisher records whether PublishReceiptRequested was invoked so
// tests can assert that a cross-tenant SendReceipt never emails order contents.
type recordingPublisher struct {
	called bool
	err    error
}

func (p *recordingPublisher) PublishReceiptRequested(ctx context.Context, payload messaging.ReceiptRequestedPayload) error {
	p.called = true
	return p.err
}

func (p *recordingPublisher) Close() error { return nil }

func seedOrder(t *testing.T, store *inMemoryEventStore, orderID, tenantID string) {
	t.Helper()
	addr := events.Address{Street: "1 St", City: "C", State: "S", PostalCode: "1", Country: "X"}
	order := aggregates.NewOrder(orderID, tenantID, "cust-1", addr, addr)
	if err := store.Save(orderID, order.GetUncommittedEvents(), -1); err != nil {
		t.Fatalf("seed order: %v", err)
	}
}

func setupCommandRouter(store *inMemoryEventStore, publisher messaging.NotificationPublisher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	cmdHandler := commands.NewCommandHandler(store, logger)
	h := NewCommandHandler(cmdHandler, store, logger, "", "")
	if publisher != nil {
		h.SetNotificationPublisher(publisher)
	}

	router := gin.New()
	// Stub the shared Auth middleware: populate tenant_id from a test header so
	// handlers read the JWT-derived tenant, never the path/body. Absence of the
	// header simulates an anonymous caller (empty context).
	router.Use(func(c *gin.Context) {
		if tnt := c.GetHeader("X-Test-Tenant"); tnt != "" {
			c.Set("tenant_id", tnt)
		}
		c.Next()
	})

	g := router.Group("/api/v1/orders")
	{
		g.POST("/:id/items", h.AddOrderItem)
		g.DELETE("/:id/items/:itemId", h.RemoveOrderItem)
		g.POST("/:id/confirm", h.ConfirmOrder)
		g.POST("/:id/cancel", h.CancelOrder)
		g.POST("/:id/ship", h.ShipOrder)
		g.POST("/:id/deliver", h.DeliverOrder)
		g.POST("/:id/send-receipt", h.SendReceipt)
	}
	return router
}

func doJSON(router *gin.Engine, method, path, tenant, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var r *http.Request
	if body != "" {
		r, _ = http.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r, _ = http.NewRequest(method, path, nil)
	}
	if tenant != "" {
		r.Header.Set("X-Test-Tenant", tenant)
	}
	router.ServeHTTP(w, r)
	return w
}

// Cross-tenant mutations must return 404 (never 403) so order existence is not
// leaked to a caller from another tenant.
func TestCommand_CrossTenant_Returns404(t *testing.T) {
	store := newInMemoryEventStore()
	seedOrder(t, store, "order-1", "tenant-A")
	router := setupCommandRouter(store, &recordingPublisher{})

	cases := []struct {
		name, method, path, body string
	}{
		{"add item", "POST", "/api/v1/orders/order-1/items", `{"product_id":"p1","sku":"s1","name":"n","quantity":1,"unit_price":5}`},
		{"remove item", "DELETE", "/api/v1/orders/order-1/items/item-1", ""},
		{"confirm", "POST", "/api/v1/orders/order-1/confirm", `{"confirmed_by":"u"}`},
		{"cancel", "POST", "/api/v1/orders/order-1/cancel", `{"reason":"r","cancelled_by":"u"}`},
		{"ship", "POST", "/api/v1/orders/order-1/ship", `{"tracking_number":"t","carrier":"c","shipped_by":"u"}`},
		{"deliver", "POST", "/api/v1/orders/order-1/deliver", `{"received_by":"u"}`},
		{"send-receipt", "POST", "/api/v1/orders/order-1/send-receipt", `{"email":"attacker@evil.com"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Caller is tenant-B attacking tenant-A's order.
			w := doJSON(router, tc.method, tc.path, "tenant-B", tc.body)
			assert.Equal(t, http.StatusNotFound, w.Code, "cross-tenant %s should 404", tc.name)
		})
	}
}

// A cross-tenant send-receipt must not publish (i.e. must not email order data).
func TestCommand_SendReceipt_CrossTenant_DoesNotPublish(t *testing.T) {
	store := newInMemoryEventStore()
	seedOrder(t, store, "order-1", "tenant-A")
	pub := &recordingPublisher{}
	router := setupCommandRouter(store, pub)

	w := doJSON(router, "POST", "/api/v1/orders/order-1/send-receipt", "tenant-B", `{"email":"attacker@evil.com"}`)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.False(t, pub.called, "receipt must not be published for a cross-tenant order")
}

// Same-tenant callers still succeed.
func TestCommand_SameTenant_Succeeds(t *testing.T) {
	store := newInMemoryEventStore()
	seedOrder(t, store, "order-1", "tenant-A")
	pub := &recordingPublisher{}
	router := setupCommandRouter(store, pub)

	w := doJSON(router, "POST", "/api/v1/orders/order-1/items", "tenant-A",
		`{"product_id":"p1","sku":"s1","name":"n","quantity":1,"unit_price":5}`)
	assert.Equal(t, http.StatusOK, w.Code)

	w = doJSON(router, "POST", "/api/v1/orders/order-1/send-receipt", "tenant-A", `{"email":"customer@example.com"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, pub.called, "receipt should be published for the owning tenant")
}

// Unauthenticated callers are rejected before any state change.
func TestCommand_Unauthenticated_Returns401(t *testing.T) {
	store := newInMemoryEventStore()
	seedOrder(t, store, "order-1", "tenant-A")
	router := setupCommandRouter(store, &recordingPublisher{})

	w := doJSON(router, "POST", "/api/v1/orders/order-1/items", "",
		`{"product_id":"p1","sku":"s1","name":"n","quantity":1,"unit_price":5}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// A missing order is indistinguishable from a cross-tenant one: both 404.
func TestCommand_MissingOrder_Returns404(t *testing.T) {
	store := newInMemoryEventStore()
	router := setupCommandRouter(store, &recordingPublisher{})

	w := doJSON(router, "POST", "/api/v1/orders/does-not-exist/cancel", "tenant-A",
		`{"reason":"r","cancelled_by":"u"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
