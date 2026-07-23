package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecommerce/shipping-service/internal/models"
	"github.com/ecommerce/shipping-service/internal/service"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// EventConsumer reacts to upstream events that should produce shipments.
type EventConsumer struct {
	readers []*kafka.Reader
	service service.ShippingService
	logger  *logrus.Logger
	stop    chan struct{}

	defaultCarrier  string
	defaultFromAddr models.AddressRequest
}

// EventConsumerConfig holds dependencies + defaults for the consumer.
type EventConsumerConfig struct {
	Brokers []string
	GroupID string
	// DefaultCarrier is used when an event doesn't specify one. For Bangladesh,
	// "pathao" / "redx" / "sundarban" are common; we default to "pathao".
	DefaultCarrier string
	// DefaultFromAddress is the warehouse / merchant ship-from used when the
	// event payload omits one.
	DefaultFromAddress models.AddressRequest
}

// EventEnvelope mirrors the wire format published by the other services.
type EventEnvelope struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

func (e *EventEnvelope) GetPayload() map[string]interface{} {
	if e.Payload != nil {
		return e.Payload
	}
	return e.Data
}

// shippingTopics are the Kafka topics this consumer subscribes to.
var shippingTopics = []string{"order-events", "payment-events"}

// NewEventConsumer constructs a consumer wired to the supplied shipping service.
func NewEventConsumer(cfg EventConsumerConfig, svc service.ShippingService, logger *logrus.Logger) *EventConsumer {
	c := &EventConsumer{
		service:         svc,
		logger:          logger,
		stop:            make(chan struct{}),
		defaultCarrier:  firstNonEmpty(cfg.DefaultCarrier, "pathao"),
		defaultFromAddr: cfg.DefaultFromAddress,
	}
	for _, topic := range shippingTopics {
		c.readers = append(c.readers, kafka.NewReader(kafka.ReaderConfig{
			Brokers:        cfg.Brokers,
			Topic:          topic,
			GroupID:        cfg.GroupID,
			MinBytes:       10e3,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
		}))
	}
	return c
}

// Start launches a goroutine per topic. Cancel ctx to stop, or call Stop().
func (c *EventConsumer) Start(ctx context.Context) {
	for i, reader := range c.readers {
		go c.consumeLoop(ctx, reader, shippingTopics[i])
	}
	c.logger.Infof("Shipping consumer started for topics: %v", shippingTopics)
}

// Stop closes Kafka readers.
func (c *EventConsumer) Stop() {
	close(c.stop)
	for _, r := range c.readers {
		if err := r.Close(); err != nil {
			c.logger.WithError(err).Error("Failed to close Kafka reader")
		}
	}
	c.logger.Info("Shipping consumer stopped")
}

func (c *EventConsumer) consumeLoop(ctx context.Context, reader *kafka.Reader, topic string) {
	for {
		select {
		case <-c.stop:
			return
		case <-ctx.Done():
			return
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.WithError(err).WithField("topic", topic).Error("Failed to fetch message")
				time.Sleep(time.Second)
				continue
			}
			// Reject events that fail HMAC verification (spoofed/tampered);
			// still committed below so the poison message is not redelivered.
			if err := eventSigner.Verify(msg); err != nil {
				c.logger.WithError(err).WithFields(logrus.Fields{
					"topic":  topic,
					"offset": msg.Offset,
				}).Warn("Dropping Kafka message that failed signature verification")
			} else if err := c.processMessage(ctx, msg.Value); err != nil {
				c.logger.WithError(err).WithFields(logrus.Fields{
					"topic":  topic,
					"offset": msg.Offset,
				}).Error("Failed to process message")
			}
			if err := reader.CommitMessages(ctx, msg); err != nil {
				c.logger.WithError(err).Error("Failed to commit message")
			}
		}
	}
}

// processMessage decodes a single message and dispatches by event type.
// Exposed for unit tests.
func (c *EventConsumer) processMessage(ctx context.Context, raw []byte) error {
	var env EventEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return c.HandleEvent(ctx, &env)
}

// HandleEvent dispatches a parsed envelope to the relevant handler.
// Exported so it can be exercised directly from unit tests.
func (c *EventConsumer) HandleEvent(ctx context.Context, env *EventEnvelope) error {
	payload := env.GetPayload()
	if payload == nil {
		return nil
	}
	switch env.EventType {
	case "OrderConfirmed", "OrderPaid", "PaymentCompleted":
		return c.handleOrderReadyToShip(ctx, payload)
	case "OrderCancelled":
		return c.handleOrderCancelled(ctx, payload)
	default:
		c.logger.WithField("event_type", env.EventType).Debug("Shipping consumer ignoring event")
		return nil
	}
}

