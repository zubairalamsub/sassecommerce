package messaging

import (
	"context"
	"encoding/json"

	"github.com/ecommerce/vendor-service/internal/models"
	"github.com/ecommerce/vendor-service/internal/service"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type EventConsumer struct {
	readers       []*kafka.Reader
	vendorService service.VendorService
	logger        *logrus.Logger
}

func NewEventConsumer(brokers []string, groupID string, vendorService service.VendorService, logger *logrus.Logger) *EventConsumer {
	topics := []string{"order-events", "product-events"}
	readers := make([]*kafka.Reader, len(topics))

	for i, topic := range topics {
		readers[i] = kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		})
	}

	return &EventConsumer{
		readers:       readers,
		vendorService: vendorService,
		logger:        logger,
	}
}

func (c *EventConsumer) Start(ctx context.Context) {
	for _, reader := range c.readers {
		go c.consume(ctx, reader)
	}
	c.logger.Info("Kafka consumers started for vendor service")
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
	case "ProductCreated":
		c.handleProductCreated(ctx, &envelope)
	default:
		c.logger.WithField("event_type", envelope.EventType).Debug("Ignoring unhandled event type")
	}
}

func (c *EventConsumer) handleOrderPlaced(ctx context.Context, envelope *models.EventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	vendorID, _ := payload["vendor_id"].(string)
	tenantID, _ := payload["tenant_id"].(string)
	orderID, _ := payload["order_id"].(string)
	amount := 0.0
	if a, ok := payload["total"].(float64); ok {
		amount = a
	} else if a, ok := payload["total_amount"].(float64); ok {
		amount = a
	}

	if vendorID == "" || orderID == "" {
		c.logger.Debug("OrderPlaced event missing vendor_id or order_id, skipping")
		return
	}

	if err := c.vendorService.RecordOrder(ctx, vendorID, tenantID, orderID, amount); err != nil {
		c.logger.WithError(err).WithFields(logrus.Fields{
			"vendor_id": vendorID,
			"order_id":  orderID,
		}).Error("Failed to record vendor order")
	} else {
		c.logger.WithFields(logrus.Fields{
			"vendor_id": vendorID,
			"order_id":  orderID,
			"amount":    amount,
		}).Info("Vendor order recorded")
	}
}

func (c *EventConsumer) handleProductCreated(ctx context.Context, envelope *models.EventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	vendorID, _ := payload["vendor_id"].(string)
	productID, _ := payload["product_id"].(string)

	if vendorID == "" {
		return
	}

	c.logger.WithFields(logrus.Fields{
		"vendor_id":  vendorID,
		"product_id": productID,
	}).Info("Product created by vendor - pending approval workflow")
}
