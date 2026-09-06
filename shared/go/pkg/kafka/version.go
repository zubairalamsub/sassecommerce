package kafka

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// EventVersion is the `version` field on a Kafka event envelope.
//
// It is lenient because producers in this platform disagree about the field in
// two independent ways, and both are legitimate:
//
//   - JSON type: order-service emits a bare number, others emit a quoted
//     string. Consumers declared it `string`, so every order event was dropped
//     with "cannot unmarshal number into Go struct field
//     EventEnvelope.version" — order confirmation, shipped, cancelled, payment
//     and receipt emails never sent, and review invitations never fired.
//
//   - Meaning: on order-events it is the aggregate's sequence number (6), while
//     on user-events it is a schema version ("1.0.0"). A type that insisted on
//     an integer would fix the first problem by breaking the second — the
//     user-events path is the one that currently works.
//
// So the raw text is preserved and Int() is offered for callers that want the
// sequence. Decoding leniently rather than changing the producer is deliberate:
// a producer change would need every consumer redeployed in lockstep, and would
// break replay of every event already in the log.
type EventVersion string

// UnmarshalJSON accepts a JSON number, a string, or null.
func (v *EventVersion) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*v = ""
		return nil
	}

	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		*v = EventVersion(s)
		return nil
	}

	// A bare number. json.Number keeps the original text, so 6 stays "6" and a
	// float is not silently truncated.
	var n json.Number
	if err := json.Unmarshal(trimmed, &n); err != nil {
		return err
	}
	*v = EventVersion(n.String())
	return nil
}

// MarshalJSON emits a bare number when the value is an integer and a string
// otherwise, so a re-published envelope keeps the shape its producer used.
func (v EventVersion) MarshalJSON() ([]byte, error) {
	if v == "" {
		return []byte(`""`), nil
	}
	if _, err := strconv.Atoi(string(v)); err == nil {
		return []byte(v), nil
	}
	return json.Marshal(string(v))
}

// Int returns the version as an integer, or 0 when it is not one (a schema
// version such as "1.0.0", or absent). Callers that care about the difference
// should use IsSequence.
func (v EventVersion) Int() int {
	n, err := strconv.Atoi(string(v))
	if err != nil {
		return 0
	}
	return n
}

// IsSequence reports whether the version is a plain integer — an aggregate
// sequence number rather than a schema version.
func (v EventVersion) IsSequence() bool {
	_, err := strconv.Atoi(string(v))
	return err == nil
}

// String returns the version exactly as it arrived.
func (v EventVersion) String() string { return string(v) }
