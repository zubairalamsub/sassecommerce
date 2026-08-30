package projection

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"go.uber.org/zap"
)

var testTime = time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)

func base(orderID string, eventType events.EventType, version int) events.BaseEvent {
	return events.BaseEvent{
		ID:          "evt-1",
		AggregateID: orderID,
		EventType:   eventType,
		Timestamp:   testTime,
		Version:     version,
	}
}

// newProjection wires an OrderProjection to a mock driver. The constructor
// issues the DDL, so that expectation is set up here for every test.
func newProjection(t *testing.T) (*OrderProjection, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS order_read_model").
		WillReturnResult(sqlmock.NewResult(0, 0))

	p, err := NewOrderProjection(db, zap.NewNop())
	if err != nil {
		t.Fatalf("NewOrderProjection: %v", err)
	}

	return p, mock, func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
		_ = db.Close()
	}
}

func TestNewOrderProjectionCreatesReadModelTables(t *testing.T) {
	_, _, done := newProjection(t)
	done()
}

func TestNewOrderProjectionFailsWhenDDLFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = db.Close() }()

	ddlErr := errors.New("permission denied")
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS order_read_model").WillReturnError(ddlErr)

	p, err := NewOrderProjection(db, zap.NewNop())

	if p != nil {
		t.Errorf("projection = %v, want nil when the DDL fails", p)
	}
	if !errors.Is(err, ddlErr) {
		t.Errorf("error = %v, want it to wrap %v", err, ddlErr)
	}
}

