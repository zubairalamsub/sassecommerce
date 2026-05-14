package messaging

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/ecommerce/shipping-service/internal/models"
	"github.com/sirupsen/logrus"
)

// fakeShippingService captures CreateShipment / CancelShipment calls and
// reports a configurable existing-shipment lookup so the consumer's idempotency
// branch is testable.
type fakeShippingService struct {
	created   []*models.CreateShipmentRequest
	cancelled []string

	// existingByOrder simulates the GetShipmentByOrderID lookup.
	existingByOrder map[string]*models.ShipmentResponse
	// errOnGet simulates a "not found" error for GetShipmentByOrderID.
	errOnGet bool
}

func (f *fakeShippingService) CreateShipment(ctx context.Context, req *models.CreateShipmentRequest) (*models.ShipmentResponse, error) {
	f.created = append(f.created, req)
	return &models.ShipmentResponse{ID: "ship-" + req.OrderID, OrderID: req.OrderID, TrackingNumber: "TRK1"}, nil
}
func (f *fakeShippingService) GetShipment(ctx context.Context, id string) (*models.ShipmentResponse, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeShippingService) GetShipmentByTracking(ctx context.Context, trackingNumber string) (*models.ShipmentResponse, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeShippingService) GetShipmentByOrderID(ctx context.Context, tenantID, orderID string) (*models.ShipmentResponse, error) {
	if f.errOnGet {
		return nil, errors.New("shipment not found")
	}
	if s, ok := f.existingByOrder[orderID]; ok {
		return s, nil
	}
	return nil, errors.New("shipment not found")
}
func (f *fakeShippingService) ListShipments(ctx context.Context, tenantID string, page, pageSize int, status string) ([]models.ShipmentResponse, int64, error) {
	return nil, 0, nil
}
func (f *fakeShippingService) UpdateStatus(ctx context.Context, id string, req *models.UpdateStatusRequest) (*models.ShipmentResponse, error) {
	return nil, nil
}
func (f *fakeShippingService) CancelShipment(ctx context.Context, id string) (*models.ShipmentResponse, error) {
	f.cancelled = append(f.cancelled, id)
	return &models.ShipmentResponse{ID: id}, nil
}
func (f *fakeShippingService) CalculateRates(ctx context.Context, req *models.CalculateRateRequest) (*models.RateCalculationResponse, error) {
	return nil, nil
}

func newTestConsumer(svc *fakeShippingService) *EventConsumer {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return &EventConsumer{
		service:        svc,
		logger:         logger,
		stop:           make(chan struct{}),
		defaultCarrier: "pathao",
		defaultFromAddr: models.AddressRequest{
			Name: "WH", Street: "1 Main", City: "Dhaka", State: "Dhaka",
			PostalCode: "1212", Country: "BD",
		},
	}
}

func validShippingAddress() map[string]interface{} {
	return map[string]interface{}{
		"name":        "Asha Khan",
		"street":      "House 7, Road 3",
		"city":        "Dhaka",
		"state":       "Dhaka",
		"postal_code": "1207",
		"country":     "BD",
	}
}

func TestHandleEvent_OrderConfirmed_CreatesShipment(t *testing.T) {
	svc := &fakeShippingService{errOnGet: true}
	c := newTestConsumer(svc)

	env := &EventEnvelope{
		EventType: "OrderConfirmed",
		Payload: map[string]interface{}{
			"tenant_id":        "tenant-1",
			"order_id":         "order-1",
			"shipping_address": validShippingAddress(),
			"items": []interface{}{
				map[string]interface{}{
					"product_id": "p1",
					"quantity":   float64(2),
					"name":       "T-Shirt",
				},
			},
		},
	}

	if err := c.HandleEvent(context.Background(), env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.created) != 1 {
		t.Fatalf("expected 1 shipment created, got %d", len(svc.created))
	}
	got := svc.created[0]
	if got.OrderID != "order-1" || got.TenantID != "tenant-1" {
		t.Errorf("wrong order/tenant: %+v", got)
	}
	if got.Carrier != "pathao" {
		t.Errorf("expected default carrier pathao, got %s", got.Carrier)
	}
	if got.ToAddress.City != "Dhaka" || got.ToAddress.PostalCode != "1207" {
		t.Errorf("address not extracted: %+v", got.ToAddress)
	}
	if len(got.Items) != 1 || got.Items[0].Quantity != 2 {
		t.Errorf("items not extracted: %+v", got.Items)
	}
}

