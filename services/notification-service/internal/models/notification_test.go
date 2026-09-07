package models

import (
	"encoding/json"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

// Gin's binding engine is exactly validator.New() with SetTagName("binding")
// (see gin-gonic/gin/binding/default_validator.go) -- mirroring that here
// exercises the real validation contract these tags declare, not just their
// presence. Shared across this package's test files.
func newBindingValidator() *validator.Validate {
	v := validator.New()
	v.SetTagName("binding")
	return v
}

func TestEventEnvelope_GetPayload(t *testing.T) {
	tests := []struct {
		name     string
		envelope EventEnvelope
		wantFrom interface{}
	}{
		{
			name:     "prefers payload over data",
			envelope: EventEnvelope{Payload: map[string]interface{}{"from": "payload"}, Data: map[string]interface{}{"from": "data"}},
			wantFrom: "payload",
		},
		{
			name:     "falls back to data when payload is nil",
			envelope: EventEnvelope{Data: map[string]interface{}{"from": "data"}},
			wantFrom: "data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.envelope.GetPayload()
			assert.Equal(t, tt.wantFrom, got["from"])
		})
	}

	t.Run("both nil returns nil", func(t *testing.T) {
		assert.Nil(t, (&EventEnvelope{}).GetPayload())
	})
}

// The version field is the one that took down order-event consumption
// platform-wide: producers disagree on whether it is a bare JSON number or a
// quoted string, and a consumer declared as one type silently dropped every
// event of the other shape (see sharedkafka.EventVersion's doc comment).
// This pins that notification-service's own envelope uses that lenient type
// -- not a plain string -- so both wire shapes decode to the same value
// instead of failing json.Unmarshal.
func TestEventEnvelope_Version_AcceptsBothWireShapes(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{name: "bare number (order-service's historical shape)", json: `{"event_id":"e1","event_type":"OrderCreated","version":6}`, want: "6"},
		{name: "quoted string (user-service's shape)", json: `{"event_id":"e1","event_type":"UserCreated","version":"1.0.0"}`, want: "1.0.0"},
		{name: "explicit null", json: `{"event_id":"e1","event_type":"UserCreated","version":null}`, want: ""},
		{name: "absent", json: `{"event_id":"e1","event_type":"UserCreated"}`, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var env EventEnvelope
			err := json.Unmarshal([]byte(tt.json), &env)

			assert.NoError(t, err, "the envelope must decode regardless of which shape the producer used for version")
			assert.Equal(t, tt.want, env.Version.String())
		})
	}
}

func TestEventEnvelope_JSON_WireFieldNames(t *testing.T) {
	env := EventEnvelope{
		EventID:       "evt-1",
		EventType:     "OrderPlaced",
		AggregateID:   "order-1",
		AggregateType: "Order",
		Payload:       map[string]interface{}{"order_id": "order-1"},
		Metadata:      map[string]interface{}{"source": "order-service"},
	}

	raw, err := json.Marshal(env)
	assert.NoError(t, err)

	var decoded map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &decoded))

	for _, key := range []string{"event_id", "event_type", "aggregate_id", "aggregate_type", "timestamp", "payload", "metadata"} {
		assert.Contains(t, decoded, key, "expected the wire field %q", key)
	}
	// Data and Version were never set and are both tagged omitempty.
	assert.NotContains(t, decoded, "data")
	assert.NotContains(t, decoded, "version")
}

func TestNotification_JSON_WireFieldNames(t *testing.T) {
	n := Notification{
		ID: "n1", TenantID: "tenant-1", UserID: "user-1", Channel: ChannelEmail,
		Type: TypeOrderConfirmation, Status: StatusSent, Subject: "Your order",
		Body: "Thanks!", Recipient: "buyer@example.com",
		ReferenceID: "order-1", ReferenceType: "order",
	}

	raw, err := json.Marshal(n)
	assert.NoError(t, err)

	var decoded map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &decoded))

	for _, key := range []string{
		"id", "tenant_id", "user_id", "channel", "type", "status", "subject",
		"body", "recipient", "reference_id", "reference_type", "created_at", "updated_at",
	} {
		assert.Contains(t, decoded, key, "expected the wire field %q", key)
	}
	for _, key := range []string{"provider_name", "provider_message_id", "failure_reason", "read_at", "sent_at", "metadata"} {
		assert.NotContains(t, decoded, key, "unset optional field %q should be omitted rather than sent as null/empty", key)
	}
}

