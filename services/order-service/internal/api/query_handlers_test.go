package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/yourusername/ecommerce/order-service/internal/projection"
	"go.uber.org/zap"
)

var queryTestTime = time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)

// setupQueryRouter wires the query handler to a mock-backed projection and the
// in-memory event store, with the shared Auth middleware stubbed by a header.
func setupQueryRouter(t *testing.T, store *inMemoryEventStore) (*gin.Engine, sqlmock.Sqlmock, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS order_read_model").WillReturnResult(sqlmock.NewResult(0, 0))

	proj, err := projection.NewOrderProjection(db, zap.NewNop())
	if err != nil {
		t.Fatalf("NewOrderProjection: %v", err)
	}

	h := NewQueryHandler(proj, store, zap.NewNop())

	router := gin.New()
	router.Use(func(c *gin.Context) {
		if tnt := c.GetHeader("X-Test-Tenant"); tnt != "" {
			c.Set("tenant_id", tnt)
		}
		c.Next()
	})
	router.GET("/orders/:id", h.GetOrder)
	router.GET("/customers/:customerId/orders", h.GetOrdersByCustomer)
	router.GET("/tenants/:tenantId/orders", h.GetOrdersByTenant)

	return router, mock, func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
		_ = db.Close()
	}
}

func doGet(router *gin.Engine, path, tenant string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if tenant != "" {
		req.Header.Set("X-Test-Tenant", tenant)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	return body
}

func queryOrderRow() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "tenant_id", "customer_id", "status", "total_amount", "currency",
		"shipping_street", "shipping_city", "shipping_state", "shipping_postal_code", "shipping_country",
		"billing_street", "billing_city", "billing_state", "billing_postal_code", "billing_country",
		"payment_id", "reservation_id", "tracking_number", "carrier",
		"created_at", "updated_at", "version",
	}).AddRow(
		"order-1", "tenant_saajan", "cust-1", "confirmed", 1000.0, "BDT",
		"1 St", "Dhaka", "DH", "1200", "BD",
		"1 St", "Dhaka", "DH", "1200", "BD",
		"pay-1", "res-1", "TRK-9", "Pathao",
		queryTestTime, queryTestTime, 6,
	)
}

func queryItemRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "order_id", "product_id", "variant_id", "sku", "name", "quantity", "unit_price", "total_price"}).
		AddRow("prod-1", "order-1", "prod-1", "", "SKU-1", "Shirt", 2, 500.0, 1000.0)
}

func TestGetOrderServesTheReadModel(t *testing.T) {
	router, mock, done := setupQueryRouter(t, newInMemoryEventStore())
	defer done()

	mock.ExpectQuery("SELECT id, tenant_id, customer_id").WithArgs("order-1").WillReturnRows(queryOrderRow())
	mock.ExpectQuery("FROM order_item_read_model").WithArgs("order-1").WillReturnRows(queryItemRows())

	w := doGet(router, "/orders/order-1", "tenant_saajan")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	order, ok := body["order"].(map[string]interface{})
	if !ok {
		t.Fatalf("body has no order object: %s", w.Body.String())
	}
	if order["id"] != "order-1" || order["status"] != "confirmed" {
		t.Errorf("order = %+v", order)
	}
	items, ok := body["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("items = %v, want one item", body["items"])
	}
}

// A projection miss must not 404 while the event stream still has the order —
// the read model is a cache, the event store is the truth.
func TestGetOrderRebuildsFromEventsOnProjectionMiss(t *testing.T) {
	store := newInMemoryEventStore()
	router, mock, done := setupQueryRouter(t, store)
	defer done()
	seedOrder(t, store, "order-1", "tenant_saajan")

	mock.ExpectQuery("SELECT id, tenant_id, customer_id").WithArgs("order-1").WillReturnError(sql.ErrNoRows)
	// The rebuild re-projects the stream to repair the read model.
	mock.ExpectExec("INSERT INTO order_read_model").WillReturnResult(sqlmock.NewResult(1, 1))

	w := doGet(router, "/orders/order-1", "tenant_saajan")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	order, ok := decodeBody(t, w)["order"].(map[string]interface{})
	if !ok {
		t.Fatalf("body has no order object: %s", w.Body.String())
	}
	if order["id"] != "order-1" || order["tenant_id"] != "tenant_saajan" {
		t.Errorf("rebuilt order = %+v, want it reconstructed from the stream", order)
	}
}

func TestGetOrderNotFoundWhenNeitherStoreHasIt(t *testing.T) {
	router, mock, done := setupQueryRouter(t, newInMemoryEventStore())
	defer done()

	mock.ExpectQuery("SELECT id, tenant_id, customer_id").WithArgs("ghost").WillReturnError(sql.ErrNoRows)

	w := doGet(router, "/orders/ghost", "tenant_saajan")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %s)", w.Code, w.Body.String())
	}
	if got := decodeBody(t, w)["error"]; got != "order_not_found" {
		t.Errorf("error = %v, want order_not_found", got)
	}
}

// A failure fetching items degrades to an empty list rather than failing the
// whole read; the rebuild then gets a chance to fill it in.
func TestGetOrderToleratesItemLookupFailure(t *testing.T) {
	store := newInMemoryEventStore()
	router, mock, done := setupQueryRouter(t, store)
	defer done()

	mock.ExpectQuery("SELECT id, tenant_id, customer_id").WithArgs("order-1").WillReturnRows(queryOrderRow())
	mock.ExpectQuery("FROM order_item_read_model").WillReturnError(errors.New("items table locked"))
	// Empty items triggers a rebuild attempt; the stream is empty, so it is a no-op.

	w := doGet(router, "/orders/order-1", "tenant_saajan")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	items, ok := decodeBody(t, w)["items"].([]interface{})
	if !ok {
		t.Fatalf("items = %v, want an empty array not null", decodeBody(t, w)["items"])
	}
	if len(items) != 0 {
		t.Errorf("items = %v, want empty", items)
	}
}

