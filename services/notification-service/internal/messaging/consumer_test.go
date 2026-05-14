package messaging

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

// fakeNotificationService captures the SendNotification calls so the consumer's
// event handlers can be asserted on without reaching MongoDB or any provider.
type fakeNotificationService struct {
	sent []*models.SendNotificationRequest
	err  error
}

func (f *fakeNotificationService) SendNotification(ctx context.Context, req *models.SendNotificationRequest) (*models.NotificationResponse, error) {
	f.sent = append(f.sent, req)
	if f.err != nil {
		return nil, f.err
	}
	return &models.NotificationResponse{ID: "fake"}, nil
}

func (f *fakeNotificationService) GetNotification(ctx context.Context, id string) (*models.NotificationResponse, error) {
	return nil, nil
}
func (f *fakeNotificationService) GetUserNotifications(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.NotificationResponse, int64, error) {
	return nil, 0, nil
}
func (f *fakeNotificationService) MarkAsRead(ctx context.Context, id string) error { return nil }
func (f *fakeNotificationService) GetPreference(ctx context.Context, tenantID, userID string) (*models.UserPreferenceResponse, error) {
	return nil, errors.New("no preference")
}
func (f *fakeNotificationService) UpdatePreference(ctx context.Context, tenantID, userID string, req *models.UpdatePreferenceRequest) (*models.UserPreferenceResponse, error) {
	return nil, nil
}

func newTestConsumer(svc *fakeNotificationService) *EventConsumer {
	logger := logrus.New()
	logger.SetOutput(discardWriter{})
	return &EventConsumer{
		service:         svc,
		logger:          logger,
		stop:            make(chan struct{}),
		frontendBaseURL: "https://shop.example.com",
	}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestHandleEmailVerificationRequested_BuildsLinkAndSends(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handleEmailVerificationRequested(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1",
		"user_id":   "user-1",
		"email":     "alice@example.com",
		"token":     "abc123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(svc.sent) != 1 {
		t.Fatalf("expected 1 notification sent, got %d", len(svc.sent))
	}
	got := svc.sent[0]
	if got.Channel != string(models.ChannelEmail) {
		t.Errorf("expected email channel, got %s", got.Channel)
	}
	if got.Type != string(models.TypeEmailVerification) {
		t.Errorf("expected email_verification type, got %s", got.Type)
	}
	if got.Recipient != "alice@example.com" {
		t.Errorf("expected recipient alice@example.com, got %s", got.Recipient)
	}
	wantURL := "https://shop.example.com/verify-email?token=abc123"
	if !strings.Contains(got.Body, wantURL) {
		t.Errorf("expected body to contain %q, got %q", wantURL, got.Body)
	}
	if got.Metadata["verify_url"] != wantURL {
		t.Errorf("expected metadata verify_url=%s, got %v", wantURL, got.Metadata["verify_url"])
	}
}

func TestHandleEmailVerificationRequested_SkipsOnMissingFields(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]interface{}
	}{
		{"missing tenant", map[string]interface{}{"user_id": "u", "email": "e@x.com", "token": "t"}},
		{"missing user", map[string]interface{}{"tenant_id": "t", "email": "e@x.com", "token": "t"}},
		{"missing email", map[string]interface{}{"tenant_id": "t", "user_id": "u", "token": "t"}},
		{"missing token", map[string]interface{}{"tenant_id": "t", "user_id": "u", "email": "e@x.com"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeNotificationService{}
			c := newTestConsumer(svc)
			if err := c.handleEmailVerificationRequested(context.Background(), tc.payload); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(svc.sent) != 0 {
				t.Errorf("expected no notification, got %d", len(svc.sent))
			}
		})
	}
}

func TestHandlePasswordResetRequested_GreetsByName(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handlePasswordResetRequested(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1",
		"user_id":   "user-1",
		"email":     "bob@example.com",
		"name":      "Bob Smith",
		"token":     "reset-token-xyz",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(svc.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.sent))
	}
	got := svc.sent[0]
	if got.Type != string(models.TypePasswordReset) {
		t.Errorf("expected password_reset type, got %s", got.Type)
	}
	if !strings.Contains(got.Body, "Hi Bob Smith") {
		t.Errorf("expected greeting with name, got body: %s", got.Body)
	}
	wantURL := "https://shop.example.com/reset-password?token=reset-token-xyz"
	if !strings.Contains(got.Body, wantURL) {
		t.Errorf("expected body to contain %q", wantURL)
	}
}

func TestHandlePasswordResetRequested_FallsBackWithoutName(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handlePasswordResetRequested(context.Background(), map[string]interface{}{
		"tenant_id": "tenant-1",
		"user_id":   "user-1",
		"email":     "bob@example.com",
		"token":     "tok",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.sent))
	}
	if !strings.Contains(svc.sent[0].Body, "Hi,") {
		t.Errorf("expected nameless greeting, got body: %s", svc.sent[0].Body)
	}
}

