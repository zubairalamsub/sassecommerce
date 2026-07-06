package kafka

import (
	"errors"
	"testing"

	"github.com/segmentio/kafka-go"
)

func TestEventSigner_SignAndVerify(t *testing.T) {
	s := NewEventSigner([]byte("shared-signing-key"))

	msg := kafka.Message{Value: []byte(`{"event_type":"UserRegistered"}`)}
	s.Sign(&msg)

	if len(msg.Headers) != 1 || msg.Headers[0].Key != SignatureHeader {
		t.Fatalf("expected one %s header, got %+v", SignatureHeader, msg.Headers)
	}
	if err := s.Verify(msg); err != nil {
		t.Errorf("valid message rejected: %v", err)
	}
}

func TestEventSigner_RejectsTamperedValue(t *testing.T) {
	s := NewEventSigner([]byte("shared-signing-key"))
	msg := kafka.Message{Value: []byte(`{"amount":10}`)}
	s.Sign(&msg)

	msg.Value = []byte(`{"amount":9999}`)
	if err := s.Verify(msg); !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("tampered message: got %v, want ErrInvalidSignature", err)
	}
}

func TestEventSigner_RejectsUnsigned(t *testing.T) {
	s := NewEventSigner([]byte("shared-signing-key"))
	if err := s.Verify(kafka.Message{Value: []byte("x")}); !errors.Is(err, ErrMissingSignature) {
		t.Errorf("unsigned message: got %v, want ErrMissingSignature", err)
	}
}

func TestEventSigner_RejectsWrongKey(t *testing.T) {
	producer := NewEventSigner([]byte("key-a"))
	consumer := NewEventSigner([]byte("key-b"))

	msg := kafka.Message{Value: []byte("payload")}
	producer.Sign(&msg)
	if err := consumer.Verify(msg); !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("wrong key: got %v, want ErrInvalidSignature", err)
	}
}

func TestEventSigner_DisabledAcceptsEverything(t *testing.T) {
	s := NewEventSigner(nil)

	msg := kafka.Message{Value: []byte("payload")}
	s.Sign(&msg)
	if len(msg.Headers) != 0 {
		t.Errorf("disabled signer added headers: %+v", msg.Headers)
	}
	if err := s.Verify(msg); err != nil {
		t.Errorf("disabled signer rejected message: %v", err)
	}
}
