package messaging

import (
	"context"
	"encoding/json"
	"strings"

	sharedkafka "github.com/ecommerce/shared/go/pkg/kafka"
	"github.com/ecommerce/tenant-service/internal/models"
	"github.com/ecommerce/tenant-service/internal/service"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// eventSigner verifies incoming Kafka event signatures when EVENT_SIGNING_KEY is set.
var eventSigner = sharedkafka.NewEventSignerFromEnv()

// ServiceEvent is the common envelope used by all services when publishing Kafka events.
// Version can be a string ("1.0.0") or a number (1) depending on the service.
type ServiceEvent struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp string                 `json:"timestamp"`
	Version   interface{}            `json:"version"`
	Payload   map[string]interface{} `json:"payload"`
	// .NET services sometimes use "data" instead of "payload"
	Data map[string]interface{} `json:"data,omitempty"`
	// Order service uses these fields
	AggregateID   string `json:"aggregate_id,omitempty"`
	AggregateType string `json:"aggregate_type,omitempty"`
}

func (e *ServiceEvent) getPayload() map[string]interface{} {
	if len(e.Payload) > 0 {
		return e.Payload
	}
	return e.Data
}

// topicToResource maps Kafka topic names to audit resource types.
var topicToResource = map[string]models.AuditResource{
	"product-events":   models.ResourceProduct,
	"order-events":     models.ResourceOrder,
	"tenant-events":    models.ResourceTenant,
	"user-events":      "user",
	"payment-events":   "payment",
	"inventory-events": "inventory",
	"shipping-events":  "shipping",
}

// eventTypeToAction maps event type suffixes to audit actions.
func eventTypeToAction(eventType string) models.AuditAction {
	lower := strings.ToLower(eventType)
	switch {
	case strings.Contains(lower, "created") || strings.Contains(lower, "placed"):
		return models.ActionCreate
	case strings.Contains(lower, "updated") || strings.Contains(lower, "confirmed") ||
		strings.Contains(lower, "shipped") || strings.Contains(lower, "delivered") ||
		strings.Contains(lower, "reserved") || strings.Contains(lower, "changed") ||
		strings.Contains(lower, "processed") || strings.Contains(lower, "completed"):
		return models.ActionUpdate
	case strings.Contains(lower, "deleted") || strings.Contains(lower, "cancelled") ||
		strings.Contains(lower, "released") || strings.Contains(lower, "failed"):
		return models.ActionDelete
	case strings.Contains(lower, "login"):
		return models.ActionLogin
	case strings.Contains(lower, "logout"):
		return models.ActionLogout
	default:
		return models.ActionUpdate
	}
}

// securityEventActions maps known security-relevant event types to their
// canonical audit action label. When an event matches, the consumer uses
// this label instead of the generic CREATE/UPDATE/DELETE classification so
// the audit log (and the security-events filter in the admin UI) can pick
// these out with a clean prefix-based filter.
//
// Labels follow a "<resource>.<area>.<verb>" convention so the admin UI
// can filter by prefix (e.g. "user.login.*" → all login activity).
var securityEventActions = map[string]string{
	// user-service auth events
	"LoginSucceeded":             "user.login.succeeded",
	"LoginFailed":                "user.login.failed",
	"PasswordChanged":            "user.password.changed",
	"PasswordReset":              "user.password.reset",
	"PasswordResetRequested":     "user.password.reset_requested",
	"EmailVerificationRequested": "user.email.verification_requested",
	"EmailVerified":              "user.email.verified",
	"TwoFactorEnabled":           "user.2fa.enabled",
	"TwoFactorDisabled":          "user.2fa.disabled",
	"UserRoleChanged":            "user.role.changed",
	"UserSuspended":              "user.suspended.changed",
	"UserReactivated":            "user.reactivated.changed",
	"UserStatusChanged":          "user.status.changed",
	"SessionRevoked":             "user.session.revoked",
	"AllSessionsRevoked":         "user.session.revoked_all",
	// tenant-service events
	"TenantSuspended":     "tenant.suspended.changed",
	"TenantReactivated":   "tenant.reactivated.changed",
	"TenantConfigChanged": "tenant.config.changed",
	// product-service events
	"ProductPriceChanged": "product.price.changed",
	// order-service events
	"OrderRefunded": "payment.refund.created",
}

// AuditEventConsumer listens to Kafka topics from all services and creates
// centralised audit log entries in the tenant service database.
type AuditEventConsumer struct {
	readers      []*kafka.Reader
	auditService service.AuditService
	logger       *logrus.Logger
}