func TestGetOrdersByCustomerRequiresAuthentication(t *testing.T) {
	router, _, done := setupQueryRouter(t, newInMemoryEventStore())
	defer done()

	w := doGet(router, "/customers/cust-1/orders", "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body %s)", w.Code, w.Body.String())
	}
}

// The tenant reaching the query must be the JWT's, so a customer id belonging
// to another tenant returns nothing instead of that tenant's orders.
func TestGetOrdersByCustomerScopesToTheJWTTenant(t *testing.T) {
	router, mock, done := setupQueryRouter(t, newInMemoryEventStore())
	defer done()

	rows := sqlmock.NewRows([]string{"id", "customer_id", "status", "total_amount", "currency", "created_at", "updated_at", "item_count"}).
		AddRow("order-1", "cust-1", "confirmed", 1000.0, "BDT", queryTestTime, queryTestTime, 2)
	mock.ExpectQuery("WITH page AS").
		WithArgs("tenant_saajan", "cust-1", 10, 0).
		WillReturnRows(rows)

	w := doGet(router, "/customers/cust-1/orders", "tenant_saajan")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	orders, ok := body["orders"].([]interface{})
	if !ok || len(orders) != 1 {
		t.Fatalf("orders = %v, want one summary", body["orders"])
	}
	pagination, ok := body["pagination"].(map[string]interface{})
	if !ok {
		t.Fatalf("body has no pagination object: %s", w.Body.String())
	}
	if pagination["limit"] != float64(10) || pagination["offset"] != float64(0) || pagination["count"] != float64(1) {
		t.Errorf("pagination = %+v, want the defaults and a count of 1", pagination)
	}
}

func TestGetOrdersByCustomerHonoursPagination(t *testing.T) {
	router, mock, done := setupQueryRouter(t, newInMemoryEventStore())
	defer done()

	mock.ExpectQuery("WITH page AS").
		WithArgs("tenant_saajan", "cust-1", 25, 50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "customer_id", "status", "total_amount", "currency", "created_at", "updated_at", "item_count"}))

	w := doGet(router, "/customers/cust-1/orders?limit=25&offset=50", "tenant_saajan")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
}

// A non-numeric limit falls back to the default rather than erroring, so a
// junk query string cannot take the endpoint down.
func TestGetOrdersByCustomerFallsBackOnJunkPagination(t *testing.T) {
	router, mock, done := setupQueryRouter(t, newInMemoryEventStore())
	defer done()

	mock.ExpectQuery("WITH page AS").
		WithArgs("tenant_saajan", "cust-1", 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "customer_id", "status", "total_amount", "currency", "created_at", "updated_at", "item_count"}))

	w := doGet(router, "/customers/cust-1/orders?limit=lots&offset=", "tenant_saajan")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
}

func TestGetOrdersByCustomerSurfacesQueryFailure(t *testing.T) {
	router, mock, done := setupQueryRouter(t, newInMemoryEventStore())
	defer done()

	mock.ExpectQuery("WITH page AS").WillReturnError(errors.New("connection reset"))

	w := doGet(router, "/customers/cust-1/orders", "tenant_saajan")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body %s)", w.Code, w.Body.String())
	}
	if got := decodeBody(t, w)["error"]; got != "failed_to_get_orders" {
		t.Errorf("error = %v, want failed_to_get_orders", got)
	}
}

func TestGetOrdersByTenantRequiresAuthentication(t *testing.T) {
	router, _, done := setupQueryRouter(t, newInMemoryEventStore())
	defer done()

	w := doGet(router, "/tenants/tenant_saajan/orders", "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body %s)", w.Code, w.Body.String())
	}
}

// The :tenantId segment is client-controlled. Pointing it at another tenant
// must be refused outright, not quietly answered with the caller's own data.
func TestGetOrdersByTenantRejectsAnotherTenantInThePath(t *testing.T) {
	router, _, done := setupQueryRouter(t, newInMemoryEventStore())
	defer done()

	w := doGet(router, "/tenants/tenant_rival/orders", "tenant_saajan")

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body %s)", w.Code, w.Body.String())
	}
	if got := decodeBody(t, w)["error"]; got != "forbidden" {
		t.Errorf("error = %v, want forbidden", got)
	}
}

func TestGetOrdersByTenantServesTheCallersOwnTenant(t *testing.T) {
	router, mock, done := setupQueryRouter(t, newInMemoryEventStore())
	defer done()

	rows := sqlmock.NewRows([]string{"id", "customer_id", "status", "total_amount", "currency", "created_at", "updated_at", "item_count"}).
		AddRow("order-1", "cust-1", "confirmed", 1000.0, "BDT", queryTestTime, queryTestTime, 2)
	mock.ExpectQuery("WITH page AS").WithArgs("tenant_saajan", 10, 0).WillReturnRows(rows)

	w := doGet(router, "/tenants/tenant_saajan/orders", "tenant_saajan")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	orders, ok := decodeBody(t, w)["orders"].([]interface{})
	if !ok || len(orders) != 1 {
		t.Fatalf("orders = %v, want one summary", decodeBody(t, w)["orders"])
	}
}

func TestGetOrdersByTenantSurfacesQueryFailure(t *testing.T) {
	router, mock, done := setupQueryRouter(t, newInMemoryEventStore())
	defer done()

	mock.ExpectQuery("WITH page AS").WillReturnError(errors.New("connection reset"))

	w := doGet(router, "/tenants/tenant_saajan/orders", "tenant_saajan")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body %s)", w.Code, w.Body.String())
	}
}
