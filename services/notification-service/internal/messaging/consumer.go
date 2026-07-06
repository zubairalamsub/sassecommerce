// Package messaging consumes Kafka events and dispatches user-facing
// notifications. Email bodies are either rendered via the hardcoded
// templates.RenderEmailHTML helper or — when a tenant has configured an active
// NotificationTemplate for the (channel, type) pair — by substituting the
// event payload into the admin-authored template.
package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/repository"
	"github.com/ecommerce/notification-service/internal/service"
	"github.com/ecommerce/notification-service/internal/templates"
	sharedkafka "github.com/ecommerce/shared/go/pkg/kafka"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// eventSigner verifies incoming Kafka event signatures when EVENT_SIGNING_KEY is set.
var eventSigner = sharedkafka.NewEventSignerFromEnv()

// EventConsumer consumes events from Kafka and triggers notifications
type EventConsumer struct {
	readers         []*kafka.Reader
	service         service.NotificationService
	repo            repository.NotificationRepository
	logger          *logrus.Logger
	stop            chan struct{}
	frontendBaseURL string
}

// Topics consumed by the notification service
var consumedTopics = []string{
	"user-events",
	"order-events",
	"payment-events",
	"inventory-events",
	"shipping-events",
}

func NewEventConsumer(brokers []string, groupID string, svc service.NotificationService, repo repository.NotificationRepository, frontendBaseURL string, logger *logrus.Logger) *EventConsumer {
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

	if frontendBaseURL == "" {
		frontendBaseURL = "https://shop.example.com"
	}

	return &EventConsumer{
		readers:         readers,
		service:         svc,
		repo:            repo,
		logger:          logger,
		stop:            make(chan struct{}),
		frontendBaseURL: frontendBaseURL,
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

			// Reject events that fail HMAC verification (spoofed/tampered);
			// still committed below so the poison message is not redelivered.
			if err := eventSigner.Verify(msg); err != nil {
				c.logger.WithError(err).WithFields(logrus.Fields{
					"topic":  topic,
					"offset": msg.Offset,
				}).Warn("Dropping Kafka message that failed signature verification")
			} else if err := c.processMessage(ctx, msg); err != nil {
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
	case "EmailVerificationRequested":
		return c.handleEmailVerificationRequested(ctx, payload)
	case "PasswordResetRequested":
		return c.handlePasswordResetRequested(ctx, payload)
	case "OrderPlaced", "OrderCreated":
		return c.handleOrderPlaced(ctx, payload)
	case "OrderShipped":
		return c.handleOrderShipped(ctx, payload)
	case "OrderCancelled":
		return c.handleOrderCancelled(ctx, payload)
	case "ReceiptRequested":
		return c.handleReceiptRequested(ctx, payload)
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

// renderWithTemplateOrFallback looks up a tenant-specific template for the
// given (channel, type). When found AND active, it renders that template with
// the supplied vars and returns (subject, body, true). Otherwise returns
// false so the caller can use its hardcoded fallback.
//
// Errors during lookup or rendering are logged but do not block the
// notification — we return false and the caller's hardcoded copy is used.
func (c *EventConsumer) renderWithTemplateOrFallback(ctx context.Context, tenantID string, channel models.Channel, notifType models.NotificationType, vars map[string]interface{}) (string, string, bool) {
	if c.repo == nil {
		return "", "", false
	}
	t, err := c.repo.GetTemplateByType(ctx, tenantID, string(channel), string(notifType))
	if err != nil {
		c.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"channel":   channel,
			"type":      notifType,
		}).Warn("template lookup failed; falling back to hardcoded copy")
		return "", "", false
	}
	if t == nil || !t.IsActive {
		return "", "", false
	}
	// Layer the consumer-supplied vars over the type-specific defaults so
	// admins always see the same "always available" placeholders even when
	// the event payload doesn't carry them.
	merged := service.MergeSampleVars(notifType, vars)
	subject, body, err := service.RenderTemplate(t, merged)
	if err != nil {
		c.logger.WithError(err).WithField("template_id", t.ID).Error("template render failed; falling back")
		return "", "", false
	}
	return subject, body, true
}

func (c *EventConsumer) handleUserRegistered(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	userID, _ := payload["user_id"].(string)
	email, _ := payload["email"].(string)
	name, _ := payload["name"].(string)

	if tenantID == "" || userID == "" || email == "" {
		return nil
	}

	subject := "Welcome to our platform!"
	body := fmt.Sprintf("Hi %s, welcome! Your account has been created successfully.", name)

	vars := map[string]interface{}{
		"UserName":     name,
		"CustomerName": name,
	}
	if subj, b, ok := c.renderWithTemplateOrFallback(ctx, tenantID, models.ChannelEmail, models.TypeWelcome, vars); ok {
		subject, body = subj, b
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        userID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypeWelcome),
		Subject:       subject,
		Body:          body,
		Recipient:     email,
		ReferenceID:   userID,
		ReferenceType: "user",
	}

	_, err := c.service.SendNotification(ctx, req)
	return err
}