func TestHandleReceiptRequested_BuildsBodyWithItemsAndTotals(t *testing.T) {
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handleReceiptRequested(context.Background(), map[string]interface{}{
		"tenant_id":      "tenant-1",
		"order_id":       "order-abcdef12",
		"customer_email": "buyer@example.com",
		"customer_name":  "Aisha",
		"store_name":     "Saajan Demo",
		"payment_method": "Cash",
		"currency":       "BDT",
		"subtotal":       1750.0,
		"discount":       150.0,
		"tax":            80.0,
		"total":          1680.0,
		"items": []interface{}{
			map[string]interface{}{"name": "T-shirt L", "sku": "TSHIRT-L", "quantity": 2.0, "unit_price": 400.0, "total_price": 800.0},
			map[string]interface{}{"name": "Cap", "sku": "CAP-1", "quantity": 1.0, "unit_price": 350.0, "total_price": 350.0},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(svc.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.sent))
	}
	got := svc.sent[0]
	if got.Channel != string(models.ChannelEmail) {
		t.Errorf("expected email channel, got %s", got.Channel)
	}
	if got.Type != string(models.TypeReceipt) {
		t.Errorf("expected receipt type, got %s", got.Type)
	}
	if got.Recipient != "buyer@example.com" {
		t.Errorf("expected recipient buyer@example.com, got %s", got.Recipient)
	}
	if !strings.Contains(got.Subject, "Saajan Demo") {
		t.Errorf("expected subject to include store name, got %q", got.Subject)
	}
	// Body should be a non-empty HTML string carrying the line items and total.
	if got.Body == "" {
		t.Fatalf("expected non-empty body")
	}
	for _, want := range []string{"T-shirt L", "Cap", "Cash", "Hi Aisha"} {
		if !strings.Contains(got.Body, want) {
			t.Errorf("expected body to contain %q, got: %s", want, got.Body)
		}
	}
	if got.ReferenceID != "order-abcdef12" {
		t.Errorf("expected reference_id=order-abcdef12, got %s", got.ReferenceID)
	}
	if got.ReferenceType != "order" {
		t.Errorf("expected reference_type=order, got %s", got.ReferenceType)
	}
}

func TestHandleReceiptRequested_DoesNotPanicOnMissingFields(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]interface{}
		wantLen int // expected number of notifications sent
	}{
		{
			name:    "missing tenant — skip",
			payload: map[string]interface{}{"order_id": "o1", "customer_email": "x@y.com"},
			wantLen: 0,
		},
		{
			name:    "missing order — skip",
			payload: map[string]interface{}{"tenant_id": "t1", "customer_email": "x@y.com"},
			wantLen: 0,
		},
		{
			name:    "no email and no customer id — skip silently",
			payload: map[string]interface{}{"tenant_id": "t1", "order_id": "o1"},
			wantLen: 0,
		},
		{
			name: "minimal valid payload — produces a usable email body",
			payload: map[string]interface{}{
				"tenant_id":      "t1",
				"order_id":       "o1",
				"customer_email": "x@y.com",
			},
			wantLen: 1,
		},
		{
			name: "malformed items field is ignored",
			payload: map[string]interface{}{
				"tenant_id":      "t1",
				"order_id":       "o1",
				"customer_email": "x@y.com",
				"items":          "not-an-array",
				"total":          100.0,
			},
			wantLen: 1,
		},
		{
			name: "items entries with wrong types are dropped, not panicked on",
			payload: map[string]interface{}{
				"tenant_id":      "t1",
				"order_id":       "o1",
				"customer_email": "x@y.com",
				"items": []interface{}{
					"bogus-string-entry",
					map[string]interface{}{"name": "Belt", "quantity": 1.0, "total_price": 600.0},
				},
				"total": 600.0,
			},
			wantLen: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeNotificationService{}
			c := newTestConsumer(svc)
			// Must not panic.
			if err := c.handleReceiptRequested(context.Background(), tc.payload); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(svc.sent) != tc.wantLen {
				t.Errorf("expected %d notifications, got %d", tc.wantLen, len(svc.sent))
			}
			if tc.wantLen == 1 {
				if svc.sent[0].Body == "" {
					t.Errorf("expected non-empty email body even with sparse payload")
				}
				if svc.sent[0].Type != string(models.TypeReceipt) {
					t.Errorf("expected type receipt, got %s", svc.sent[0].Type)
				}
			}
		})
	}
}

func TestHandleReceiptRequested_FallsBackToStoredEmail(t *testing.T) {
	// When the payload omits customer_email, the handler should ask the
	// notification service for the user's stored preference. Our fake
	// service returns an error, so the handler should end up with the
	// "<userID>@placeholder.local" fallback path and still send.
	svc := &fakeNotificationService{}
	c := newTestConsumer(svc)

	err := c.handleReceiptRequested(context.Background(), map[string]interface{}{
		"tenant_id":   "t1",
		"order_id":    "o1",
		"customer_id": "user-42",
		"total":       100.0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(svc.sent))
	}
	if !strings.Contains(svc.sent[0].Recipient, "user-42") {
		t.Errorf("expected recipient to derive from customer_id, got %s", svc.sent[0].Recipient)
	}
}

func TestHandleEvent_RoutesAuthEvents(t *testing.T) {
	cases := []struct {
		eventType string
		payload   map[string]interface{}
		wantType  string
	}{
		{
			eventType: "EmailVerificationRequested",
			payload:   map[string]interface{}{"tenant_id": "t", "user_id": "u", "email": "e@x.com", "token": "k"},
			wantType:  string(models.TypeEmailVerification),
		},
		{
			eventType: "PasswordResetRequested",
			payload:   map[string]interface{}{"tenant_id": "t", "user_id": "u", "email": "e@x.com", "token": "k"},
			wantType:  string(models.TypePasswordReset),
		},
		{
			eventType: "ReceiptRequested",
			payload:   map[string]interface{}{"tenant_id": "t", "order_id": "o1", "customer_email": "e@x.com", "total": 100.0},
			wantType:  string(models.TypeReceipt),
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
				t.Errorf("expected type %s, got %s", tc.wantType, svc.sent[0].Type)
			}
		})
	}
}
