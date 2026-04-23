package messaging

import (
	"context"
	"encoding/json"

	"github.com/ecommerce/recommendation-service/internal/models"
	"github.com/ecommerce/recommendation-service/internal/service"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type EventConsumer struct {
	readers              []*kafka.Reader
	recommendationService service.RecommendationService
	logger               *logrus.Logger
}

func NewEventConsumer(brokers []string, groupID string, svc service.RecommendationService, logger *logrus.Logger) *EventConsumer {
	topics := []string{"order-events", "product-events", "cart-events"}
	readers := make([]*kafka.Reader, len(topics))

	for i, topic := range topics {
		readers[i] = kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		})
	}

	return &EventConsumer{
		readers:              readers,
		recommendationService: svc,
		logger:               logger,
	}
}

func (c *EventConsumer) Start(ctx context.Context) {
	for _, reader := range c.readers {
		go c.consume(ctx, reader)
	}
	c.logger.Info("Kafka consumers started for recommendation service")
}

func (c *EventConsumer) Stop() {
	for _, reader := range c.readers {
		if err := reader.Close(); err != nil {
			c.logger.WithError(err).Error("Failed to close Kafka reader")
		}
	}
}

func (c *EventConsumer) consume(ctx context.Context, reader *kafka.Reader) {
	for {
		msg, err := reader.ReadMessage(ctx)
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
	var envelope models.EventEnvelope
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		c.logger.WithError(err).Error("Failed to unmarshal event envelope")
		return
	}

	c.logger.WithFields(logrus.Fields{
		"event_type": envelope.EventType,
		"event_id":   envelope.EventID,
		"topic":      msg.Topic,
	}).Info("Processing recommendation event")

	switch envelope.EventType {
	case "OrderPlaced", "OrderCreated":
		c.handleOrderEvent(ctx, &envelope)
	case "ProductViewed":
		c.handleProductViewed(ctx, &envelope)
	case "CartUpdated":
		c.handleCartEvent(ctx, &envelope)
	default:
		c.logger.WithField("event_type", envelope.EventType).Debug("Ignoring unhandled event type")
	}
}

func (c *EventConsumer) handleOrderEvent(ctx context.Context, envelope *models.EventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	tenantID, _ := payload["tenant_id"].(string)
	userID, _ := payload["user_id"].(string)

	if tenantID == "" || userID == "" {
		return
	}

	// Record purchase interactions for each item in the order
	if items, ok := payload["items"].([]interface{}); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				productID, _ := itemMap["product_id"].(string)
				if productID != "" {
					if err := c.recommendationService.RecordInteraction(ctx, tenantID, userID, productID, models.InteractionPurchase); err != nil {
						c.logger.WithError(err).Error("Failed to record purchase interaction")
					}
				}
			}
		}
	}

	// Also handle single product_id for simple order events
	if productID, ok := payload["product_id"].(string); ok && productID != "" {
		if err := c.recommendationService.RecordInteraction(ctx, tenantID, userID, productID, models.InteractionPurchase); err != nil {
			c.logger.WithError(err).Error("Failed to record purchase interaction")
		}
	}
}

func (c *EventConsumer) handleProductViewed(ctx context.Context, envelope *models.EventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	tenantID, _ := payload["tenant_id"].(string)
	userID, _ := payload["user_id"].(string)
	productID, _ := payload["product_id"].(string)

	if tenantID == "" || userID == "" || productID == "" {
		return
	}

	if err := c.recommendationService.RecordInteraction(ctx, tenantID, userID, productID, models.InteractionView); err != nil {
		c.logger.WithError(err).Error("Failed to record view interaction")
	}
}

func (c *EventConsumer) handleCartEvent(ctx context.Context, envelope *models.EventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	tenantID, _ := payload["tenant_id"].(string)
	userID, _ := payload["user_id"].(string)

	if tenantID == "" || userID == "" {
		return
	}

	if items, ok := payload["items"].([]interface{}); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				productID, _ := itemMap["product_id"].(string)
				if productID != "" {
					if err := c.recommendationService.RecordInteraction(ctx, tenantID, userID, productID, models.InteractionCart); err != nil {
						c.logger.WithError(err).Error("Failed to record cart interaction")
					}
				}
			}
		}
	}
}