func TestProjectOrderCreatedInsertsPendingRow(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectExec("INSERT INTO order_read_model").
		WithArgs(
			"order-1", "tenant_saajan", "cust-1", "pending", 0.0, "BDT",
			"1 St", "Dhaka", "DH", "1200", "BD",
			"2 St", "Dhaka", "DH", "1201", "BD",
			testTime, testTime, 1,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := p.Project(events.OrderCreated{
		BaseEvent:    base("order-1", events.OrderCreatedEvent, 1),
		TenantID:     "tenant_saajan",
		CustomerID:   "cust-1",
		TotalAmount:  0,
		Currency:     "BDT",
		ShippingAddr: events.Address{Street: "1 St", City: "Dhaka", State: "DH", PostalCode: "1200", Country: "BD"},
		BillingAddr:  events.Address{Street: "2 St", City: "Dhaka", State: "DH", PostalCode: "1201", Country: "BD"},
	})
	if err != nil {
		t.Fatalf("Project(OrderCreated) = %v", err)
	}
}

func TestProjectOrderCreatedSurfacesInsertError(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	insertErr := errors.New("unique violation")
	mock.ExpectExec("INSERT INTO order_read_model").WillReturnError(insertErr)

	err := p.Project(events.OrderCreated{BaseEvent: base("order-1", events.OrderCreatedEvent, 1)})

	if !errors.Is(err, insertErr) {
		t.Fatalf("Project = %v, want %v", err, insertErr)
	}
}

func TestProjectOrderItemAddedCommitsBothWrites(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO order_item_read_model").
		WithArgs("prod-1", "order-1", "prod-1", "", "SKU-1", "Shirt", 2, 500.0, 1000.0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE order_read_model").
		WithArgs("order-1", testTime, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := p.Project(events.OrderItemAdded{
		BaseEvent:  base("order-1", events.OrderItemAddedEvent, 2),
		ItemID:     "prod-1",
		ProductID:  "prod-1",
		SKU:        "SKU-1",
		Name:       "Shirt",
		Quantity:   2,
		UnitPrice:  500,
		TotalPrice: 1000,
	})
	if err != nil {
		t.Fatalf("Project(OrderItemAdded) = %v", err)
	}
}

func TestProjectOrderItemAddedRollsBackWhenItemInsertFails(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	insertErr := errors.New("item insert failed")
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO order_item_read_model").WillReturnError(insertErr)
	mock.ExpectRollback()

	err := p.Project(events.OrderItemAdded{BaseEvent: base("order-1", events.OrderItemAddedEvent, 2)})

	if !errors.Is(err, insertErr) {
		t.Fatalf("Project = %v, want %v", err, insertErr)
	}
}

// The item row and the recomputed order total must land together: if the total
// update fails, the already-inserted item has to be rolled back with it.
func TestProjectOrderItemAddedRollsBackWhenTotalUpdateFails(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	updateErr := errors.New("total update failed")
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO order_item_read_model").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE order_read_model").WillReturnError(updateErr)
	mock.ExpectRollback()

	err := p.Project(events.OrderItemAdded{BaseEvent: base("order-1", events.OrderItemAddedEvent, 2)})

	if !errors.Is(err, updateErr) {
		t.Fatalf("Project = %v, want %v", err, updateErr)
	}
}

func TestProjectOrderItemAddedSurfacesBeginError(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	beginErr := errors.New("no connection")
	mock.ExpectBegin().WillReturnError(beginErr)

	err := p.Project(events.OrderItemAdded{BaseEvent: base("order-1", events.OrderItemAddedEvent, 2)})

	if !errors.Is(err, beginErr) {
		t.Fatalf("Project = %v, want %v", err, beginErr)
	}
}

// The delete is scoped by both item id and order id so one order's event can
// never remove another order's item row.
func TestProjectOrderItemRemovedDeletesScopedByOrder(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM order_item_read_model").
		WithArgs("prod-1", "order-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE order_read_model").
		WithArgs("order-1", testTime, 3).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := p.Project(events.OrderItemRemoved{
		BaseEvent: base("order-1", events.OrderItemRemovedEvent, 3),
		ItemID:    "prod-1",
	})
	if err != nil {
		t.Fatalf("Project(OrderItemRemoved) = %v", err)
	}
}

func TestProjectOrderItemRemovedRollsBackWhenDeleteFails(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	deleteErr := errors.New("delete failed")
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM order_item_read_model").WillReturnError(deleteErr)
	mock.ExpectRollback()

	err := p.Project(events.OrderItemRemoved{BaseEvent: base("order-1", events.OrderItemRemovedEvent, 3)})

	if !errors.Is(err, deleteErr) {
		t.Fatalf("Project = %v, want %v", err, deleteErr)
	}
}

func TestProjectOrderItemRemovedRollsBackWhenTotalUpdateFails(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	updateErr := errors.New("total update failed")
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM order_item_read_model").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE order_read_model").WillReturnError(updateErr)
	mock.ExpectRollback()

	err := p.Project(events.OrderItemRemoved{BaseEvent: base("order-1", events.OrderItemRemovedEvent, 3)})

	if !errors.Is(err, updateErr) {
		t.Fatalf("Project = %v, want %v", err, updateErr)
	}
}

func TestProjectOrderItemRemovedSurfacesBeginError(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	beginErr := errors.New("no connection")
	mock.ExpectBegin().WillReturnError(beginErr)

	err := p.Project(events.OrderItemRemoved{BaseEvent: base("order-1", events.OrderItemRemovedEvent, 3)})

	if !errors.Is(err, beginErr) {
		t.Fatalf("Project = %v, want %v", err, beginErr)
	}
}

// Confirm/Cancel/Deliver all funnel through updateOrderStatus, so the status
// string each one writes is the thing worth pinning down.
func TestProjectStatusTransitions(t *testing.T) {
	tests := []struct {
		name       string
		event      events.Event
		wantStatus string
	}{
		{
			name:       "confirmed",
			event:      events.OrderConfirmed{BaseEvent: base("order-1", events.OrderConfirmedEvent, 4), ConfirmedBy: "staff-1"},
			wantStatus: "confirmed",
		},
		{
			name:       "cancelled",
			event:      events.OrderCancelled{BaseEvent: base("order-1", events.OrderCancelledEvent, 5), Reason: "fraud"},
			wantStatus: "cancelled",
		},
		{
			name:       "delivered",
			event:      events.OrderDelivered{BaseEvent: base("order-1", events.OrderDeliveredEvent, 7), ReceivedBy: "cust-1"},
			wantStatus: "delivered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, mock, done := newProjection(t)
			defer done()

			mock.ExpectExec("UPDATE order_read_model").
				WithArgs(tt.wantStatus, testTime, tt.event.GetVersion(), "order-1").
				WillReturnResult(sqlmock.NewResult(0, 1))

			if err := p.Project(tt.event); err != nil {
				t.Fatalf("Project(%s) = %v", tt.name, err)
			}
		})
	}
}

func TestProjectStatusTransitionSurfacesUpdateError(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	updateErr := errors.New("update failed")
	mock.ExpectExec("UPDATE order_read_model").WillReturnError(updateErr)

	err := p.Project(events.OrderConfirmed{BaseEvent: base("order-1", events.OrderConfirmedEvent, 4)})

	if !errors.Is(err, updateErr) {
		t.Fatalf("Project = %v, want %v", err, updateErr)
	}
}