// handleEmailVerificationRequested sends a verification email built from the
// hardcoded template (or the tenant's active override). The verification URL
// is computed from frontendBaseURL + the token in the payload.
func (c *EventConsumer) handleEmailVerificationRequested(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	userID, _ := payload["user_id"].(string)
	email, _ := payload["email"].(string)
	token, _ := payload["token"].(string)
	name, _ := payload["name"].(string)

	if tenantID == "" || userID == "" || email == "" || token == "" {
		return nil
	}

	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", c.frontendBaseURL, token)
	subject := "Please verify your email"

	greeting := "Hi,"
	if name != "" {
		greeting = fmt.Sprintf("Hi %s,", name)
	}
	body := templates.RenderEmailHTML(
		"Verify your email address",
		greeting,
		"Please click the button below to verify your email and finish setting up your account.\n\n"+verifyURL,
		"Verify email",
		verifyURL,
		"This link will expire in 24 hours.",
	)

	vars := map[string]interface{}{
		"VerifyURL": verifyURL,
		"UserName":  name,
	}
	if subj, b, ok := c.renderWithTemplateOrFallback(ctx, tenantID, models.ChannelEmail, models.TypeEmailVerification, vars); ok {
		subject, body = subj, b
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        userID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypeEmailVerification),
		Subject:       subject,
		Body:          body,
		Recipient:     email,
		ReferenceID:   userID,
		ReferenceType: "user",
		Metadata: map[string]interface{}{
			"verify_url": verifyURL,
		},
	}

	_, err := c.service.SendNotification(ctx, req)
	return err
}

// handlePasswordResetRequested sends a password reset email. Like the
// verification handler it builds a URL from frontendBaseURL + the token.
func (c *EventConsumer) handlePasswordResetRequested(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	userID, _ := payload["user_id"].(string)
	email, _ := payload["email"].(string)
	token, _ := payload["token"].(string)
	name, _ := payload["name"].(string)

	if tenantID == "" || userID == "" || email == "" || token == "" {
		return nil
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", c.frontendBaseURL, token)
	subject := "Reset your password"

	greeting := "Hi,"
	if name != "" {
		greeting = fmt.Sprintf("Hi %s,", name)
	}
	body := templates.RenderEmailHTML(
		"Reset your password",
		greeting,
		"We received a request to reset your password. Click the button below to choose a new one.\n\n"+resetURL,
		"Reset password",
		resetURL,
		"If you didn't request this, you can safely ignore this email.",
	)

	vars := map[string]interface{}{
		"ResetURL": resetURL,
		"UserName": name,
	}
	if subj, b, ok := c.renderWithTemplateOrFallback(ctx, tenantID, models.ChannelEmail, models.TypePasswordReset, vars); ok {
		subject, body = subj, b
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        userID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypePasswordReset),
		Subject:       subject,
		Body:          body,
		Recipient:     email,
		ReferenceID:   userID,
		ReferenceType: "user",
		Metadata: map[string]interface{}{
			"reset_url": resetURL,
		},
	}

	_, err := c.service.SendNotification(ctx, req)
	return err
}

