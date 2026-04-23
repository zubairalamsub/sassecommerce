package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/service"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// EventConsumer consumes events from Kafka and triggers notifications
type EventConsumer struct {
	readers []*kafka.Reader
	service service.NotificationService
	logger  *logrus.Logger
	stop    chan struct{}
}

// Topics consumed by the notification service
var consumedTopics = []string{
	"user-events",
	"order-events",
	"payment-events",
	"inventory-events",
	"shipping-events",
}

func NewEventConsumer(brokers []string, groupID string, svc service.NotificationService, logger *logrus.Logger) *EventConsumer {
	var readers []*kafka.Reader
	for _, topic := range consumedTopics {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       10e3,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
		})
		readers = append(readers, reader)
	}

	return &EventConsumer{
		readers: readers,
		service: svc,
		logger:  logger,
		stop:    make(chan struct{}),
	}
}

func (c *EventConsumer) Start(ctx context.Context) {
	for i, reader := range c.readers {
		go c.consumeLoop(ctx, reader, consumedTopics[i])
	}
	c.logger.Info("Kafka event consumers started for topics: ", consumedTopics)
}

func (c *EventConsumer) Stop() {
	close(c.stop)
	for _, reader := range c.readers {
		if err := reader.Close(); err != nil {
			c.logger.WithError(err).Error("Failed to close Kafka reader")
		}
	}
	c.logger.Info("Kafka event consumers stopped")
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

			if err := c.processMessage(ctx, msg); err != nil {
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

func (c *EventConsumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var envelope models.EventEnvelope
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"event_type": envelope.EventType,
		"event_id":   envelope.EventID,
	}).Debug("Processing event")

	return c.handleEvent(ctx, &envelope)
}

func (c *EventConsumer) handleEvent(ctx context.Context, envelope *models.EventEnvelope) error {
	payload := envelope.GetPayload()
	if payload == nil {
		return fmt.Errorf("event has no payload")
	}

	switch envelope.EventType {
	case "UserRegistered", "UserCreated":
		return c.handleUserRegistered(ctx, payload)
	case "OrderPlaced", "OrderCreated":
		return c.handleOrderPlaced(ctx, payload)
	case "OrderShipped":
		return c.handleOrderShipped(ctx, payload)
	case "OrderCancelled":
		return c.handleOrderCancelled(ctx, payload)
	case "PaymentCompleted":
		return c.handlePaymentCompleted(ctx, payload)
	case "PaymentFailed":
		return c.handlePaymentFailed(ctx, payload)
	case "StockLevelLow":
		return c.handleStockLevelLow(ctx, payload)
	default:
		c.logger.WithField("event_type", envelope.EventType).Debug("Ignoring unhandled event type")
		return nil
	}
}

func (c *EventConsumer) handleUserRegistered(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	userID, _ := payload["user_id"].(string)
	email, _ := payload["email"].(string)
	name, _ := payload["name"].(string)

	if tenantID == "" || userID == "" || email == "" {
		return nil
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        userID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypeWelcome),
		Subject:       "Welcome to our platform!",
		Body:          fmt.Sprintf("Hi %s, welcome! Your account has been created successfully.", name),
		Recipient:     email,
		ReferenceID:   userID,
		ReferenceType: "user",
	}

	_, err := c.service.SendNotification(ctx, req)
	return err
}

func (c *EventConsumer) handleOrderPlaced(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	customerID, _ := payload["customer_id"].(string)
	orderID, _ := payload["order_id"].(string)
	email, _ := payload["email"].(string)

	if tenantID == "" || customerID == "" {
		return nil
	}

	if email == "" {
		email = c.getUserEmail(ctx, tenantID, customerID)
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        customerID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypeOrderConfirmation),
		Subject:       fmt.Sprintf("Order Confirmation - %s", orderID),
		Body:          fmt.Sprintf("Your order %s has been placed successfully. We'll notify you when it ships.", orderID),
		Recipient:     email,
		ReferenceID:   orderID,
		ReferenceType: "order",
	}

	_, err := c.service.SendNotification(ctx, req)
	return err
}