func TestProjectOrderShippedStoresTracking(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectExec("UPDATE order_read_model").
		WithArgs("shipped", "TRK-9", "Pathao", testTime, 6, "order-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := p.Project(events.OrderShipped{
		BaseEvent:      base("order-1", events.OrderShippedEvent, 6),
		TrackingNumber: "TRK-9",
		Carrier:        "Pathao",
	})
	if err != nil {
		t.Fatalf("Project(OrderShipped) = %v", err)
	}
}

func TestProjectOrderShippedSurfacesUpdateError(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	updateErr := errors.New("update failed")
	mock.ExpectExec("UPDATE order_read_model").WillReturnError(updateErr)

	err := p.Project(events.OrderShipped{BaseEvent: base("order-1", events.OrderShippedEvent, 6)})

	if !errors.Is(err, updateErr) {
		t.Fatalf("Project = %v, want %v", err, updateErr)
	}
}

func TestProjectPaymentProcessedStoresPaymentID(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectExec("UPDATE order_read_model").
		WithArgs("pay-1", testTime, 8, "order-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := p.Project(events.PaymentProcessed{
		BaseEvent:     base("order-1", events.PaymentProcessedEvent, 8),
		PaymentID:     "pay-1",
		Amount:        1000,
		PaymentMethod: "card",
		TransactionID: "txn-1",
	})
	if err != nil {
		t.Fatalf("Project(PaymentProcessed) = %v", err)
	}
}

func TestProjectInventoryReservedStoresReservationID(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectExec("UPDATE order_read_model").
		WithArgs("res-1", testTime, 9, "order-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := p.Project(events.InventoryReserved{
		BaseEvent:     base("order-1", events.InventoryReservedEvent, 9),
		ReservationID: "res-1",
		Items:         []events.ReservedItem{{ProductID: "prod-1", Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("Project(InventoryReserved) = %v", err)
	}
}

// Releasing must clear the reservation id, not overwrite it with the released
// one — a stale id would make the order look like it still holds stock.
func TestProjectInventoryReleasedClearsReservationID(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectExec("UPDATE order_read_model").
		WithArgs(testTime, 10, "order-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := p.Project(events.InventoryReleased{
		BaseEvent:     base("order-1", events.InventoryReleasedEvent, 10),
		ReservationID: "res-1",
		Reason:        "payment failed",
	})
	if err != nil {
		t.Fatalf("Project(InventoryReleased) = %v", err)
	}
}

func TestProjectInventoryReleasedSurfacesUpdateError(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	updateErr := errors.New("update failed")
	mock.ExpectExec("UPDATE order_read_model").WillReturnError(updateErr)

	err := p.Project(events.InventoryReleased{BaseEvent: base("order-1", events.InventoryReleasedEvent, 10)})

	if !errors.Is(err, updateErr) {
		t.Fatalf("Project = %v, want %v", err, updateErr)
	}
}

// PaymentFailed and OrderRefunded carry no read-model state of their own, so
// the projection deliberately ignores them rather than erroring. Asserting no
// SQL runs keeps that a decision rather than an accident.
func TestProjectIgnoresEventsWithNoReadModelEffect(t *testing.T) {
	ignored := []events.Event{
		events.PaymentFailed{BaseEvent: base("order-1", events.PaymentFailedEvent, 11), PaymentID: "pay-1", Reason: "declined"},
		events.OrderRefunded{BaseEvent: base("order-1", events.OrderRefundedEvent, 12), PaymentID: "pay-1", Amount: 1000},
	}

	for _, event := range ignored {
		t.Run(string(event.GetEventType()), func(t *testing.T) {
			p, _, done := newProjection(t)
			defer done()

			// No further expectations: any SQL issued here fails the test.
			if err := p.Project(event); err != nil {
				t.Fatalf("Project(%T) = %v, want nil", event, err)
			}
		})
	}
}

func orderRow() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "tenant_id", "customer_id", "status", "total_amount", "currency",
		"shipping_street", "shipping_city", "shipping_state", "shipping_postal_code", "shipping_country",
		"billing_street", "billing_city", "billing_state", "billing_postal_code", "billing_country",
		"payment_id", "reservation_id", "tracking_number", "carrier",
		"created_at", "updated_at", "version",
	}).AddRow(
		"order-1", "tenant_saajan", "cust-1", "confirmed", 1000.0, "BDT",
		"1 St", "Dhaka", "DH", "1200", "BD",
		"2 St", "Dhaka", "DH", "1201", "BD",
		"pay-1", "res-1", "TRK-9", "Pathao",
		testTime, testTime, 6,
	)
}

