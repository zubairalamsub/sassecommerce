package messaging

import (
	"context"
	"encoding/json"

	"github.com/ecommerce/cart-service/internal/models"
	"github.com/ecommerce/cart-service/internal/service"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type EventConsumer struct {
	readers     []*kafka.Reader
	cartService service.CartService
	logger      *logrus.Logger
}

func NewEventConsumer(brokers []string, groupID string, cartService service.CartService, logger *logrus.Logger) *EventConsumer {
	topics := []string{"product-events", "price-events"}
	readers := make([]*kafka.Reader, len(topics))

	for i, topic := range topics {
		readers[i] = kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		})
	}

	return &EventConsumer{
		readers:     readers,
		cartService: cartService,
		logger:      logger,
	}
}

func (c *EventConsumer) Start(ctx context.Context) {
	for _, reader := range c.readers {
		go c.consume(ctx, reader)
	}
	c.logger.Info("Kafka consumers started for cart service")
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
	case "PriceChanged", "ProductPriceUpdated":
		c.handlePriceChanged(ctx, &envelope)
	case "ProductDeleted":
		c.handleProductDeleted(ctx, &envelope)
	default:
		c.logger.WithField("event_type", envelope.EventType).Debug("Ignoring unhandled event type")
	}
}

func (c *EventConsumer) handlePriceChanged(ctx context.Context, envelope *models.EventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	productID, _ := payload["product_id"].(string)
	newPrice, _ := payload["new_price"].(float64)

	if productID == "" || newPrice <= 0 {
		c.logger.Warn("Invalid PriceChanged event: missing product_id or new_price")
		return
	}

	if err := c.cartService.UpdateProductPrice(ctx, productID, newPrice); err != nil {
		c.logger.WithError(err).WithField("product_id", productID).Error("Failed to update product price in carts")
	} else {
		c.logger.WithFields(logrus.Fields{
			"product_id": productID,
			"new_price":  newPrice,
		}).Info("Updated product price in carts")
	}
}

func (c *EventConsumer) handleProductDeleted(ctx context.Context, envelope *models.EventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	productID, _ := payload["product_id"].(string)
	if productID == "" {
		c.logger.Warn("Invalid ProductDeleted event: missing product_id")
		return
	}

	if err := c.cartService.RemoveProduct(ctx, productID); err != nil {
		c.logger.WithError(err).WithField("product_id", productID).Error("Failed to remove product from carts")
	} else {
		c.logger.WithField("product_id", productID).Info("Removed product from carts")
	}
}
