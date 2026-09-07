package messaging

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ecommerce/product-service/internal/repository"
	sharedkafka "github.com/ecommerce/shared/go/pkg/kafka"
	"github.com/ecommerce/shared/go/pkg/metrics"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// metricsService labels this service's event metrics, kept as a constant
// so it cannot drift from the dashboard queries that group by it.
const metricsService = "product-service"

// EventEnvelope represents a Kafka event envelope
type EventEnvelope struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Timestamp string `json:"timestamp"`
	// sharedkafka.EventVersion, not string. Producers on this platform disagree
	// about the JSON type of `version`: order-service emits a bare number,
	// everything else a quoted string. Against a `string` field the numeric
	// shape makes json.Unmarshal return an UnmarshalTypeError -- and the
	// handler returns on that error without looking at the fields the decoder
	// did populate, so the event is dropped and the log says "unmarshal
	// error" rather than naming the event lost. That is the failure that took
	// out order confirmation, shipped, cancelled, payment and receipt emails
	// once already. Nothing here reads this field; it is declared only so an
	// unexpected shape cannot reject the envelope.
	Version sharedkafka.EventVersion `json:"version,omitempty"`
	Payload map[string]interface{}   `json:"payload,omitempty"`
	Data    map[string]interface{}   `json:"data,omitempty"`
}

// GetPayload returns the payload, falling back to Data if Payload is nil
func (e *EventEnvelope) GetPayload() map[string]interface{} {
	if e.Payload != nil {
		return e.Payload
	}
	return e.Data
}

// EventConsumer consumes inventory events to keep product stock in sync
type EventConsumer struct {
	reader      *kafka.Reader
	productRepo repository.ProductRepository
	logger      *logrus.Logger
}

// NewEventConsumer creates a new Kafka event consumer for inventory events
func NewEventConsumer(brokers []string, groupID string, productRepo repository.ProductRepository, logger *logrus.Logger) *EventConsumer {
	// Pre-create the zero-valued series for this topic, so a consumer that
	// has never dropped anything reads as 0 rather than as no data.
	metrics.InitTopic(metricsService, "inventory-events")

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   "inventory-events",
		GroupID: groupID,
	})

	return &EventConsumer{
		reader:      reader,
		productRepo: productRepo,
		logger:      logger,
	}
}

// Start begins consuming messages
func (c *EventConsumer) Start(ctx context.Context) {
	go c.consume(ctx)
	c.logger.Info("Kafka consumer started for product service (inventory-events)")
}

// Stop closes the consumer
func (c *EventConsumer) Stop() {
	if err := c.reader.Close(); err != nil {
		c.logger.WithError(err).Error("Failed to close Kafka reader")
	}
}

func (c *EventConsumer) consume(ctx context.Context) {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.WithError(err).Error("Failed to read Kafka message")
			continue
		}

		// Reject events that fail HMAC verification (spoofed/tampered)
		if err := eventSigner.Verify(msg); err != nil {
			metrics.EventDropped(metricsService, msg.Topic, "", metrics.ReasonSignatureInvalid)
			c.logger.WithError(err).WithField("topic", msg.Topic).Warn("Dropping Kafka message that failed signature verification")
			continue
		}

		c.handleMessage(ctx, msg)
	}
}

func (c *EventConsumer) handleMessage(ctx context.Context, msg kafka.Message) {
	start := time.Now()

	var envelope EventEnvelope
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		// The counter this exists for: the offset is committed regardless, so
		// an envelope that will not decode leaves lag at zero while the stock
		// update it carried is simply lost.
		metrics.EventDropped(metricsService, msg.Topic, "", metrics.ReasonDecodeError)
		c.logger.WithError(err).Error("Failed to unmarshal event envelope")
		return
	}

	defer func() {
		metrics.EventConsumed(metricsService, msg.Topic, envelope.EventType, time.Since(start))
	}()

	c.logger.WithFields(logrus.Fields{
		"event_type": envelope.EventType,
		"event_id":   envelope.EventID,
	}).Info("Processing inventory event")

	switch envelope.EventType {
	case "InventoryUpdated", "StockLevelChanged":
		c.handleInventoryUpdated(ctx, &envelope)
	default:
		c.logger.WithField("event_type", envelope.EventType).Debug("Ignoring unhandled event type")
	}
}

func (c *EventConsumer) handleInventoryUpdated(ctx context.Context, envelope *EventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		c.logger.Warn("Inventory event has nil payload")
		return
	}

	tenantID, _ := payload["tenant_id"].(string)
	productID, _ := payload["product_id"].(string)
	if productID == "" {
		c.logger.Warn("Inventory event missing product_id")
		return
	}

	quantity := 0
	if q, ok := payload["quantity"].(float64); ok {
		quantity = int(q)
	}

	inStock := quantity > 0
	if v, ok := payload["in_stock"].(bool); ok {
		inStock = v
	}

	if err := c.productRepo.UpdateStock(ctx, tenantID, productID, quantity, inStock); err != nil {
		c.logger.WithError(err).WithFields(logrus.Fields{
			"product_id": productID,
			"tenant_id":  tenantID,
		}).Error("Failed to update product stock")
		return
	}

	c.logger.WithFields(logrus.Fields{
		"product_id": productID,
		"tenant_id":  tenantID,
		"quantity":   quantity,
		"in_stock":   inStock,
	}).Info("Product stock updated from inventory event")
}
