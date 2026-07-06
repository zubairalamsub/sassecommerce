package kafka

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"

	"github.com/segmentio/kafka-go"
)

// SignatureHeader is the Kafka message header carrying the HMAC-SHA256 of the
// message value, hex-encoded. Signing via a header keeps the JSON payload
// byte-for-byte unchanged for consumers that predate signing.
const SignatureHeader = "x-event-signature"

const envEventSigningKey = "EVENT_SIGNING_KEY"

var (
	// ErrMissingSignature is returned when verification is enabled and the
	// message has no signature header (unsigned producer or stripped header).
	ErrMissingSignature = errors.New("kafka message has no event signature")
	// ErrInvalidSignature is returned when the signature does not match the
	// message value (tampered payload or wrong signing key).
	ErrInvalidSignature = errors.New("kafka message signature mismatch")
)

// EventSigner signs and verifies Kafka message values with HMAC-SHA256 using
// a shared key from EVENT_SIGNING_KEY. A zero-value or keyless signer is
// disabled: Header returns nothing and Verify accepts everything, so
// deployments can roll the key out producer-side first without breaking
// consumers.
type EventSigner struct {
	key []byte
}

// NewEventSignerFromEnv builds an EventSigner from EVENT_SIGNING_KEY.
// The signer is disabled when the variable is unset or empty.
func NewEventSignerFromEnv() *EventSigner {
	return &EventSigner{key: []byte(os.Getenv(envEventSigningKey))}
}

// NewEventSigner builds an EventSigner with an explicit key (tests).
func NewEventSigner(key []byte) *EventSigner {
	return &EventSigner{key: key}
}

// Enabled reports whether a signing key is configured.
func (s *EventSigner) Enabled() bool {
	return s != nil && len(s.key) > 0
}

func (s *EventSigner) sum(value []byte) []byte {
	mac := hmac.New(sha256.New, s.key)
	mac.Write(value)
	return mac.Sum(nil)
}

// Header returns the signature header for a message value, or (zero, false)
// when signing is disabled.
func (s *EventSigner) Header(value []byte) (kafka.Header, bool) {
	if !s.Enabled() {
		return kafka.Header{}, false
	}
	return kafka.Header{
		Key:   SignatureHeader,
		Value: []byte(hex.EncodeToString(s.sum(value))),
	}, true
}

// Sign appends the signature header to a message in place. No-op when
// signing is disabled.
func (s *EventSigner) Sign(msg *kafka.Message) {
	if h, ok := s.Header(msg.Value); ok {
		msg.Headers = append(msg.Headers, h)
	}
}

// Verify checks the message's signature header against its value. When
// signing is disabled it accepts every message. When enabled it returns
// ErrMissingSignature for unsigned messages and ErrInvalidSignature on
// mismatch — callers must skip (not process) the message in both cases.
func (s *EventSigner) Verify(msg kafka.Message) error {
	if !s.Enabled() {
		return nil
	}
	for _, h := range msg.Headers {
		if h.Key != SignatureHeader {
			continue
		}
		want, err := hex.DecodeString(string(h.Value))
		if err != nil {
			return ErrInvalidSignature
		}
		if !hmac.Equal(want, s.sum(msg.Value)) {
			return ErrInvalidSignature
		}
		return nil
	}
	return ErrMissingSignature
}
