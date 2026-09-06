package kafka

import (
	"encoding/json"
	"strings"
	"testing"
)

type versionEnvelope struct {
	EventType string       `json:"event_type"`
	Version   EventVersion `json:"version,omitempty"`
}

// The bug this type exists for: order-service emits version as a JSON number,
// and consumers declaring it a string dropped every order event with
// "cannot unmarshal number into Go struct field EventEnvelope.version".
func TestEventVersionAcceptsNumberFromOrderService(t *testing.T) {
	var env versionEnvelope

	err := json.Unmarshal([]byte(`{"event_type":"OrderShipped","version":6}`), &env)

	if err != nil {
		t.Fatalf("unmarshal = %v, want a numeric version to decode", err)
	}
	if env.Version.Int() != 6 {
		t.Errorf("Int() = %d, want 6", env.Version.Int())
	}
	if !env.Version.IsSequence() {
		t.Error("IsSequence() = false for a bare number")
	}
}

// The other half of the problem: user-events carry a schema version, not a
// sequence. Rejecting it would break the flows that currently work — welcome,
// password reset and email verification.
func TestEventVersionAcceptsSemanticVersion(t *testing.T) {
	var env versionEnvelope

	err := json.Unmarshal([]byte(`{"event_type":"PasswordResetRequested","version":"1.0.0"}`), &env)

	if err != nil {
		t.Fatalf("unmarshal = %v, want a schema version to decode", err)
	}
	if env.Version.String() != "1.0.0" {
		t.Errorf("String() = %q, want the text preserved", env.Version.String())
	}
	if env.Version.IsSequence() {
		t.Error("IsSequence() = true for \"1.0.0\"")
	}
	// Callers asking for a sequence get zero rather than a panic or an error.
	if env.Version.Int() != 0 {
		t.Errorf("Int() = %d, want 0 for a non-sequence", env.Version.Int())
	}
}

func TestEventVersionAcceptsQuotedNumber(t *testing.T) {
	var env versionEnvelope

	if err := json.Unmarshal([]byte(`{"version":"3"}`), &env); err != nil {
		t.Fatalf("unmarshal = %v", err)
	}
	if env.Version.Int() != 3 || !env.Version.IsSequence() {
		t.Errorf("version = %q, want it read as sequence 3", env.Version)
	}
}

func TestEventVersionAcceptsAbsentNullAndEmpty(t *testing.T) {
	cases := []struct{ name, raw string }{
		{name: "absent", raw: `{"event_type":"X"}`},
		{name: "null", raw: `{"event_type":"X","version":null}`},
		{name: "empty string", raw: `{"event_type":"X","version":""}`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var env versionEnvelope
			if err := json.Unmarshal([]byte(c.raw), &env); err != nil {
				t.Fatalf("unmarshal = %v, want %s tolerated", err, c.name)
			}
			if env.Version.String() != "" || env.Version.Int() != 0 {
				t.Errorf("version = %q, want empty", env.Version)
			}
		})
	}
}

// Leniency covers the shapes producers actually emit, not arbitrary JSON — a
// structured value here means the envelope is genuinely malformed.
func TestEventVersionRejectsStructuredValues(t *testing.T) {
	cases := []struct{ name, raw string }{
		{name: "object", raw: `{"version":{"a":1}}`},
		{name: "array", raw: `{"version":[1]}`},
		{name: "boolean", raw: `{"version":true}`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var env versionEnvelope
			if err := json.Unmarshal([]byte(c.raw), &env); err == nil {
				t.Errorf("unmarshal accepted %s, want an error", c.name)
			}
		})
	}
}

// A re-published envelope should keep the shape its producer used rather than
// converting between the two conventions.
func TestEventVersionMarshalPreservesShape(t *testing.T) {
	cases := []struct {
		name    string
		version EventVersion
		want    string
	}{
		{name: "sequence stays a number", version: "6", want: `"version":6`},
		{name: "schema version stays a string", version: "1.0.0", want: `"version":"1.0.0"`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := json.Marshal(versionEnvelope{Version: c.version})
			if err != nil {
				t.Fatalf("marshal = %v", err)
			}
			if !strings.Contains(string(out), c.want) {
				t.Errorf("marshalled %s, want it to contain %s", out, c.want)
			}
		})
	}
}

func TestEventVersionRoundTrips(t *testing.T) {
	for _, want := range []string{"", "0", "1", "42", "1.0.0", "v2"} {
		out, err := json.Marshal(versionEnvelope{Version: EventVersion(want)})
		if err != nil {
			t.Fatalf("marshal(%q) = %v", want, err)
		}
		var back versionEnvelope
		if err := json.Unmarshal(out, &back); err != nil {
			t.Fatalf("unmarshal(%s) = %v", out, err)
		}
		if back.Version.String() != want {
			t.Errorf("round trip of %q = %q", want, back.Version)
		}
	}
}

// The two real envelopes this platform puts on the wire, side by side.
func TestBothRealEnvelopeShapesDecode(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantSeq bool
		wantStr string
	}{
		{
			name:    "order-service: numeric sequence",
			raw:     `{"event_id":"e1","event_type":"OrderShipped","aggregate_id":"order-1","version":6,"data":{"tracking_number":"TRK-9"}}`,
			wantSeq: true,
			wantStr: "6",
		},
		{
			name:    "user-service: schema version",
			raw:     `{"event_id":"e2","event_type":"PasswordResetRequested","version":"1.0.0","payload":{"email":"a@b.test"}}`,
			wantSeq: false,
			wantStr: "1.0.0",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var env struct {
				EventType string                 `json:"event_type"`
				Version   EventVersion           `json:"version"`
				Data      map[string]interface{} `json:"data"`
				Payload   map[string]interface{} `json:"payload"`
			}

			if err := json.Unmarshal([]byte(c.raw), &env); err != nil {
				t.Fatalf("real envelope failed to decode: %v", err)
			}
			if env.Version.String() != c.wantStr {
				t.Errorf("version = %q, want %q", env.Version, c.wantStr)
			}
			if env.Version.IsSequence() != c.wantSeq {
				t.Errorf("IsSequence() = %v, want %v", env.Version.IsSequence(), c.wantSeq)
			}
			// The payload must survive either way — that is the whole point.
			if env.Data == nil && env.Payload == nil {
				t.Error("payload lost")
			}
		})
	}
}
