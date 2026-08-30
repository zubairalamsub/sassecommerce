package messaging

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

// prefStubService overrides GetPreference so the email-fallback path can be
// driven both ways; everything else comes from fakeNotificationService.
type prefStubService struct {
	*fakeNotificationService
	pref *models.UserPreferenceResponse
}

func (s *prefStubService) GetPreference(ctx context.Context, tenantID, userID string) (*models.UserPreferenceResponse, error) {
	if s.pref == nil {
		return nil, errors.New("no preference")
	}
	return s.pref, nil
}

func newConsumerWithRepo(svc *fakeNotificationService, repo *mocks.MockNotificationRepository) *EventConsumer {
	logger := logrus.New()
	logger.SetOutput(discardWriter{})
	return &EventConsumer{
		service:         svc,
		repo:            repo,
		logger:          logger,
		stop:            make(chan struct{}),
		frontendBaseURL: "https://shop.example.com",
	}
}

// ------------------------------------------------------- handleUserRegistered

func TestHandleUserRegistered_SendsWelcome(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handleUserRegistered(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1",
		"user_id":   "user-1",
		"email":     "alice@example.com",
		"name":      "Alice",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(svc.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.sent))
	}
	got := svc.sent[0]
	if got.Type != string(models.TypeWelcome) {
		t.Errorf("type = %s, want welcome", got.Type)
	}
	if got.Recipient != "alice@example.com" {
		t.Errorf("recipient = %s, want alice@example.com", got.Recipient)
	}
	if !strings.Contains(got.Body, "Alice") {
		t.Errorf("body should greet the user by name, got %q", got.Body)
	}
	if got.ReferenceType != "user" || got.ReferenceID != "user-1" {
		t.Errorf("reference = %s/%s, want user/user-1", got.ReferenceType, got.ReferenceID)
	}
}

func TestHandleUserRegistered_SkipsOnMissingFields(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]interface{}
	}{
		{"missing tenant", map[string]interface{}{"user_id": "u", "email": "e@x.com"}},
		{"missing user", map[string]interface{}{"tenant_id": "t", "email": "e@x.com"}},
		{"missing email", map[string]interface{}{"tenant_id": "t", "user_id": "u"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeNotificationService{}
			c := newTestConsumer(svc)
			if err := c.handleUserRegistered(context.Background(), tc.payload); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(svc.sent) != 0 {
				t.Errorf("expected no notification, got %d", len(svc.sent))
			}
		})
	}
}

// A send failure has to reach the caller so the message is not committed as
// successfully processed.
func TestHandleUserRegistered_PropagatesSendFailure(t *testing.T) {
	svc := &fakeNotificationService{err: errors.New("smtp down")}
	c := newTestConsumer(svc)

	err := c.handleUserRegistered(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "user_id": "user-1", "email": "alice@example.com",
	})

	if err == nil {
		t.Fatal("expected the send failure to propagate")
	}
}

// ---------------------------------------------------------- handleOrderPlaced

func TestHandleOrderPlaced_SendsConfirmation(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handleOrderPlaced(context.Background(), map[string]interface{}{
		"tenant_id":   "tenant-1",
		"customer_id": "cust-1",
		"order_id":    "order-1",
		"email":       "alice@example.com",
		"total":       1250.5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(svc.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.sent))
	}
	got := svc.sent[0]
	if got.Type != string(models.TypeOrderConfirmation) {
		t.Errorf("type = %s, want order_confirmation", got.Type)
	}
	if !strings.Contains(got.Subject, "order-1") {
		t.Errorf("subject should carry the order id, got %q", got.Subject)
	}
	if got.ReferenceType != "order" || got.ReferenceID != "order-1" {
		t.Errorf("reference = %s/%s, want order/order-1", got.ReferenceType, got.ReferenceID)
	}
}

