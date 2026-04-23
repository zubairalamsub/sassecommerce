package messaging

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ecommerce/search-service/internal/models"
	"github.com/ecommerce/search-service/internal/service"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type EventConsumer struct {
	readers       []*kafka.Reader
	searchService service.SearchService
	logger        *logrus.Logger
}

func NewEventConsumer(brokers []string, groupID string, searchService service.SearchService, logger *logrus.Logger) *EventConsumer {
	topics := []string{"product-events", "inventory-events"}
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
		searchService: searchService,
		logger:        logger,
	}
}

func (c *EventConsumer) Start(ctx context.Context) {
	for _, reader := range c.readers {
		go c.consume(ctx, reader)
	}
	c.logger.Info("Kafka consumers started for search service")
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
	case "ProductCreated":
		c.handleProductCreated(ctx, &envelope)
	case "ProductUpdated":
		c.handleProductUpdated(ctx, &envelope)
	case "ProductDeleted":
		c.handleProductDeleted(ctx, &envelope)
	case "InventoryUpdated", "StockLevelChanged":
		c.handleInventoryUpdated(ctx, &envelope)
	default:
		c.logger.WithField("event_type", envelope.EventType).Debug("Ignoring unhandled event type")
	}
}

func (c *EventConsumer) handleProductCreated(ctx context.Context, envelope *models.EventEnvelope) {
	product := c.extractProductDocument(envelope)
	if product == nil {
		return
	}

	if err := c.searchService.IndexProduct(ctx, product); err != nil {
		c.logger.WithError(err).WithField("product_id", product.ID).Error("Failed to index new product")
	}
}

func (c *EventConsumer) handleProductUpdated(ctx context.Context, envelope *models.EventEnvelope) {
	product := c.extractProductDocument(envelope)
	if product == nil {
		return
	}

	if err := c.searchService.IndexProduct(ctx, product); err != nil {
		c.logger.WithError(err).WithField("product_id", product.ID).Error("Failed to update product in index")
	}
}

func (c *EventConsumer) handleProductDeleted(ctx context.Context, envelope *models.EventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	productID, _ := payload["product_id"].(string)
	if productID == "" {
		if id, ok := payload["id"].(string); ok {
			productID = id
		}
	}

	if productID == "" {
		c.logger.Warn("ProductDeleted event missing product_id")
		return
	}

	if err := c.searchService.DeleteProduct(ctx, productID); err != nil {
		c.logger.WithError(err).WithField("product_id", productID).Error("Failed to delete product from index")
	}
}

func (c *EventConsumer) handleInventoryUpdated(ctx context.Context, envelope *models.EventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	productID, _ := payload["product_id"].(string)
	if productID == "" {
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

	if err := c.searchService.UpdateProductStock(ctx, productID, quantity, inStock); err != nil {
		c.logger.WithError(err).WithField("product_id", productID).Error("Failed to update product stock in index")
	}
}

func (c *EventConsumer) extractProductDocument(envelope *models.EventEnvelope) *models.ProductDocument {
	payload := envelope.GetPayload()
	if payload == nil {
		c.logger.Warn("Event has nil payload")
		return nil
	}

	// Marshal payload back to JSON and unmarshal into ProductDocument
	data, err := json.Marshal(payload)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal event payload")
		return nil
	}

	var product models.ProductDocument
	if err := json.Unmarshal(data, &product); err != nil {
		c.logger.WithError(err).Error("Failed to unmarshal product from event payload")
		return nil
	}

	if product.ID == "" {
		c.logger.Warn("Product document from event has no ID")
		return nil
	}

	if product.CreatedAt.IsZero() {
		product.CreatedAt = time.Now().UTC()
	}
	if product.UpdatedAt.IsZero() {
		product.UpdatedAt = time.Now().UTC()
	}
	if product.Status == "" {
		product.Status = "active"
	}

	return &product
}
