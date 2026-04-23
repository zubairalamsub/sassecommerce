package messaging

import (
	"context"
	"encoding/json"

	"github.com/ecommerce/analytics-service/internal/models"
	"github.com/ecommerce/analytics-service/internal/service"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type EventConsumer struct {
	readers          []*kafka.Reader
	analyticsService service.AnalyticsService
	logger           *logrus.Logger
}

func NewEventConsumer(brokers []string, groupID string, analyticsService service.AnalyticsService, logger *logrus.Logger) *EventConsumer {
	topics := []string{"order-events", "product-events", "vendor-events", "cart-events"}
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
		analyticsService: analyticsService,
		logger:           logger,
	}
}

func (c *EventConsumer) Start(ctx context.Context) {
	for _, reader := range c.readers {
		go c.consume(ctx, reader)
	}
	c.logger.Info("Kafka consumers started for analytics service")
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
	}).Info("Processing analytics event")

	switch envelope.EventType {
	case "OrderPlaced", "OrderCreated":
		c.handleOrderEvent(ctx, &envelope)
	case "ProductCreated", "ProductUpdated":
		c.handleProductEvent(ctx, &envelope, "product_listed")
	case "ProductDeleted":
		c.handleProductEvent(ctx, &envelope, "product_removed")
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
	orderID, _ := payload["order_id"].(string)
	userID, _ := payload["user_id"].(string)
	vendorID, _ := payload["vendor_id"].(string)
	channel, _ := payload["channel"].(string)

	amount := 0.0
	if a, ok := payload["total"].(float64); ok {
		amount = a
	} else if a, ok := payload["total_amount"].(float64); ok {
		amount = a
	}

	if orderID == "" || tenantID == "" {
		c.logger.Debug("Order event missing order_id or tenant_id, skipping")
		return
	}

	if err := c.analyticsService.RecordSale(ctx, tenantID, orderID, userID, vendorID, amount, channel); err != nil {
		c.logger.WithError(err).WithFields(logrus.Fields{
			"order_id":  orderID,
			"tenant_id": tenantID,
		}).Error("Failed to record sale for analytics")
	} else {
		c.logger.WithFields(logrus.Fields{
			"order_id": orderID,
			"amount":   amount,
		}).Info("Sale recorded for analytics")
	}
}

func (c *EventConsumer) handleProductEvent(ctx context.Context, envelope *models.EventEnvelope, eventType string) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	tenantID, _ := payload["tenant_id"].(string)
	productID, _ := payload["product_id"].(string)

	if productID == "" || tenantID == "" {
		return
	}

	quantity := 0
	if q, ok := payload["quantity"].(float64); ok {
		quantity = int(q)
	}

	revenue := 0.0
	if r, ok := payload["price"].(float64); ok {
		revenue = r * float64(quantity)
	}

	if err := c.analyticsService.RecordProductActivity(ctx, tenantID, productID, eventType, quantity, revenue); err != nil {
		c.logger.WithError(err).Error("Failed to record product activity")
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

	if err := c.analyticsService.RecordCustomerActivity(ctx, tenantID, userID, "cart_update", "", 0); err != nil {
		c.logger.WithError(err).Error("Failed to record cart activity")
	}
}
