package messaging

import (
	"encoding/json"
	"testing"
)

// external_consumer consumes payment-events, inventory-events and
// shipping-events. Every producer on those topics currently sends `version` as
// a quoted string, so this was latent rather than live -- but order-service's
// own publisher emits it as a bare number, and a `string` field rejects that
// shape outright. json.Unmarshal fails, handleMessage returns before reading
// EventType, and the event vanishes with nothing logged as an error.
func TestExternalEnvelopeAcceptsNumericVersion(t *testing.T) {
	raw := []byte(`{"event_id":"e1","event_type":"PaymentCompleted","version":6,"payload":{"order_id":"o-1"}}`)

	var env ExternalEventEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal = %v; a numeric version must not reject the envelope", err)
	}
	if env.EventType != "PaymentCompleted" {
		t.Errorf("EventType = %q, want PaymentCompleted", env.EventType)
	}
	if got := env.GetPayload()["order_id"]; got != "o-1" {
		t.Errorf("payload order_id = %v, want o-1", got)
	}
}

func TestExternalEnvelopeAcceptsStringVersion(t *testing.T) {
	raw := []byte(`{"event_id":"e2","event_type":"InventoryReserved","version":"1.0.0","payload":{"order_id":"o-2"}}`)

	var env ExternalEventEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal = %v; the shape every current producer sends", err)
	}
	if env.Version.String() != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", env.Version)
	}
}

// Data is the field order-service's own events use; Payload is what the other
// services send. GetPayload has to cover both.
func TestExternalEnvelopeFallsBackToData(t *testing.T) {
	raw := []byte(`{"event_type":"OrderShipped","version":6,"data":{"tracking_number":"TRK-9"}}`)

	var env ExternalEventEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal = %v", err)
	}
	if got := env.GetPayload()["tracking_number"]; got != "TRK-9" {
		t.Errorf("GetPayload()[tracking_number] = %v, want TRK-9", got)
	}
}

func TestExternalEnvelopeToleratesAbsentVersion(t *testing.T) {
	var env ExternalEventEnvelope
	if err := json.Unmarshal([]byte(`{"event_type":"X","payload":{}}`), &env); err != nil {
		t.Fatalf("unmarshal = %v", err)
	}
	if env.Version.String() != "" {
		t.Errorf("Version = %q, want empty", env.Version)
	}
}

// The failure this change prevents, demonstrated rather than asserted.
//
// `legacyEnvelope` is the shape this file used until now. Worth being precise
// about the mechanism: encoding/json keeps decoding past a type mismatch and
// returns the UnmarshalTypeError at the end, so the struct is actually
// populated -- the data is not lost in the decoder. What loses the event is
// the consumer's own `if err := json.Unmarshal(...); err != nil { return }`,
// which never looks at the fields it already has. The log line says
// "unmarshal error", not "dropped a PaymentCompleted event", which is why this
// class of failure stays invisible.
//
// Kept here so the field type is not innocently simplified back to a string.
func TestStringVersionFieldRejectsNumericVersion(t *testing.T) {
	type legacyEnvelope struct {
		EventType string                 `json:"event_type"`
		Version   string                 `json:"version,omitempty"`
		Payload   map[string]interface{} `json:"payload,omitempty"`
	}

	raw := []byte(`{"event_type":"PaymentCompleted","version":6,"payload":{"order_id":"o-1"}}`)

	var legacy legacyEnvelope
	if err := json.Unmarshal(raw, &legacy); err == nil {
		t.Fatal("a string-typed version field accepted a numeric version; this test no longer demonstrates anything")
	}

	// The same bytes, into the shape this file now uses: no error, so the
	// consumer proceeds to dispatch on EventType.
	var env ExternalEventEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("EventVersion field also rejected it: %v", err)
	}
	if env.EventType != "PaymentCompleted" {
		t.Errorf("EventType = %q, want PaymentCompleted", env.EventType)
	}
	if env.Version.Int() != 6 {
		t.Errorf("Version.Int() = %d, want 6", env.Version.Int())
	}
}
