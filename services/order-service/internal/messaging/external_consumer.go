package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// OrderCommand represents a command that can be dispatched to the order service.
// This interface breaks the import cycle between messaging and commands packages.
type OrderCommand interface {
	GetAggregateID() string
}

// CommandDispatcher dispatches commands to the order service
type CommandDispatcher interface {
	Handle(cmd OrderCommand) error
}

// ExternalEventConsumer consumes events from external services
// (payment-events, inventory-events, shipping-events)
// and translates them into order commands.
type ExternalEventConsumer struct {
	readers    []*kafka.Reader
	dispatcher CommandDispatcher
	logger     *zap.Logger
	stopChan   chan struct{}
}

// NewExternalEventConsumer creates a consumer for external service events
func NewExternalEventConsumer(
	brokers []string,
	groupID string,
	dispatcher CommandDispatcher,
	logger *zap.Logger,
) *ExternalEventConsumer {
	topics := []string{"payment-events", "inventory-events", "shipping-events"}
	readers := make([]*kafka.Reader, len(topics))

	for i, topic := range topics {
		readers[i] = kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       10e3, // 10KB
			MaxBytes:       10e6, // 10MB
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
			Logger:         kafka.LoggerFunc(func(msg string, args ...interface{}) {}),
		})
	}

	return &ExternalEventConsumer{
		readers:    readers,
		dispatcher: dispatcher,
		logger:     logger,
		stopChan:   make(chan struct{}),
	}
}

// Start starts consuming events from all external topics
func (c *ExternalEventConsumer) Start(ctx context.Context) error {
	c.logger.Info("Starting external event consumer",
		zap.Int("topic_count", len(c.readers)),
	)

	for _, reader := range c.readers {
		go c.consumeLoop(ctx, reader)
	}

	return nil
}

// consumeLoop continuously consumes messages from a single reader
func (c *ExternalEventConsumer) consumeLoop(ctx context.Context, reader *kafka.Reader) {
	topic := reader.Config().Topic
	c.logger.Info("Consumer loop started", zap.String("topic", topic))

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		default:
			message, err := reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				c.logger.Error("Failed to fetch message",
					zap.String("topic", topic),
					zap.Error(err),
				)
				time.Sleep(time.Second)
				continue
			}

			c.handleMessage(ctx, topic, message)

			if err := reader.CommitMessages(ctx, message); err != nil {
				c.logger.Error("Failed to commit message", zap.Error(err))
			}
		}
	}
}

// handleMessage processes a single message from an external topic
func (c *ExternalEventConsumer) handleMessage(ctx context.Context, topic string, msg kafka.Message) {
	var envelope ExternalEventEnvelope
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		c.logger.Error("Failed to unmarshal external event",
			zap.String("topic", topic),
			zap.Error(err),
		)
		return
	}

	c.logger.Debug("Processing external event",
		zap.String("topic", topic),
		zap.String("event_type", envelope.EventType),
		zap.String("event_id", envelope.EventID),
	)

	switch topic {
	case "payment-events":
		c.handlePaymentEvent(ctx, &envelope)
	case "inventory-events":
		c.handleInventoryEvent(ctx, &envelope)
	case "shipping-events":
		c.handleShippingEvent(ctx, &envelope)
	}
}

// handlePaymentEvent processes payment service events
func (c *ExternalEventConsumer) handlePaymentEvent(ctx context.Context, envelope *ExternalEventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	orderID := getPayloadString(payload, "order_id")
	if orderID == "" {
		c.logger.Warn("Payment event missing order_id")
		return
	}

	switch envelope.EventType {
	case "PaymentCompleted":
		c.logger.Info("Payment completed for order", zap.String("order_id", orderID))
		// Confirm the order since payment succeeded
		cmd := ConfirmOrderCmd{
			OrderID:     orderID,
			ConfirmedBy: "payment-service",
		}
		if err := c.dispatcher.Handle(cmd); err != nil {
			c.logger.Error("Failed to confirm order after payment",
				zap.String("order_id", orderID),
				zap.Error(err),
			)
		}

	case "PaymentFailed":
		reason := getPayloadString(payload, "reason")
		if reason == "" {
			reason = "Payment failed"
		}
		c.logger.Warn("Payment failed for order",
			zap.String("order_id", orderID),
			zap.String("reason", reason),
		)
		// Cancel the order since payment failed
		cmd := CancelOrderCmd{
			OrderID:     orderID,
			Reason:      fmt.Sprintf("Payment failed: %s", reason),
			CancelledBy: "payment-service",
		}
		if err := c.dispatcher.Handle(cmd); err != nil {
			c.logger.Error("Failed to cancel order after payment failure",
				zap.String("order_id", orderID),
				zap.Error(err),
			)
		}
	}
}