func TestGetOrderMapsAddressesAndFields(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectQuery("SELECT id, tenant_id, customer_id").
		WithArgs("order-1").
		WillReturnRows(orderRow())

	order, err := p.GetOrder("order-1")
	if err != nil {
		t.Fatalf("GetOrder = %v", err)
	}

	if order.ID != "order-1" || order.TenantID != "tenant_saajan" || order.Status != "confirmed" {
		t.Errorf("order = %+v, want the seeded identity fields", order)
	}
	if order.TotalAmount != 1000 || order.Currency != "BDT" || order.Version != 6 {
		t.Errorf("order = %+v, want total 1000 BDT at version 6", order)
	}
	// The two addresses are scanned into locals and assigned afterwards, so a
	// swap between them would be invisible without this.
	if order.ShippingAddress.Street != "1 St" || order.ShippingAddress.PostalCode != "1200" {
		t.Errorf("shipping address = %+v, want the shipping columns", order.ShippingAddress)
	}
	if order.BillingAddress.Street != "2 St" || order.BillingAddress.PostalCode != "1201" {
		t.Errorf("billing address = %+v, want the billing columns", order.BillingAddress)
	}
	if order.PaymentID != "pay-1" || order.ReservationID != "res-1" || order.TrackingNumber != "TRK-9" || order.Carrier != "Pathao" {
		t.Errorf("order = %+v, want the saga/shipping columns populated", order)
	}
}

func TestGetOrderNotFound(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectQuery("SELECT id, tenant_id, customer_id").
		WithArgs("ghost").
		WillReturnError(sql.ErrNoRows)

	order, err := p.GetOrder("ghost")

	if order != nil {
		t.Errorf("order = %+v, want nil", order)
	}
	if err == nil || err.Error() != "order not found" {
		t.Fatalf("err = %v, want order not found", err)
	}
}

func TestGetOrderSurfacesOtherErrors(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	queryErr := errors.New("connection reset")
	mock.ExpectQuery("SELECT id, tenant_id, customer_id").WillReturnError(queryErr)

	_, err := p.GetOrder("order-1")

	if !errors.Is(err, queryErr) {
		t.Fatalf("err = %v, want %v", err, queryErr)
	}
}

func TestGetOrderItems(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	rows := sqlmock.NewRows([]string{"id", "order_id", "product_id", "variant_id", "sku", "name", "quantity", "unit_price", "total_price"}).
		AddRow("prod-1", "order-1", "prod-1", "", "SKU-1", "Shirt", 2, 500.0, 1000.0).
		AddRow("prod-2-red", "order-1", "prod-2", "red", "SKU-2", "Scarf", 1, 250.0, 250.0)

	mock.ExpectQuery("FROM order_item_read_model").WithArgs("order-1").WillReturnRows(rows)

	items, err := p.GetOrderItems("order-1")
	if err != nil {
		t.Fatalf("GetOrderItems = %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].SKU != "SKU-1" || items[0].Quantity != 2 || items[0].TotalPrice != 1000 {
		t.Errorf("items[0] = %+v", items[0])
	}
	if items[1].VariantID != "red" {
		t.Errorf("items[1].VariantID = %q, want red", items[1].VariantID)
	}
}

func TestGetOrderItemsEmptyIsNotNil(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectQuery("FROM order_item_read_model").
		WithArgs("order-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "product_id", "variant_id", "sku", "name", "quantity", "unit_price", "total_price"}))

	items, err := p.GetOrderItems("order-1")
	if err != nil {
		t.Fatalf("GetOrderItems = %v", err)
	}
	// The handler JSON-encodes this directly; nil would serialise as null
	// instead of [].
	if items == nil {
		t.Fatal("items = nil, want an empty slice")
	}
	if len(items) != 0 {
		t.Errorf("got %d items, want 0", len(items))
	}
}

func TestGetOrderItemsSurfacesQueryError(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	queryErr := errors.New("query failed")
	mock.ExpectQuery("FROM order_item_read_model").WillReturnError(queryErr)

	_, err := p.GetOrderItems("order-1")

	if !errors.Is(err, queryErr) {
		t.Fatalf("err = %v, want %v", err, queryErr)
	}
}

func TestGetOrderItemsSurfacesRowError(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	rowErr := errors.New("row stream broke")
	rows := sqlmock.NewRows([]string{"id", "order_id", "product_id", "variant_id", "sku", "name", "quantity", "unit_price", "total_price"}).
		AddRow("prod-1", "order-1", "prod-1", "", "SKU-1", "Shirt", 2, 500.0, 1000.0).
		RowError(0, rowErr)

	mock.ExpectQuery("FROM order_item_read_model").WillReturnRows(rows)

	_, err := p.GetOrderItems("order-1")

	if !errors.Is(err, rowErr) {
		t.Fatalf("err = %v, want %v", err, rowErr)
	}
}

func summaryRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "customer_id", "status", "total_amount", "currency", "created_at", "updated_at", "item_count"}).
		AddRow("order-1", "cust-1", "confirmed", 1000.0, "BDT", testTime, testTime, 2)
}

