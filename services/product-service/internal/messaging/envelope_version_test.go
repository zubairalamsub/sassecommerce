package messaging

import (
	"encoding/json"
	"testing"
)

// This consumer reads inventory-events. The producer there (inventory-service,
// .NET) sends `version` as a quoted string today, so declaring the field as a
// Go string was latent rather than live -- but a numeric version from any
// producer would have failed json.Unmarshal outright, returning before
// handleMessage read EventType and dropping a legitimate stock update in
// silence.
func TestEventEnvelopeAcceptsNumericVersion(t *testing.T) {
	raw := []byte(`{"event_id":"e1","event_type":"InventoryUpdated","version":6,"payload":{"product_id":"p-1","tenant_id":"t-1"}}`)

	var env EventEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal = %v; a numeric version must not reject the envelope", err)
	}
	if env.EventType != "InventoryUpdated" {
		t.Errorf("EventType = %q, want InventoryUpdated", env.EventType)
	}
	if got := env.Payload["product_id"]; got != "p-1" {
		t.Errorf("payload product_id = %v, want p-1", got)
	}
}

func TestEventEnvelopeAcceptsStringVersion(t *testing.T) {
	raw := []byte(`{"event_type":"InventoryUpdated","version":"1.0.0","payload":{"product_id":"p-1"}}`)

	var env EventEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal = %v; the shape inventory-service actually sends", err)
	}
	if env.Version.String() != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", env.Version)
	}
}