// Topics consumed for the centralised audit log.
var auditTopics = []string{
	"product-events",
	"order-events",
	"user-events",
	"payment-events",
	"inventory-events",
	"shipping-events",
}

// NewAuditEventConsumer creates one Kafka reader per topic.
func NewAuditEventConsumer(brokers []string, auditService service.AuditService, logger *logrus.Logger) *AuditEventConsumer {
	var readers []*kafka.Reader
	for _, topic := range auditTopics {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  "tenant-audit-consumer",
			MinBytes: 1,
			MaxBytes: 10e6, // 10 MB
		})
		readers = append(readers, r)
	}

	return &AuditEventConsumer{
		readers:      readers,
		auditService: auditService,
		logger:       logger,
	}
}

// Start launches a goroutine per topic that continuously reads messages.
func (c *AuditEventConsumer) Start(ctx context.Context) {
	for _, r := range c.readers {
		go c.consume(ctx, r)
	}
	c.logger.WithField("topics", auditTopics).Info("Audit event consumer started")
}

func (c *AuditEventConsumer) consume(ctx context.Context, reader *kafka.Reader) {
	topic := reader.Config().Topic
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // shutting down
			}
			c.logger.WithError(err).WithField("topic", topic).Warn("Failed to fetch Kafka message")
			continue
		}

		// Reject events that fail HMAC verification (spoofed/tampered);
		// still committed below so the poison message is not redelivered.
		if err := eventSigner.Verify(msg); err != nil {
			c.logger.WithError(err).WithField("topic", topic).Warn("Dropping Kafka message that failed signature verification")
		} else {
			c.handleMessage(ctx, topic, msg)
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			c.logger.WithError(err).WithField("topic", topic).Warn("Failed to commit Kafka message")
		}
	}
}

func (c *AuditEventConsumer) handleMessage(ctx context.Context, topic string, msg kafka.Message) {
	var event ServiceEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.logger.WithError(err).WithField("topic", topic).Warn("Failed to unmarshal event")
		return
	}

	payload := event.getPayload()
	resource := topicToResource[topic]
	action := eventTypeToAction(event.EventType)

	// Extract tenant ID — skip events that can't be attributed to a tenant
	tenantID, _ := payload["tenant_id"].(string)
	if tenantID == "" {
		c.logger.WithFields(logrus.Fields{
			"topic":      topic,
			"event_type": event.EventType,
		}).Debug("Skipping event without tenant_id")
		return
	}

	resourceID, _ := payload["product_id"].(string)
	if resourceID == "" {
		resourceID, _ = payload["order_id"].(string)
	}
	if resourceID == "" {
		resourceID, _ = payload["user_id"].(string)
	}
	if resourceID == "" {
		resourceID, _ = payload["id"].(string)
	}
	userID, _ := payload["updated_by"].(string)

	// Separate old_values from current payload for before/after diff
	var oldValue string
	if oldVals, ok := payload["old_values"]; ok {
		if oldBytes, err := json.Marshal(oldVals); err == nil {
			oldValue = string(oldBytes)
		}
		delete(payload, "old_values") // Don't include in new_value
	}

	// Remove internal fields from the "after" snapshot
	cleanPayload := make(map[string]interface{})
	for k, v := range payload {
		if k != "tenant_id" && k != "product_id" && k != "order_id" && k != "user_id" && k != "updated_by" {
			cleanPayload[k] = v
		}
	}
	newValue, _ := json.Marshal(cleanPayload)

	// Extract resource ID from order service aggregate pattern
	if resourceID == "" && event.AggregateID != "" {
		resourceID = event.AggregateID
	}

	// Build metadata with source service info
	meta, _ := json.Marshal(map[string]interface{}{
		"source_topic":  topic,
		"event_id":      event.EventID,
		"event_version": event.Version,
	})

	req := &models.CreateAuditLogRequest{
		TenantID:     tenantID,
		UserID:       userID,
		Action:       action,
		Resource:     resource,
		ResourceID:   resourceID,
		Method:       "KAFKA",
		Path:         topic + "/" + event.EventType,
		OldValue:     oldValue,
		NewValue:     string(newValue),
		Metadata:     string(meta),
		ResponseCode: 200,
	}

	if err := c.auditService.CreateAuditLog(ctx, req); err != nil {
		c.logger.WithError(err).WithField("event_type", event.EventType).Warn("Failed to create audit log from event")
	}
}

// Close shuts down all readers.
func (c *AuditEventConsumer) Close() {
	for _, r := range c.readers {
		if err := r.Close(); err != nil {
			c.logger.WithError(err).Warn("Failed to close Kafka reader")
		}
	}
}