// The tenant id is the isolation boundary: it must reach the query as the
// first bind parameter, ahead of the customer filter and the page bounds.
func TestGetOrdersByCustomerIsTenantScoped(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectQuery("WITH page AS").
		WithArgs("tenant_saajan", "cust-1", 20, 40).
		WillReturnRows(summaryRows())

	summaries, err := p.GetOrdersByCustomer("tenant_saajan", "cust-1", 20, 40)
	if err != nil {
		t.Fatalf("GetOrdersByCustomer = %v", err)
	}

	if len(summaries) != 1 {
		t.Fatalf("got %d summaries, want 1", len(summaries))
	}
	if summaries[0].ID != "order-1" || summaries[0].ItemCount != 2 || summaries[0].TotalAmount != 1000 {
		t.Errorf("summary = %+v", summaries[0])
	}
}

func TestGetOrdersByCustomerSurfacesQueryError(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	queryErr := errors.New("query failed")
	mock.ExpectQuery("WITH page AS").WillReturnError(queryErr)

	_, err := p.GetOrdersByCustomer("tenant_saajan", "cust-1", 20, 0)

	if !errors.Is(err, queryErr) {
		t.Fatalf("err = %v, want %v", err, queryErr)
	}
}

func TestGetOrdersByTenant(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectQuery("WITH page AS").
		WithArgs("tenant_saajan", 10, 0).
		WillReturnRows(summaryRows())

	summaries, err := p.GetOrdersByTenant("tenant_saajan", 10, 0)
	if err != nil {
		t.Fatalf("GetOrdersByTenant = %v", err)
	}

	if len(summaries) != 1 || summaries[0].CustomerID != "cust-1" {
		t.Errorf("summaries = %+v, want the one seeded order", summaries)
	}
}

func TestGetOrdersByTenantSurfacesQueryError(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	queryErr := errors.New("query failed")
	mock.ExpectQuery("WITH page AS").WillReturnError(queryErr)

	_, err := p.GetOrdersByTenant("tenant_saajan", 10, 0)

	if !errors.Is(err, queryErr) {
		t.Fatalf("err = %v, want %v", err, queryErr)
	}
}

func TestGetOrdersEmptyIsNotNil(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	mock.ExpectQuery("WITH page AS").
		WillReturnRows(sqlmock.NewRows([]string{"id", "customer_id", "status", "total_amount", "currency", "created_at", "updated_at", "item_count"}))

	summaries, err := p.GetOrdersByTenant("tenant_saajan", 10, 0)
	if err != nil {
		t.Fatalf("GetOrdersByTenant = %v", err)
	}
	if summaries == nil {
		t.Fatal("summaries = nil, want an empty slice")
	}
	if len(summaries) != 0 {
		t.Errorf("got %d summaries, want 0", len(summaries))
	}
}

func TestScanOrderSummariesSurfacesRowError(t *testing.T) {
	p, mock, done := newProjection(t)
	defer done()

	rowErr := errors.New("row stream broke")
	rows := summaryRows().RowError(0, rowErr)
	mock.ExpectQuery("WITH page AS").WillReturnRows(rows)

	_, err := p.GetOrdersByTenant("tenant_saajan", 10, 0)

	if !errors.Is(err, rowErr) {
		t.Fatalf("err = %v, want %v", err, rowErr)
	}
}
