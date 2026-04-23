package models

import (
	"time"
)

// Channel represents a notification delivery channel
type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelSMS   Channel = "sms"
	ChannelPush  Channel = "push"
)

// NotificationStatus represents delivery status
type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusSent      NotificationStatus = "sent"
	StatusDelivered NotificationStatus = "delivered"
	StatusFailed    NotificationStatus = "failed"
	StatusRead      NotificationStatus = "read"
)

// NotificationType categorizes notifications
type NotificationType string

const (
	TypeOrderConfirmation  NotificationType = "order_confirmation"
	TypeOrderShipped       NotificationType = "order_shipped"
	TypeOrderDelivered     NotificationType = "order_delivered"
	TypeOrderCancelled     NotificationType = "order_cancelled"
	TypePaymentConfirmed   NotificationType = "payment_confirmed"
	TypePaymentFailed      NotificationType = "payment_failed"
	TypeWelcome            NotificationType = "welcome"
	TypePasswordReset      NotificationType = "password_reset"
	TypeStockAlert         NotificationType = "stock_alert"
	TypePromotion          NotificationType = "promotion"
)

// Notification represents a notification record
type Notification struct {
	ID          string             `bson:"_id,omitempty" json:"id"`
	TenantID    string             `bson:"tenant_id" json:"tenant_id"`
	UserID      string             `bson:"user_id" json:"user_id"`
	Channel     Channel            `bson:"channel" json:"channel"`
	Type        NotificationType   `bson:"type" json:"type"`
	Status      NotificationStatus `bson:"status" json:"status"`

	// Content
	Subject string `bson:"subject" json:"subject"`
	Body    string `bson:"body" json:"body"`

	// Recipient info
	Recipient string `bson:"recipient" json:"recipient"` // email address, phone number, or device token

	// Reference to the triggering entity
	ReferenceID   string `bson:"reference_id,omitempty" json:"reference_id,omitempty"`
	ReferenceType string `bson:"reference_type,omitempty" json:"reference_type,omitempty"` // order, payment, user, etc.

	// Provider response
	ProviderName      string `bson:"provider_name,omitempty" json:"provider_name,omitempty"`
	ProviderMessageID string `bson:"provider_message_id,omitempty" json:"provider_message_id,omitempty"`
	FailureReason     string `bson:"failure_reason,omitempty" json:"failure_reason,omitempty"`

	// Metadata
	Metadata map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Timestamps
	ReadAt    *time.Time `bson:"read_at,omitempty" json:"read_at,omitempty"`
	SentAt    *time.Time `bson:"sent_at,omitempty" json:"sent_at,omitempty"`
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
}

// UserPreference stores notification preferences per user
type UserPreference struct {
	ID       string `bson:"_id,omitempty" json:"id"`
	TenantID string `bson:"tenant_id" json:"tenant_id"`
	UserID   string `bson:"user_id" json:"user_id"`

	// Channel preferences
	EmailEnabled bool `bson:"email_enabled" json:"email_enabled"`
	SMSEnabled   bool `bson:"sms_enabled" json:"sms_enabled"`
	PushEnabled  bool `bson:"push_enabled" json:"push_enabled"`

	// Notification type opt-outs
	OptedOut []NotificationType `bson:"opted_out,omitempty" json:"opted_out,omitempty"`

	// Contact info
	Email       string `bson:"email,omitempty" json:"email,omitempty"`
	PhoneNumber string `bson:"phone_number,omitempty" json:"phone_number,omitempty"`
	DeviceToken string `bson:"device_token,omitempty" json:"device_token,omitempty"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// NotificationTemplate stores reusable notification templates
type NotificationTemplate struct {
	ID       string           `bson:"_id,omitempty" json:"id"`
	TenantID string           `bson:"tenant_id" json:"tenant_id"`
	Type     NotificationType `bson:"type" json:"type"`
	Channel  Channel          `bson:"channel" json:"channel"`
	Name     string           `bson:"name" json:"name"`

	SubjectTemplate string `bson:"subject_template" json:"subject_template"`
	BodyTemplate    string `bson:"body_template" json:"body_template"`

	IsActive  bool      `bson:"is_active" json:"is_active"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// === Request DTOs ===

type SendNotificationRequest struct {
	TenantID      string                 `json:"tenant_id" binding:"required"`
	UserID        string                 `json:"user_id" binding:"required"`
	Channel       string                 `json:"channel" binding:"required"`
	Type          string                 `json:"type" binding:"required"`
	Subject       string                 `json:"subject" binding:"required"`
	Body          string                 `json:"body" binding:"required"`
	Recipient     string                 `json:"recipient" binding:"required"`
	ReferenceID   string                 `json:"reference_id,omitempty"`
	ReferenceType string                 `json:"reference_type,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type UpdatePreferenceRequest struct {
	EmailEnabled *bool              `json:"email_enabled,omitempty"`
	SMSEnabled   *bool              `json:"sms_enabled,omitempty"`
	PushEnabled  *bool              `json:"push_enabled,omitempty"`
	OptedOut     []NotificationType `json:"opted_out,omitempty"`
	Email        string             `json:"email,omitempty"`
	PhoneNumber  string             `json:"phone_number,omitempty"`
	DeviceToken  string             `json:"device_token,omitempty"`
}

// === Response DTOs ===

type NotificationResponse struct {
	ID            string             `json:"id"`
	TenantID      string             `json:"tenant_id"`
	UserID        string             `json:"user_id"`
	Channel       Channel            `json:"channel"`
	Type          NotificationType   `json:"type"`
	Status        NotificationStatus `json:"status"`
	Subject       string             `json:"subject"`
	Body          string             `json:"body"`
	Recipient     string             `json:"recipient"`
	ReferenceID   string             `json:"reference_id,omitempty"`
	ReferenceType string             `json:"reference_type,omitempty"`
	FailureReason string             `json:"failure_reason,omitempty"`
	ReadAt        *time.Time         `json:"read_at,omitempty"`
	SentAt        *time.Time         `json:"sent_at,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
}

type UserPreferenceResponse struct {
	ID           string             `json:"id"`
	TenantID     string             `json:"tenant_id"`
	UserID       string             `json:"user_id"`
	EmailEnabled bool               `json:"email_enabled"`
	SMSEnabled   bool               `json:"sms_enabled"`
	PushEnabled  bool               `json:"push_enabled"`
	OptedOut     []NotificationType `json:"opted_out"`
	Email        string             `json:"email,omitempty"`
	PhoneNumber  string             `json:"phone_number,omitempty"`
	DeviceToken  string             `json:"device_token,omitempty"`
}

// EventEnvelope is the Kafka event wire format
type EventEnvelope struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"`
	AggregateID   string                 `json:"aggregate_id,omitempty"`
	AggregateType string                 `json:"aggregate_type,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Version       string                 `json:"version,omitempty"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
	Data          map[string]interface{} `json:"data,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// GetPayload returns whichever field is populated (payload or data)
func (e *EventEnvelope) GetPayload() map[string]interface{} {
	if e.Payload != nil {
		return e.Payload
	}
	return e.Data
}