// handleInventoryEvent processes inventory service events
func (c *ExternalEventConsumer) handleInventoryEvent(ctx context.Context, envelope *ExternalEventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	orderID := getPayloadString(payload, "order_id")

	switch envelope.EventType {
	case "StockReserved":
		if orderID == "" {
			return
		}
		c.logger.Info("Stock reserved for order", zap.String("order_id", orderID))
		// Stock is reserved — order saga can proceed to payment step
		// (In a full saga implementation this would signal the saga coordinator)

	case "StockReservationFailed":
		if orderID == "" {
			return
		}
		reason := getPayloadString(payload, "reason")
		if reason == "" {
			reason = "Insufficient stock"
		}
		c.logger.Warn("Stock reservation failed for order",
			zap.String("order_id", orderID),
			zap.String("reason", reason),
		)
		cmd := CancelOrderCmd{
			OrderID:     orderID,
			Reason:      fmt.Sprintf("Stock reservation failed: %s", reason),
			CancelledBy: "inventory-service",
		}
		if err := c.dispatcher.Handle(cmd); err != nil {
			c.logger.Error("Failed to cancel order after stock reservation failure",
				zap.String("order_id", orderID),
				zap.Error(err),
			)
		}

	case "StockLevelLow":
		productID := getPayloadString(payload, "product_id")
		c.logger.Warn("Low stock alert",
			zap.String("product_id", productID),
		)
	}
}

// handleShippingEvent processes shipping service events
func (c *ExternalEventConsumer) handleShippingEvent(ctx context.Context, envelope *ExternalEventEnvelope) {
	payload := envelope.GetPayload()
	if payload == nil {
		return
	}

	orderID := getPayloadString(payload, "order_id")
	if orderID == "" {
		c.logger.Warn("Shipping event missing order_id")
		return
	}

	switch envelope.EventType {
	case "ShipmentCreated", "OrderShipped":
		trackingNumber := getPayloadString(payload, "tracking_number")
		carrier := getPayloadString(payload, "carrier")
		c.logger.Info("Shipment created for order",
			zap.String("order_id", orderID),
			zap.String("tracking_number", trackingNumber),
			zap.String("carrier", carrier),
		)
		cmd := ShipOrderCmd{
			OrderID:        orderID,
			TrackingNumber: trackingNumber,
			Carrier:        carrier,
			ShippedBy:      "shipping-service",
		}
		if err := c.dispatcher.Handle(cmd); err != nil {
			c.logger.Error("Failed to mark order as shipped",
				zap.String("order_id", orderID),
				zap.Error(err),
			)
		}

	case "ShipmentDelivered", "OrderDelivered":
		receivedBy := getPayloadString(payload, "received_by")
		if receivedBy == "" {
			receivedBy = "customer"
		}
		c.logger.Info("Shipment delivered for order",
			zap.String("order_id", orderID),
		)
		cmd := DeliverOrderCmd{
			OrderID:    orderID,
			ReceivedBy: receivedBy,
		}
		if err := c.dispatcher.Handle(cmd); err != nil {
			c.logger.Error("Failed to mark order as delivered",
				zap.String("order_id", orderID),
				zap.Error(err),
			)
		}
	}
}

// Stop stops the external event consumer
func (c *ExternalEventConsumer) Stop() error {
	close(c.stopChan)

	for _, reader := range c.readers {
		if err := reader.Close(); err != nil {
			c.logger.Error("Failed to close external Kafka reader", zap.Error(err))
		}
	}

	c.logger.Info("External event consumer stopped")
	return nil
}

// ExternalEventEnvelope represents an event from an external service
type ExternalEventEnvelope struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp string                 `json:"timestamp"`
	Version   string                 `json:"version,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// GetPayload returns the payload, falling back to Data if Payload is nil
func (e *ExternalEventEnvelope) GetPayload() map[string]interface{} {
	if e.Payload != nil {
		return e.Payload
	}
	return e.Data
}

// ConfirmOrderCmd is a command to confirm an order (matches ConfirmOrderCmd)
type ConfirmOrderCmd struct {
	OrderID     string
	ConfirmedBy string
}

func (c ConfirmOrderCmd) GetAggregateID() string { return c.OrderID }

// CancelOrderCmd is a command to cancel an order (matches CancelOrderCmd)
type CancelOrderCmd struct {
	OrderID     string
	Reason      string
	CancelledBy string
}

func (c CancelOrderCmd) GetAggregateID() string { return c.OrderID }

// ShipOrderCmd is a command to mark an order as shipped (matches ShipOrderCmd)
type ShipOrderCmd struct {
	OrderID        string
	TrackingNumber string
	Carrier        string
	ShippedBy      string
}

func (c ShipOrderCmd) GetAggregateID() string { return c.OrderID }

// DeliverOrderCmd is a command to mark an order as delivered (matches DeliverOrderCmd)
type DeliverOrderCmd struct {
	OrderID    string
	ReceivedBy string
}

func (c DeliverOrderCmd) GetAggregateID() string { return c.OrderID }

// getPayloadString safely extracts a string from a payload map
func getPayloadString(payload map[string]interface{}, key string) string {
	if val, ok := payload[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}