// handleOrderReadyToShip creates a shipment for a confirmed/paid order. If a
// shipment already exists for the order it's a no-op (idempotent).
func (c *EventConsumer) handleOrderReadyToShip(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	orderID, _ := payload["order_id"].(string)
	if tenantID == "" || orderID == "" {
		return nil
	}

	// Idempotency — skip if we already created a shipment for this order.
	existing, err := c.service.GetShipmentByOrderID(ctx, tenantID, orderID)
	if err == nil && existing != nil {
		c.logger.WithFields(logrus.Fields{
			"order_id": orderID, "shipment_id": existing.ID,
		}).Debug("Shipment already exists for order; skipping")
		return nil
	}

	carrier, _ := payload["carrier"].(string)
	if carrier == "" {
		carrier = c.defaultCarrier
	}
	serviceType, _ := payload["shipping_service"].(string)
	if serviceType == "" {
		serviceType = "standard"
	}

	to := extractAddress(payload["shipping_address"])
	if to.Name == "" {
		// Try a flat shape as a fallback.
		to = extractAddress(payload)
	}
	if !addressValid(to) {
		c.logger.WithField("order_id", orderID).Warn("Cannot create shipment: shipping address missing on event")
		return nil
	}

	from := c.defaultFromAddr
	if maybeFrom, ok := payload["from_address"]; ok {
		extracted := extractAddress(maybeFrom)
		if addressValid(extracted) {
			from = extracted
		}
	}
	if !addressValid(from) {
		c.logger.WithField("order_id", orderID).Warn("Cannot create shipment: no ship-from address available")
		return nil
	}

	weight := toFloat(payload["weight_oz"])
	items := extractItems(payload)

	req := &models.CreateShipmentRequest{
		TenantID:    tenantID,
		OrderID:     orderID,
		Carrier:     carrier,
		ServiceType: serviceType,
		WeightOz:    weight,
		FromAddress: from,
		ToAddress:   to,
		Items:       items,
		CreatedBy:   "system:event-consumer",
	}

	shipment, err := c.service.CreateShipment(ctx, req)
	if err != nil {
		return fmt.Errorf("create shipment for order %s: %w", orderID, err)
	}
	c.logger.WithFields(logrus.Fields{
		"order_id":        orderID,
		"shipment_id":     shipment.ID,
		"tracking_number": shipment.TrackingNumber,
	}).Info("Shipment created from upstream event")
	return nil
}

// handleOrderCancelled cancels the order's shipment if it hasn't shipped yet.
func (c *EventConsumer) handleOrderCancelled(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	orderID, _ := payload["order_id"].(string)
	if tenantID == "" || orderID == "" {
		return nil
	}

	shipment, err := c.service.GetShipmentByOrderID(ctx, tenantID, orderID)
	if err != nil || shipment == nil {
		return nil
	}
	// Don't try to cancel something already in transit.
	switch string(shipment.Status) {
	case string(models.StatusInTransit), string(models.StatusDelivered), string(models.StatusCancelled):
		return nil
	}
	if _, err := c.service.CancelShipment(ctx, tenantID, shipment.ID); err != nil {
		c.logger.WithError(err).WithField("shipment_id", shipment.ID).
			Error("Failed to cancel shipment for cancelled order")
		return err
	}
	c.logger.WithField("shipment_id", shipment.ID).Info("Cancelled shipment for cancelled order")
	return nil
}

// extractAddress pulls an AddressRequest out of arbitrary JSON. Accepts both
// nested {street, city, ...} and flat {shipping_street, shipping_city, ...} forms.
func extractAddress(v interface{}) models.AddressRequest {
	addr := models.AddressRequest{}
	m, ok := v.(map[string]interface{})
	if !ok {
		return addr
	}
	get := func(keys ...string) string {
		for _, k := range keys {
			if s, ok := m[k].(string); ok && s != "" {
				return s
			}
		}
		return ""
	}
	addr.Name = get("name", "recipient_name", "shipping_name")
	addr.Street = get("street", "address_line1", "line1", "shipping_street")
	addr.City = get("city", "shipping_city")
	addr.State = get("state", "region", "division", "shipping_state")
	addr.PostalCode = get("postal_code", "postcode", "zip", "shipping_postal_code")
	addr.Country = get("country", "shipping_country")
	if addr.Country == "" {
		addr.Country = "BD"
	}
	return addr
}

// Valid reports whether an address has the minimum fields the carrier needs.
func addressValid(a models.AddressRequest) bool {
	return a.Name != "" && a.Street != "" && a.City != "" && a.PostalCode != "" && a.Country != ""
}

func extractItems(payload map[string]interface{}) []models.ShipmentItemRequest {
	rawItems, ok := payload["items"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]models.ShipmentItemRequest, 0, len(rawItems))
	for _, ri := range rawItems {
		m, ok := ri.(map[string]interface{})
		if !ok {
			continue
		}
		item := models.ShipmentItemRequest{
			ProductID: stringField(m, "product_id", "id"),
			VariantID: stringField(m, "variant_id"),
			SKU:       stringField(m, "sku"),
			Name:      stringField(m, "name"),
			Quantity:  intField(m, "quantity", "qty"),
			WeightOz:  toFloat(m["weight_oz"]),
		}
		if item.ProductID == "" || item.Quantity <= 0 {
			continue
		}
		out = append(out, item)
	}
	return out
}

func stringField(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func intField(m map[string]interface{}, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch n := v.(type) {
			case float64:
				return int(n)
			case int:
				return n
			}
		}
	}
	return 0
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	}
	return 0
}

func firstNonEmpty(vs ...string) string {
	for _, s := range vs {
		if s != "" {
			return s
		}
	}
	return ""
}