func TestChannel_WireValuesAreStable(t *testing.T) {
	tests := []struct {
		channel Channel
		want    string
	}{
		{ChannelEmail, `"email"`},
		{ChannelSMS, `"sms"`},
		{ChannelPush, `"push"`},
	}
	for _, tt := range tests {
		t.Run(string(tt.channel), func(t *testing.T) {
			raw, err := json.Marshal(tt.channel)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(raw))
		})
	}
}

func TestNotificationStatus_WireValuesAreStable(t *testing.T) {
	tests := []struct {
		status NotificationStatus
		want   string
	}{
		{StatusPending, `"pending"`},
		{StatusSent, `"sent"`},
		{StatusDelivered, `"delivered"`},
		{StatusFailed, `"failed"`},
		{StatusRead, `"read"`},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			raw, err := json.Marshal(tt.status)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(raw))
		})
	}
}

// These values are stored in Mongo and matched against by the frontend and
// by template lookups (GetTemplateByType); a renamed constant silently
// breaks both without touching a single line of Go code that reads it.
func TestNotificationType_WireValuesAreStable(t *testing.T) {
	tests := []struct {
		typ  NotificationType
		want string
	}{
		{TypeOrderConfirmation, "order_confirmation"},
		{TypeOrderShipped, "order_shipped"},
		{TypeOrderDelivered, "order_delivered"},
		{TypeOrderCancelled, "order_cancelled"},
		{TypePaymentConfirmed, "payment_confirmed"},
		{TypePaymentFailed, "payment_failed"},
		{TypeWelcome, "welcome"},
		{TypeEmailVerification, "email_verification"},
		{TypePasswordReset, "password_reset"},
		{TypeReceipt, "receipt"},
		{TypeStockAlert, "stock_alert"},
		{TypePromotion, "promotion"},
		{TypeCustom, "custom"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			raw, err := json.Marshal(tt.typ)
			assert.NoError(t, err)
			assert.Equal(t, `"`+tt.want+`"`, string(raw))
		})
	}
}

func TestSendNotificationRequest_BindingValidation(t *testing.T) {
	validate := newBindingValidator()

	base := func() SendNotificationRequest {
		return SendNotificationRequest{
			TenantID: "tenant-1", UserID: "user-1", Channel: "email", Type: "welcome",
			Subject: "Hi", Body: "Welcome!", Recipient: "user@example.com",
		}
	}

	tests := []struct {
		name    string
		mutate  func(*SendNotificationRequest)
		wantErr bool
	}{
		{name: "valid request passes", mutate: func(r *SendNotificationRequest) {}, wantErr: false},
		{name: "missing tenant_id fails", mutate: func(r *SendNotificationRequest) { r.TenantID = "" }, wantErr: true},
		{name: "missing user_id fails", mutate: func(r *SendNotificationRequest) { r.UserID = "" }, wantErr: true},
		{name: "missing channel fails", mutate: func(r *SendNotificationRequest) { r.Channel = "" }, wantErr: true},
		{name: "missing type fails", mutate: func(r *SendNotificationRequest) { r.Type = "" }, wantErr: true},
		{name: "missing subject fails", mutate: func(r *SendNotificationRequest) { r.Subject = "" }, wantErr: true},
		{name: "missing body fails", mutate: func(r *SendNotificationRequest) { r.Body = "" }, wantErr: true},
		{name: "missing recipient fails", mutate: func(r *SendNotificationRequest) { r.Recipient = "" }, wantErr: true},
		{name: "reference fields are optional", mutate: func(r *SendNotificationRequest) { r.ReferenceID = ""; r.ReferenceType = "" }, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := base()
			tt.mutate(&req)

			err := validate.Struct(req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// A user explicitly disabling a channel (false) must be distinguishable from
// simply not mentioning that channel in the request -- otherwise a partial
// update of just SMSEnabled would silently re-enable email for everyone.
func TestUpdatePreferenceRequest_OptionalBoolsDistinguishAbsentFromExplicitFalse(t *testing.T) {
	raw := []byte(`{"email_enabled":false,"sms_enabled":true}`)

	var req UpdatePreferenceRequest
	assert.NoError(t, json.Unmarshal(raw, &req))

	if assert.NotNil(t, req.EmailEnabled) {
		assert.False(t, *req.EmailEnabled)
	}
	if assert.NotNil(t, req.SMSEnabled) {
		assert.True(t, *req.SMSEnabled)
	}
	assert.Nil(t, req.PushEnabled, "an omitted preference must stay nil so the service does not overwrite it")
}