func (c *EventConsumer) handleOrderPlaced(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	customerID, _ := payload["customer_id"].(string)
	orderID, _ := payload["order_id"].(string)
	email, _ := payload["email"].(string)
	total, _ := payload["total"].(float64)

	if tenantID == "" || customerID == "" {
		return nil
	}

	if email == "" {
		email = c.getUserEmail(ctx, tenantID, customerID)
	}

	subject := fmt.Sprintf("Order Confirmation - %s", orderID)
	body := fmt.Sprintf("Your order %s has been placed successfully. We'll notify you when it ships.", orderID)

	vars := map[string]interface{}{
		"OrderID": orderID,
		"Total":   templates.FormatBDT(total),
	}
	if items, ok := payload["items"].([]interface{}); ok {
		vars["Items"] = items
	}
	if subj, b, ok := c.renderWithTemplateOrFallback(ctx, tenantID, models.ChannelEmail, models.TypeOrderConfirmation, vars); ok {
		subject, body = subj, b
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        customerID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypeOrderConfirmation),
		Subject:       subject,
		Body:          body,
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

	subject := fmt.Sprintf("Your order %s has shipped!", orderID)
	body := fmt.Sprintf("Your order %s has been shipped via %s.", orderID, carrier)
	if trackingNumber != "" {
		body += fmt.Sprintf(" Tracking number: %s", trackingNumber)
	}

	vars := map[string]interface{}{
		"OrderID":        orderID,
		"TrackingNumber": trackingNumber,
		"Carrier":        carrier,
	}
	if subj, b, ok := c.renderWithTemplateOrFallback(ctx, tenantID, models.ChannelEmail, models.TypeOrderShipped, vars); ok {
		subject, body = subj, b
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        customerID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypeOrderShipped),
		Subject:       subject,
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

	subject := fmt.Sprintf("Order %s Cancelled", orderID)
	body := fmt.Sprintf("Your order %s has been cancelled.", orderID)
	if reason != "" {
		body += fmt.Sprintf(" Reason: %s", reason)
	}

	vars := map[string]interface{}{
		"OrderID": orderID,
		"Reason":  reason,
	}
	if subj, b, ok := c.renderWithTemplateOrFallback(ctx, tenantID, models.ChannelEmail, models.TypeOrderCancelled, vars); ok {
		subject, body = subj, b
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        customerID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypeOrderCancelled),
		Subject:       subject,
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

	subject := fmt.Sprintf("Payment Confirmed for Order %s", orderID)
	body := fmt.Sprintf("Your payment of %s for order %s has been processed successfully.", templates.FormatBDT(amount), orderID)

	vars := map[string]interface{}{
		"OrderID": orderID,
		"Total":   templates.FormatBDT(amount),
	}
	if subj, b, ok := c.renderWithTemplateOrFallback(ctx, tenantID, models.ChannelEmail, models.TypePaymentConfirmed, vars); ok {
		subject, body = subj, b
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        customerID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypePaymentConfirmed),
		Subject:       subject,
		Body:          body,
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

	subject := fmt.Sprintf("Payment Failed for Order %s", orderID)
	body := fmt.Sprintf("We were unable to process your payment for order %s. Please update your payment method.", orderID)

	vars := map[string]interface{}{
		"OrderID": orderID,
	}
	if subj, b, ok := c.renderWithTemplateOrFallback(ctx, tenantID, models.ChannelEmail, models.TypePaymentFailed, vars); ok {
		subject, body = subj, b
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        customerID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypePaymentFailed),
		Subject:       subject,
		Body:          body,
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

	subject := fmt.Sprintf("Low Stock Alert - SKU: %s", sku)
	body := fmt.Sprintf("Product %s (SKU: %s) is running low. Current quantity: %.0f units.", productID, sku, currentQty)

	vars := map[string]interface{}{
		"ProductName":     productID,
		"SKU":             sku,
		"CurrentQuantity": currentQty,
	}
	if subj, b, ok := c.renderWithTemplateOrFallback(ctx, tenantID, models.ChannelEmail, models.TypeStockAlert, vars); ok {
		subject, body = subj, b
	}

	// Stock alerts go to the tenant admin, not a specific customer
	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        "admin",
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypeStockAlert),
		Subject:       subject,
		Body:          body,
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