// With no email on the event, the recipient comes from the stored preference.
func TestHandleOrderPlaced_FallsBackToStoredEmail(t *testing.T) {
	svc := &prefStubService{
		fakeNotificationService: &fakeNotificationService{},
		pref:                    &models.UserPreferenceResponse{Email: "stored@example.com"},
	}
	logger := logrus.New()
	logger.SetOutput(discardWriter{})
	c := &EventConsumer{service: svc, logger: logger, stop: make(chan struct{}), frontendBaseURL: "https://shop.example.com"}

	err := c.handleOrderPlaced(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "customer_id": "cust-1", "order_id": "order-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(svc.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.sent))
	}
	if svc.sent[0].Recipient != "stored@example.com" {
		t.Errorf("recipient = %s, want the stored preference email", svc.sent[0].Recipient)
	}
}

// Without a preference either, a placeholder keeps the record traceable rather
// than sending to an empty address.
func TestHandleOrderPlaced_PlaceholderWhenNoEmailAnywhere(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handleOrderPlaced(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "customer_id": "cust-1", "order_id": "order-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if svc.sent[0].Recipient != "cust-1@placeholder.local" {
		t.Errorf("recipient = %s, want the placeholder address", svc.sent[0].Recipient)
	}
}

func TestHandleOrderPlaced_SkipsOnMissingFields(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]interface{}
	}{
		{"missing tenant", map[string]interface{}{"customer_id": "c", "order_id": "o"}},
		{"missing customer", map[string]interface{}{"tenant_id": "t", "order_id": "o"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeNotificationService{}
			c := newTestConsumer(svc)
			if err := c.handleOrderPlaced(context.Background(), tc.payload); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(svc.sent) != 0 {
				t.Errorf("expected no notification, got %d", len(svc.sent))
			}
		})
	}
}

// --------------------------------------------------------- handleOrderShipped

