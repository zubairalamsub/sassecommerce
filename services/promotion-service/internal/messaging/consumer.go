package messaging

import (
	"context"
	"encoding/json"

	"github.com/ecommerce/promotion-service/internal/models"
	"github.com/ecommerce/promotion-service/internal/service"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type EventConsumer struct {
	readers          []*kafka.Reader
	promotionService service.PromotionService
	logger           *logrus.Logger
}

func NewEventConsumer(brokers []string, groupID string, promotionService service.PromotionService, logger *logrus.Logger) *EventConsumer {
	topics := []string{"order-events"}
	readers := make([]*kafka.Reader, len(topics))

	for i, topic := range topics {
		readers[i] = kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		})
	}

	return &EventConsumer{
		readers:          readers,
		promotionService: promotionService,
		logger:           logger,
	}
}

func (c *EventConsumer) Start(ctx context.Context) {
	for _, reader := range c.readers {
		go c.consume(ctx, reader)
	}
	c.logger.Info("Kafka consumers started for promotion service")
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
	}).Info("Processing event")

	switch envelope.EventType {
	case "OrderPlaced", "OrderCreated":
		c.handleOrderPlaced(ctx, &envelope)
	default:
		c.logger.WithField("event_type", envelope.EventType).Debug("Ignoring unhandled event type")
	}
}

func (c *EventConsumer) handleOrderPlaced(ctx context.Context, envelope *models.EventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	tenantID, _ := payload["tenant_id"].(string)
	userID, _ := payload["user_id"].(string)
	orderTotal := 0.0
	if t, ok := payload["total"].(float64); ok {
		orderTotal = t
	} else if t, ok := payload["total_amount"].(float64); ok {
		orderTotal = t
	}

	if tenantID == "" || userID == "" {
		c.logger.Warn("OrderPlaced event missing tenant_id or user_id")
		return
	}

	// Award loyalty points: 1 point per dollar spent
	points := int(orderTotal)
	if points <= 0 {
		return
	}

	orderID, _ := payload["order_id"].(string)

	req := &models.LoyaltyPointsRequest{
		TenantID:    tenantID,
		UserID:      userID,
		Type:        models.TransactionEarn,
		Points:      points,
		OrderID:     orderID,
		Description: "Points earned from order",
	}

	if _, err := c.promotionService.ProcessLoyaltyPoints(ctx, req); err != nil {
		c.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":  userID,
			"order_id": orderID,
			"points":   points,
		}).Error("Failed to award loyalty points")
	} else {
		c.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"points":  points,
		}).Info("Loyalty points awarded for order")
	}
}