// handleReceiptRequested dispatches the POS receipt email triggered when a
// cashier hits "Email to customer" in the Instant Sell flow. The event is
// published by the order service with the order summary inline (items,
// totals, payment method) so the notification service doesn't need to
// re-fetch the order to compose the email.
//
// The payload shape is intentionally tolerant — any of the following may be
// missing without panicking:
//   - customer_email (we fall back to the user's stored preference)
//   - items         (the email simply omits the line-item table)
//   - subtotal/tax  (the email omits unsupplied lines)
//   - store_name    (defaults to the templates package's tenant placeholder)
//
// The only truly required fields are `tenant_id` and `order_id`.
func (c *EventConsumer) handleReceiptRequested(ctx context.Context, payload map[string]interface{}) error {
	tenantID, _ := payload["tenant_id"].(string)
	orderID, _ := payload["order_id"].(string)
	customerID, _ := payload["customer_id"].(string)
	customerEmail, _ := payload["customer_email"].(string)
	customerName, _ := payload["customer_name"].(string)
	storeName, _ := payload["store_name"].(string)
	paymentMethod, _ := payload["payment_method"].(string)
	currency, _ := payload["currency"].(string)

	if tenantID == "" || orderID == "" {
		return nil
	}

	// Resolve recipient: payload override > stored preference > fail-soft skip.
	if customerEmail == "" && customerID != "" {
		customerEmail = c.getUserEmail(ctx, tenantID, customerID)
	}
	if customerEmail == "" {
		// No way to deliver. Don't error — the order was still recorded; the
		// cashier can print a paper copy instead.
		c.logger.WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"order_id":  orderID,
		}).Info("Skipping ReceiptRequested: no recipient email")
		return nil
	}

	if storeName == "" {
		storeName = templates.DefaultTenantName
	}
	if currency == "" {
		currency = "BDT"
	}
	if paymentMethod == "" {
		paymentMethod = "—"
	}

	subtotal := readFloat(payload["subtotal"])
	discount := readFloat(payload["discount"])
	tax := readFloat(payload["tax"])
	shipping := readFloat(payload["shipping_cost"])
	total := readFloat(payload["total"])

	// items: tolerate either a JSON array of objects or a missing entry.
	items := readItems(payload["items"])

	// Body content — a compact text-then-table description. Currency is
	// formatted via FormatBDT for the BDT case (the templates package's
	// helper) and falls back to the bare number otherwise.
	formatMoney := func(v float64) string {
		if currency == "BDT" || currency == "" {
			return templates.FormatBDT(v)
		}
		return fmt.Sprintf("%.2f %s", v, currency)
	}

	greeting := "Hi,"
	if customerName != "" {
		greeting = fmt.Sprintf("Hi %s,", customerName)
	}

	var lineList strings.Builder
	if len(items) > 0 {
		lineList.WriteString("Here is your receipt:\n\n")
		for _, it := range items {
			lineList.WriteString(fmt.Sprintf("%d × %s — %s\n", it.Quantity, it.Name, formatMoney(it.LineTotal())))
		}
		lineList.WriteString("\n")
	} else {
		lineList.WriteString("Here is your receipt.\n\n")
	}
	lineList.WriteString(fmt.Sprintf("Subtotal: %s\n", formatMoney(subtotal)))
	if discount > 0 {
		lineList.WriteString(fmt.Sprintf("Discount: -%s\n", formatMoney(discount)))
	}
	if shipping > 0 {
		lineList.WriteString(fmt.Sprintf("Shipping: %s\n", formatMoney(shipping)))
	}
	if tax > 0 {
		lineList.WriteString(fmt.Sprintf("Tax: %s\n", formatMoney(tax)))
	}
	lineList.WriteString(fmt.Sprintf("Total: %s\n\nPayment: %s", formatMoney(total), paymentMethod))

	subject := fmt.Sprintf("Your receipt from %s", storeName)
	body := templates.RenderEmailHTML(
		fmt.Sprintf("Your receipt from %s", storeName),
		greeting,
		lineList.String(),
		"",
		"",
		fmt.Sprintf("Receipt #%s · Thank you for shopping!", shortOrderRef(orderID)),
	)

	// Allow admins to override with a tenant template if one exists.
	vars := map[string]interface{}{
		"OrderID":       orderID,
		"StoreName":     storeName,
		"CustomerName":  customerName,
		"PaymentMethod": paymentMethod,
		"Total":         formatMoney(total),
		"Subtotal":      formatMoney(subtotal),
	}
	if subj, b, ok := c.renderWithTemplateOrFallback(ctx, tenantID, models.ChannelEmail, models.TypeReceipt, vars); ok {
		subject, body = subj, b
	}

	userID := customerID
	if userID == "" {
		userID = "guest"
	}

	req := &models.SendNotificationRequest{
		TenantID:      tenantID,
		UserID:        userID,
		Channel:       string(models.ChannelEmail),
		Type:          string(models.TypeReceipt),
		Subject:       subject,
		Body:          body,
		Recipient:     customerEmail,
		ReferenceID:   orderID,
		ReferenceType: "order",
		Metadata: map[string]interface{}{
			"store_name":     storeName,
			"payment_method": paymentMethod,
			"total":          total,
		},
	}

	_, err := c.service.SendNotification(ctx, req)
	return err
}

