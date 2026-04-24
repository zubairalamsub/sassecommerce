package messaging

import (
	"context"
	"encoding/json"

	"github.com/ecommerce/product-service/internal/repository"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// EventEnvelope represents a Kafka event envelope
type EventEnvelope struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp string                 `json:"timestamp"`
	Version   string                 `json:"version,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
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

		c.handleMessage(ctx, msg)
	}
}

func (c *EventConsumer) handleMessage(ctx context.Context, msg kafka.Message) {
	var envelope EventEnvelope
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		c.logger.WithError(err).Error("Failed to unmarshal event envelope")
		return
	}

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