func TestHandleEvent_OrderConfirmed_IsIdempotent(t *testing.T) {
	svc := &fakeShippingService{
		existingByOrder: map[string]*models.ShipmentResponse{
			"order-1": {ID: "ship-existing", OrderID: "order-1"},
		},
	}
	c := newTestConsumer(svc)

	env := &EventEnvelope{
		EventType: "OrderConfirmed",
		Payload: map[string]interface{}{
			"tenant_id":        "tenant-1",
			"order_id":         "order-1",
			"shipping_address": validShippingAddress(),
		},
	}
	if err := c.HandleEvent(context.Background(), env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.created) != 0 {
		t.Errorf("expected no new shipment, got %d", len(svc.created))
	}
}

func TestHandleEvent_PaymentCompleted_CreatesShipment(t *testing.T) {
	svc := &fakeShippingService{errOnGet: true}
	c := newTestConsumer(svc)
	env := &EventEnvelope{
		EventType: "PaymentCompleted",
		Payload: map[string]interface{}{
			"tenant_id":        "tenant-1",
			"order_id":         "order-x",
			"shipping_address": validShippingAddress(),
		},
	}
	if err := c.HandleEvent(context.Background(), env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.created) != 1 {
		t.Fatalf("expected 1 shipment, got %d", len(svc.created))
	}
}

func TestHandleEvent_OrderConfirmed_SkipsWithoutAddress(t *testing.T) {
	svc := &fakeShippingService{errOnGet: true}
	c := newTestConsumer(svc)
	env := &EventEnvelope{
		EventType: "OrderConfirmed",
		Payload: map[string]interface{}{
			"tenant_id": "tenant-1",
			"order_id":  "order-1",
			// shipping_address absent
		},
	}
	if err := c.HandleEvent(context.Background(), env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.created) != 0 {
		t.Errorf("expected no shipment created without address, got %d", len(svc.created))
	}
}

func TestHandleEvent_OrderCancelled_CancelsPendingShipment(t *testing.T) {
	svc := &fakeShippingService{
		existingByOrder: map[string]*models.ShipmentResponse{
			"order-1": {ID: "ship-1", OrderID: "order-1", Status: models.StatusPending},
		},
	}
	c := newTestConsumer(svc)

	env := &EventEnvelope{
		EventType: "OrderCancelled",
		Payload:   map[string]interface{}{"tenant_id": "t1", "order_id": "order-1"},
	}
	if err := c.HandleEvent(context.Background(), env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.cancelled) != 1 || svc.cancelled[0] != "ship-1" {
		t.Errorf("expected ship-1 cancelled, got %+v", svc.cancelled)
	}
}

func TestHandleEvent_OrderCancelled_LeavesInTransitAlone(t *testing.T) {
	svc := &fakeShippingService{
		existingByOrder: map[string]*models.ShipmentResponse{
			"order-1": {ID: "ship-1", OrderID: "order-1", Status: models.StatusInTransit},
		},
	}
	c := newTestConsumer(svc)
	env := &EventEnvelope{
		EventType: "OrderCancelled",
		Payload:   map[string]interface{}{"tenant_id": "t1", "order_id": "order-1"},
	}
	if err := c.HandleEvent(context.Background(), env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.cancelled) != 0 {
		t.Errorf("expected no cancellation for in-transit shipment, got %+v", svc.cancelled)
	}
}

func TestHandleEvent_UnknownEventTypeIgnored(t *testing.T) {
	svc := &fakeShippingService{}
	c := newTestConsumer(svc)
	env := &EventEnvelope{
		EventType: "InventoryRestocked",
		Payload:   map[string]interface{}{"tenant_id": "t", "order_id": "o"},
	}
	if err := c.HandleEvent(context.Background(), env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.created)+len(svc.cancelled) != 0 {
		t.Errorf("expected no action, got created=%d cancelled=%d", len(svc.created), len(svc.cancelled))
	}
}

func TestExtractAddress_DefaultsCountryToBD(t *testing.T) {
	a := extractAddress(map[string]interface{}{
		"name":        "X",
		"street":      "1 Y",
		"city":        "Dhaka",
		"state":       "Dhaka",
		"postal_code": "1200",
	})
	if a.Country != "BD" {
		t.Errorf("expected default country BD, got %s", a.Country)
	}
}