// receiptItem mirrors the per-line shape carried in the ReceiptRequested
// payload. Quantities and prices arrive as JSON numbers (so float64 after
// decoding); the LineTotal helper centralises the multiplication.
type receiptItem struct {
	Name      string
	SKU       string
	Quantity  int
	UnitPrice float64
	Total     float64
}

func (i receiptItem) LineTotal() float64 {
	if i.Total > 0 {
		return i.Total
	}
	return i.UnitPrice * float64(i.Quantity)
}

// readFloat coerces numeric-looking interface values to a float64.
// JSON unmarshalling lands every numeric as float64, but tests may pass
// raw ints — accept both to keep the handler robust.
func readFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

func readInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case float32:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

// readItems parses the items array out of a payload map. It tolerates
// missing fields (returns zero-valued entries) and unexpected shapes
// (returns an empty slice) so the handler never panics on malformed input.
func readItems(v interface{}) []receiptItem {
	raw, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]receiptItem, 0, len(raw))
	for _, entry := range raw {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		sku, _ := m["sku"].(string)
		out = append(out, receiptItem{
			Name:      name,
			SKU:       sku,
			Quantity:  readInt(m["quantity"]),
			UnitPrice: readFloat(m["unit_price"]),
			Total:     readFloat(m["total_price"]),
		})
	}
	return out
}

// shortOrderRef trims an order id (typically a UUID) to the last 8
// alnum characters in upper-case so the receipt subject line stays brief.
func shortOrderRef(orderID string) string {
	cleaned := make([]byte, 0, len(orderID))
	for i := 0; i < len(orderID); i++ {
		ch := orderID[i]
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			if ch >= 'a' && ch <= 'z' {
				ch -= 'a' - 'A'
			}
			cleaned = append(cleaned, ch)
		}
	}
	if len(cleaned) > 8 {
		cleaned = cleaned[len(cleaned)-8:]
	}
	if len(cleaned) == 0 {
		return "RECEIPT"
	}
	return string(cleaned)
}
