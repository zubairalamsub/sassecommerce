package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// NotificationPublisher publishes lightweight, non-domain notification events
// to Kafka. The order service uses it for events that don't belong on the
// order aggregate's event stream — currently just `ReceiptRequested`, which
// is fired when a POS cashier hits "Email to customer" on the success screen.
//
// We deliberately keep this separate from EventPublisher (which handles
// domain events) so we don't have to thread a fake `events.Event` through
// the aggregate just to ship a stateless notification request.
type NotificationPublisher interface {
	// PublishReceiptRequested publishes a `ReceiptRequested` event onto the
	// order-events topic. The notification-service consumer picks it up and
	// renders/sends the email.
	PublishReceiptRequested(ctx context.Context, payload ReceiptRequestedPayload) error
	Close() error
}

// ReceiptRequestedPayload is the wire shape consumed by notification-service
// (see internal/messaging/consumer.go::handleReceiptRequested). All fields are
// optional except TenantID and OrderID. Currency defaults to BDT in the
// consumer.
type ReceiptRequestedPayload struct {
	TenantID      string                  `json:"tenant_id"`
	OrderID       string                  `json:"order_id"`
	CustomerID    string                  `json:"customer_id,omitempty"`
	CustomerEmail string                  `json:"customer_email,omitempty"`
	CustomerName  string                  `json:"customer_name,omitempty"`
	StoreName     string                  `json:"store_name,omitempty"`
	PaymentMethod string                  `json:"payment_method,omitempty"`
	Currency      string                  `json:"currency,omitempty"`
	Subtotal      float64                 `json:"subtotal,omitempty"`
	Discount      float64                 `json:"discount,omitempty"`
	Tax           float64                 `json:"tax,omitempty"`
	ShippingCost  float64                 `json:"shipping_cost,omitempty"`
	Total         float64                 `json:"total,omitempty"`
	Items         []ReceiptRequestedItem  `json:"items,omitempty"`
}

// ReceiptRequestedItem is a single line item carried in the payload.
type ReceiptRequestedItem struct {
	Name       string  `json:"name"`
	SKU        string  `json:"sku,omitempty"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price,omitempty"`
}

// KafkaNotificationPublisher is the production implementation backed by a
// `kafka.Writer`. Tests can substitute a fake satisfying NotificationPublisher.
type KafkaNotificationPublisher struct {
	writer *kafka.Writer
	logger *zap.Logger
	topic  string
}

// NewKafkaNotificationPublisher constructs a writer targeting the given topic.
// We use the same `order-events` topic the existing EventPublisher writes to
// so the notification-service consumer (already subscribed) picks it up
// without an additional reader.
func NewKafkaNotificationPublisher(brokers []string, topic string, logger *zap.Logger) *KafkaNotificationPublisher {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
		MaxAttempts:  3,
		// Silence the underlying segmentio logger; we surface our own.
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {}),
	}
	return &KafkaNotificationPublisher{
		writer: writer,
		logger: logger,
		topic:  topic,
	}
}

// PublishReceiptRequested marshals the payload into the same envelope shape
// used by EventEnvelope (in kafka_publisher.go) so notification-service can
// decode it identically to a domain event.
func (p *KafkaNotificationPublisher) PublishReceiptRequested(ctx context.Context, payload ReceiptRequestedPayload) error {
	if payload.TenantID == "" || payload.OrderID == "" {
		return fmt.Errorf("tenant_id and order_id are required")
	}

	envelope := EventEnvelope{
		EventID:       uuid.New().String(),
		EventType:     "ReceiptRequested",
		AggregateID:   payload.OrderID,
		AggregateType: "Order",
		Version:       1,
		Timestamp:     time.Now().UTC(),
		Data:          payload,
	}

	value, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal receipt event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(payload.OrderID),
		Value: value,
		Headers: []kafka.Header{
			{Key: "event-type", Value: []byte("ReceiptRequested")},
			{Key: "aggregate-id", Value: []byte(payload.OrderID)},
		},
		Time: envelope.Timestamp,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.logger.Error("Failed to publish ReceiptRequested",
			zap.String("order_id", payload.OrderID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish receipt event: %w", err)
	}

	p.logger.Info("Published ReceiptRequested",
		zap.String("order_id", payload.OrderID),
		zap.String("topic", p.topic),
	)
	return nil
}

// Close releases the underlying Kafka writer.
func (p *KafkaNotificationPublisher) Close() error {
	return p.writer.Close()
}