func (c *EventConsumer) handleOrderShipped(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	customerID, _ := payload["customer_id"].(string)
	orderID, _ := payload["order_id"].(string)
	trackingNumber, _ := payload["tracking_number"].(string)
	carrier, _ := payload["carrier"].(string)
	email, _ := payload["email"].(string)

	if tenantID == "" || customerID == "" {
		return nil
	}

	if email == "" {
		email = c.getUserEmail(ctx, tenantID, customerID)
	}

	body := fmt.Sprintf("Your order %s has been shipped via %s.", orderID, carrier)
	if trackingNumber != "" {
		body += fmt.Sprintf(" Tracking number: %s", trackingNumber)
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        customerID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypeOrderShipped),
		Subject:       fmt.Sprintf("Your order %s has shipped!", orderID),
		Body:          body,
		Recipient:     email,
		ReferenceID:   orderID,
		ReferenceType: "order",
		Metadata: map[string]interface{}{
			"tracking_number": trackingNumber,
			"carrier":         carrier,
		},
	}

	_, err := c.service.SendNotification(ctx, req)
	return err
}

func (c *EventConsumer) handleOrderCancelled(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	customerID, _ := payload["customer_id"].(string)
	orderID, _ := payload["order_id"].(string)
	reason, _ := payload["reason"].(string)
	email, _ := payload["email"].(string)

	if tenantID == "" || customerID == "" {
		return nil
	}

	if email == "" {
		email = c.getUserEmail(ctx, tenantID, customerID)
	}

	body := fmt.Sprintf("Your order %s has been cancelled.", orderID)
	if reason != "" {
		body += fmt.Sprintf(" Reason: %s", reason)
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        customerID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypeOrderCancelled),
		Subject:       fmt.Sprintf("Order %s Cancelled", orderID),
		Body:          body,
		Recipient:     email,
		ReferenceID:   orderID,
		ReferenceType: "order",
	}

	_, err := c.service.SendNotification(ctx, req)
	return err
}

func (c *EventConsumer) handlePaymentCompleted(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	customerID, _ := payload["customer_id"].(string)
	paymentID, _ := payload["payment_id"].(string)
	orderID, _ := payload["order_id"].(string)
	amount, _ := payload["amount"].(float64)
	email, _ := payload["email"].(string)

	if tenantID == "" || customerID == "" {
		return nil
	}

	if email == "" {
		email = c.getUserEmail(ctx, tenantID, customerID)
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        customerID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypePaymentConfirmed),
		Subject:       fmt.Sprintf("Payment Confirmed for Order %s", orderID),
		Body:          fmt.Sprintf("Your payment of $%.2f for order %s has been processed successfully.", amount, orderID),
		Recipient:     email,
		ReferenceID:   paymentID,
		ReferenceType: "payment",
	}

	_, err := c.service.SendNotification(ctx, req)
	return err
}

func (c *EventConsumer) handlePaymentFailed(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	customerID, _ := payload["customer_id"].(string)
	orderID, _ := payload["order_id"].(string)
	email, _ := payload["email"].(string)

	if tenantID == "" || customerID == "" {
		return nil
	}

	if email == "" {
		email = c.getUserEmail(ctx, tenantID, customerID)
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        customerID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypePaymentFailed),
		Subject:       fmt.Sprintf("Payment Failed for Order %s", orderID),
		Body:          fmt.Sprintf("We were unable to process your payment for order %s. Please update your payment method.", orderID),
		Recipient:     email,
		ReferenceID:   orderID,
		ReferenceType: "payment",
	}

	_, err := c.service.SendNotification(ctx, req)
	return err
}

func (c *EventConsumer) handleStockLevelLow(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	productID, _ := payload["product_id"].(string)
	sku, _ := payload["sku"].(string)
	currentQty, _ := payload["current_quantity"].(float64)

	if tenantID == "" || productID == "" {
		return nil
	}

	// Stock alerts go to the tenant admin, not a specific customer
	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        "admin",
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypeStockAlert),
		Subject:       fmt.Sprintf("Low Stock Alert - SKU: %s", sku),
		Body:          fmt.Sprintf("Product %s (SKU: %s) is running low. Current quantity: %.0f units.", productID, sku, currentQty),
		Recipient:     "admin@tenant.local",
		ReferenceID:   productID,
		ReferenceType: "product",
	}

	_, err := c.service.SendNotification(ctx, req)
	return err
}

func (c *EventConsumer) getUserEmail(ctx context.Context, tenantID, userID string) string {
	pref, err := c.service.GetPreference(ctx, tenantID, userID)
	if err == nil && pref != nil && pref.Email != "" {
		return pref.Email
	}
	return fmt.Sprintf("%s@placeholder.local", userID)
}