func TestHandleOrderShipped_IncludesTracking(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handleOrderShipped(context.Background(), map[string]interface{}{
		"tenant_id":       "tenant-1",
		"customer_id":     "cust-1",
		"order_id":        "order-1",
		"tracking_number": "TRK-9",
		"carrier":         "Pathao",
		"email":           "alice@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := svc.sent[0]
	if got.Type != string(models.TypeOrderShipped) {
		t.Errorf("type = %s, want order_shipped", got.Type)
	}
	if !strings.Contains(got.Body, "TRK-9") || !strings.Contains(got.Body, "Pathao") {
		t.Errorf("body should carry tracking and carrier, got %q", got.Body)
	}
	// The tracking details are also structured so the UI can deep-link.
	if got.Metadata["tracking_number"] != "TRK-9" || got.Metadata["carrier"] != "Pathao" {
		t.Errorf("metadata = %v, want tracking number and carrier", got.Metadata)
	}
}

func TestHandleOrderShipped_OmitsTrackingWhenAbsent(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handleOrderShipped(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "customer_id": "cust-1", "order_id": "order-1", "carrier": "Pathao",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(svc.sent[0].Body, "Tracking number:") {
		t.Errorf("body advertises a tracking number that was never supplied: %q", svc.sent[0].Body)
	}
}

func TestHandleOrderShipped_SkipsOnMissingFields(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	if err := c.handleOrderShipped(context.Background(), map[string]interface{}{"order_id": "o"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.sent) != 0 {
		t.Errorf("expected no notification, got %d", len(svc.sent))
	}
}

// ------------------------------------------------------- handleOrderCancelled

func TestHandleOrderCancelled_IncludesReason(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handleOrderCancelled(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "customer_id": "cust-1", "order_id": "order-1",
		"reason": "out of stock", "email": "alice@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := svc.sent[0]
	if got.Type != string(models.TypeOrderCancelled) {
		t.Errorf("type = %s, want order_cancelled", got.Type)
	}
	if !strings.Contains(got.Body, "out of stock") {
		t.Errorf("body should carry the reason, got %q", got.Body)
	}
}

func TestHandleOrderCancelled_OmitsReasonWhenAbsent(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handleOrderCancelled(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "customer_id": "cust-1", "order_id": "order-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(svc.sent[0].Body, "Reason:") {
		t.Errorf("body includes an empty reason: %q", svc.sent[0].Body)
	}
}

func TestHandleOrderCancelled_SkipsOnMissingFields(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	if err := c.handleOrderCancelled(context.Background(), map[string]interface{}{"order_id": "o"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.sent) != 0 {
		t.Errorf("expected no notification, got %d", len(svc.sent))
	}
}

// ------------------------------------------------------------ payment events

func TestHandlePaymentCompleted_FormatsAmountAsBDT(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handlePaymentCompleted(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "customer_id": "cust-1", "order_id": "order-1",
		"payment_id": "pay-1", "amount": 1250.5, "email": "alice@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := svc.sent[0]
	if got.Type != string(models.TypePaymentConfirmed) {
		t.Errorf("type = %s, want payment_confirmed", got.Type)
	}
	if !strings.Contains(got.Body, "৳1,250.50") {
		t.Errorf("body should carry the taka-formatted amount, got %q", got.Body)
	}
	// The receipt is filed against the payment, not the order.
	if got.ReferenceType != "payment" || got.ReferenceID != "pay-1" {
		t.Errorf("reference = %s/%s, want payment/pay-1", got.ReferenceType, got.ReferenceID)
	}
}

func TestHandlePaymentCompleted_SkipsOnMissingFields(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	if err := c.handlePaymentCompleted(context.Background(), map[string]interface{}{"order_id": "o"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.sent) != 0 {
		t.Errorf("expected no notification, got %d", len(svc.sent))
	}
}

func TestHandlePaymentFailed_SendsRecoveryPrompt(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handlePaymentFailed(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "customer_id": "cust-1", "order_id": "order-1", "email": "alice@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := svc.sent[0]
	if got.Type != string(models.TypePaymentFailed) {
		t.Errorf("type = %s, want payment_failed", got.Type)
	}
	if !strings.Contains(got.Body, "payment method") {
		t.Errorf("body should tell the customer what to do next, got %q", got.Body)
	}
}

func TestHandlePaymentFailed_SkipsOnMissingFields(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	if err := c.handlePaymentFailed(context.Background(), map[string]interface{}{"order_id": "o"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.sent) != 0 {
		t.Errorf("expected no notification, got %d", len(svc.sent))
	}
}

// ------------------------------------------------------- handleStockLevelLow

// Stock alerts are operational: they go to the tenant admin, never to whoever
// happened to trigger the event.
func TestHandleStockLevelLow_GoesToTheAdmin(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handleStockLevelLow(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "product_id": "prod-1", "sku": "SKU-1", "current_quantity": 3.0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := svc.sent[0]
	if got.Type != string(models.TypeStockAlert) {
		t.Errorf("type = %s, want stock_alert", got.Type)
	}
	if got.UserID != "admin" || got.Recipient != "admin@tenant.local" {
		t.Errorf("alert addressed to %s/%s, want the tenant admin", got.UserID, got.Recipient)
	}
	if !strings.Contains(got.Subject, "SKU-1") {
		t.Errorf("subject should name the SKU, got %q", got.Subject)
	}
	if !strings.Contains(got.Body, "3 units") {
		t.Errorf("body should state the remaining quantity, got %q", got.Body)
	}
	if got.ReferenceType != "product" || got.ReferenceID != "prod-1" {
		t.Errorf("reference = %s/%s, want product/prod-1", got.ReferenceType, got.ReferenceID)
	}
}

func TestHandleStockLevelLow_SkipsOnMissingFields(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]interface{}
	}{
		{"missing tenant", map[string]interface{}{"product_id": "p"}},
		{"missing product", map[string]interface{}{"tenant_id": "t"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeNotificationService{}
			c := newTestConsumer(svc)
			if err := c.handleStockLevelLow(context.Background(), tc.payload); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(svc.sent) != 0 {
				t.Errorf("expected no notification, got %d", len(svc.sent))
			}
		})
	}
}

// ------------------------------------------------------------- handleEvent

func TestHandleEvent_RoutesOrderAndPaymentEvents(t *testing.T) {
	cases := []struct {
		eventType string
		payload   map[string]interface{}
		wantType  string
	}{
		{
			eventType: "UserRegistered",
			payload:   map[string]interface{}{"tenant_id": "t", "user_id": "u", "email": "e@x.com"},
			wantType:  string(models.TypeWelcome),
		},
		{
			// The user service emits either name for the same thing.
			eventType: "UserCreated",
			payload:   map[string]interface{}{"tenant_id": "t", "user_id": "u", "email": "e@x.com"},
			wantType:  string(models.TypeWelcome),
		},
		{
			eventType: "OrderPlaced",
			payload:   map[string]interface{}{"tenant_id": "t", "customer_id": "c", "order_id": "o"},
			wantType:  string(models.TypeOrderConfirmation),
		},
		{
			eventType: "OrderCreated",
			payload:   map[string]interface{}{"tenant_id": "t", "customer_id": "c", "order_id": "o"},
			wantType:  string(models.TypeOrderConfirmation),
		},
		{
			eventType: "OrderShipped",
			payload:   map[string]interface{}{"tenant_id": "t", "customer_id": "c", "order_id": "o"},
			wantType:  string(models.TypeOrderShipped),
		},
		{
			eventType: "OrderCancelled",
			payload:   map[string]interface{}{"tenant_id": "t", "customer_id": "c", "order_id": "o"},
			wantType:  string(models.TypeOrderCancelled),
		},
		{
			eventType: "PaymentCompleted",
			payload:   map[string]interface{}{"tenant_id": "t", "customer_id": "c", "order_id": "o"},
			wantType:  string(models.TypePaymentConfirmed),
		},
		{
			eventType: "PaymentFailed",
			payload:   map[string]interface{}{"tenant_id": "t", "customer_id": "c", "order_id": "o"},
			wantType:  string(models.TypePaymentFailed),
		},
		{
			eventType: "StockLevelLow",
			payload:   map[string]interface{}{"tenant_id": "t", "product_id": "p", "sku": "s"},
			wantType:  string(models.TypeStockAlert),
		},
	}

	for _, tc := range cases {
		t.Run(tc.eventType, func(t *testing.T) {
			svc := &fakeNotificationService{}
			c := newTestConsumer(svc)

			env := &models.EventEnvelope{EventType: tc.eventType, Payload: tc.payload}
			if err := c.handleEvent(context.Background(), env); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(svc.sent) != 1 {
				t.Fatalf("expected 1 notification, got %d", len(svc.sent))
			}
			if svc.sent[0].Type != tc.wantType {
				t.Errorf("type = %s, want %s", svc.sent[0].Type, tc.wantType)
			}
		})
	}
}

// An event this service does not care about is not an error — every topic it
// subscribes to carries traffic for other consumers too.
func TestHandleEvent_IgnoresUnhandledTypes(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	env := &models.EventEnvelope{EventType: "OrderTeleported", Payload: map[string]interface{}{"tenant_id": "t"}}

	if err := c.handleEvent(context.Background(), env); err != nil {
		t.Fatalf("unhandled event type returned an error: %v", err)
	}
	if len(svc.sent) != 0 {
		t.Errorf("expected no notification, got %d", len(svc.sent))
	}
}

func TestHandleEvent_RejectsEmptyPayload(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handleEvent(context.Background(), &models.EventEnvelope{EventType: "OrderPlaced"})

	if err == nil {
		t.Fatal("expected an error for an event with no payload")
	}
}

// Data is the alternative envelope field; both must route identically.
func TestHandleEvent_AcceptsDataInsteadOfPayload(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	env := &models.EventEnvelope{
		EventType: "OrderPlaced",
		Data:      map[string]interface{}{"tenant_id": "t", "customer_id": "c", "order_id": "o"},
	}

	if err := c.handleEvent(context.Background(), env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.sent))
	}
}

// ------------------------------------------- renderWithTemplateOrFallback

func TestRenderWithTemplateOrFallback_UsesActiveTenantTemplate(t *testing.T) {
	svc := &fakeNotificationService{}
	repo := new(mocks.MockNotificationRepository)
	repo.On("GetTemplateByType", mock.Anything, "tenant-1", "email", "welcome").Return(&models.NotificationTemplate{
		ID:              "tmpl-1",
		TenantID:        "tenant-1",
		Channel:         models.ChannelEmail,
		Type:            models.TypeWelcome,
		SubjectTemplate: "Welcome, {{.UserName}}!",
		BodyTemplate:    "Glad to have you, {{.UserName}}.",
		IsActive:        true,
	}, nil)
	c := newConsumerWithRepo(svc, repo)

	err := c.handleUserRegistered(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "user_id": "user-1", "email": "alice@example.com", "name": "Alice",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := svc.sent[0]
	if got.Subject != "Welcome, Alice!" {
		t.Errorf("subject = %q, want the tenant template's rendering", got.Subject)
	}
	if !strings.Contains(got.Body, "Glad to have you, Alice.") {
		t.Errorf("body = %q, want the tenant template's rendering", got.Body)
	}
	repo.AssertExpectations(t)
}

// An inactive template is a draft; the hardcoded copy must still go out.
func TestRenderWithTemplateOrFallback_IgnoresInactiveTemplate(t *testing.T) {
	svc := &fakeNotificationService{}
	repo := new(mocks.MockNotificationRepository)
	repo.On("GetTemplateByType", mock.Anything, "tenant-1", "email", "welcome").Return(&models.NotificationTemplate{
		SubjectTemplate: "Draft subject",
		BodyTemplate:    "Draft body",
		IsActive:        false,
	}, nil)
	c := newConsumerWithRepo(svc, repo)

	if err := c.handleUserRegistered(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "user_id": "user-1", "email": "alice@example.com", "name": "Alice",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if svc.sent[0].Subject == "Draft subject" {
		t.Error("an inactive template was used")
	}
}

// A template lookup failure must not stop the notification going out.
func TestRenderWithTemplateOrFallback_SurvivesLookupFailure(t *testing.T) {
	svc := &fakeNotificationService{}
	repo := new(mocks.MockNotificationRepository)
	repo.On("GetTemplateByType", mock.Anything, "tenant-1", "email", "welcome").Return(nil, errors.New("mongo down"))
	c := newConsumerWithRepo(svc, repo)

	if err := c.handleUserRegistered(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "user_id": "user-1", "email": "alice@example.com", "name": "Alice",
	}); err != nil {
		t.Fatalf("a template lookup failure blocked the notification: %v", err)
	}

	if len(svc.sent) != 1 {
		t.Fatalf("expected the fallback copy to still be sent, got %d", len(svc.sent))
	}
}

func TestRenderWithTemplateOrFallback_HandlesNilTemplate(t *testing.T) {
	svc := &fakeNotificationService{}
	repo := new(mocks.MockNotificationRepository)
	repo.On("GetTemplateByType", mock.Anything, "tenant-1", "email", "welcome").Return(nil, nil)
	c := newConsumerWithRepo(svc, repo)

	if err := c.handleUserRegistered(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "user_id": "user-1", "email": "alice@example.com",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.sent) != 1 {
		t.Fatalf("expected the fallback copy to be sent, got %d", len(svc.sent))
	}
}

// A template that fails to render (bad placeholder) falls back rather than
// sending a half-rendered email.
func TestRenderWithTemplateOrFallback_SurvivesRenderFailure(t *testing.T) {
	svc := &fakeNotificationService{}
	repo := new(mocks.MockNotificationRepository)
	repo.On("GetTemplateByType", mock.Anything, "tenant-1", "email", "welcome").Return(&models.NotificationTemplate{
		SubjectTemplate: "{{.Unclosed",
		BodyTemplate:    "body",
		IsActive:        true,
	}, nil)
	c := newConsumerWithRepo(svc, repo)

	if err := c.handleUserRegistered(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1", "user_id": "user-1", "email": "alice@example.com",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(svc.sent) != 1 {
		t.Fatalf("expected the fallback copy to be sent, got %d", len(svc.sent))
	}
	if strings.Contains(svc.sent[0].Subject, "Unclosed") {
		t.Errorf("a broken template leaked into the subject: %q", svc.sent[0].Subject)
	}
}

// ------------------------------------------------------------------ helpers

func TestReadFloat(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want float64
	}{
		{name: "float64 (what JSON yields)", in: float64(12.5), want: 12.5},
		{name: "float32", in: float32(2.5), want: 2.5},
		{name: "int", in: 7, want: 7},
		{name: "int64", in: int64(9), want: 9},
		{name: "string is not coerced", in: "12.5", want: 0},
		{name: "nil", in: nil, want: 0},
		{name: "map", in: map[string]interface{}{}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readFloat(tt.in); got != tt.want {
				t.Errorf("readFloat(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestReadInt(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want int
	}{
		{name: "float64 truncates", in: float64(3.9), want: 3},
		{name: "float32", in: float32(2.9), want: 2},
		{name: "int", in: 7, want: 7},
		{name: "int64", in: int64(9), want: 9},
		{name: "string is not coerced", in: "3", want: 0},
		{name: "nil", in: nil, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readInt(tt.in); got != tt.want {
				t.Errorf("readInt(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// An explicit line total wins; otherwise it is derived. A discounted line
// would otherwise be billed at list price on the receipt.
func TestReceiptItemLineTotal(t *testing.T) {
	tests := []struct {
		name string
		item receiptItem
		want float64
	}{
		{name: "explicit total wins", item: receiptItem{Quantity: 2, UnitPrice: 500, Total: 900}, want: 900},
		{name: "derived from quantity and price", item: receiptItem{Quantity: 2, UnitPrice: 500}, want: 1000},
		{name: "zero quantity", item: receiptItem{UnitPrice: 500}, want: 0},
		{name: "empty item", item: receiptItem{}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.LineTotal(); got != tt.want {
				t.Errorf("LineTotal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUserEmail(t *testing.T) {
	t.Run("uses the stored preference", func(t *testing.T) {
		svc := &prefStubService{
			fakeNotificationService: &fakeNotificationService{},
			pref:                    &models.UserPreferenceResponse{Email: "stored@example.com"},
		}
		logger := logrus.New()
		logger.SetOutput(discardWriter{})
		c := &EventConsumer{service: svc, logger: logger, stop: make(chan struct{})}

		if got := c.getUserEmail(context.Background(), "tenant-1", "user-1"); got != "stored@example.com" {
			t.Errorf("getUserEmail = %q, want the stored address", got)
		}
	})

	t.Run("falls back to a placeholder", func(t *testing.T) {
		c := newTestConsumer(&fakeNotificationService{})

		if got := c.getUserEmail(context.Background(), "tenant-1", "user-1"); got != "user-1@placeholder.local" {
			t.Errorf("getUserEmail = %q, want the placeholder address", got)
		}
	})

	t.Run("falls back when the preference has no email", func(t *testing.T) {
		svc := &prefStubService{
			fakeNotificationService: &fakeNotificationService{},
			pref:                    &models.UserPreferenceResponse{},
		}
		logger := logrus.New()
		logger.SetOutput(discardWriter{})
		c := &EventConsumer{service: svc, logger: logger, stop: make(chan struct{})}

		if got := c.getUserEmail(context.Background(), "tenant-1", "user-1"); got != "user-1@placeholder.local" {
			t.Errorf("getUserEmail = %q, want the placeholder address", got)
		}
	})
}
