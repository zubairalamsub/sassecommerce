package service

import (
	"testing"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func newTestLogger() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

// === Email Provider Tests ===

func TestEmailProvider_Send_Success(t *testing.T) {
	provider := NewSimulatedEmailProvider(newTestLogger())

	notification := &models.Notification{
		ID:        "notif-1",
		Recipient: "user@example.com",
		Subject:   "Test Email",
		Body:      "Hello!",
	}

	result, err := provider.Send(notification)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "simulated-sendgrid", result.ProviderName)
	assert.Contains(t, result.MessageID, "email_")
}

func TestEmailProvider_Send_EmptyRecipient(t *testing.T) {
	provider := NewSimulatedEmailProvider(newTestLogger())

	notification := &models.Notification{
		ID:        "notif-1",
		Recipient: "",
		Subject:   "Test",
	}

	result, err := provider.Send(notification)

	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "recipient email is required")
}

func TestEmailProvider_Channel(t *testing.T) {
	provider := NewSimulatedEmailProvider(newTestLogger())
	assert.Equal(t, models.ChannelEmail, provider.Channel())
}

// === SMS Provider Tests ===

func TestSMSProvider_Send_Success(t *testing.T) {
	provider := NewSimulatedSMSProvider(newTestLogger())

	notification := &models.Notification{
		ID:        "notif-1",
		Recipient: "+1234567890",
		Body:      "Your order shipped!",
	}

	result, err := provider.Send(notification)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "simulated-twilio", result.ProviderName)
	assert.Contains(t, result.MessageID, "sms_")
}

func TestSMSProvider_Send_EmptyRecipient(t *testing.T) {
	provider := NewSimulatedSMSProvider(newTestLogger())

	notification := &models.Notification{
		ID:        "notif-1",
		Recipient: "",
	}

	result, err := provider.Send(notification)

	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "recipient phone number is required")
}

func TestSMSProvider_Channel(t *testing.T) {
	provider := NewSimulatedSMSProvider(newTestLogger())
	assert.Equal(t, models.ChannelSMS, provider.Channel())
}

// === Push Provider Tests ===

func TestPushProvider_Send_Success(t *testing.T) {
	provider := NewSimulatedPushProvider(newTestLogger())

	notification := &models.Notification{
		ID:        "notif-1",
		Recipient: "device-token-abc123xyz456",
		Subject:   "New Order",
		Body:      "You have a new order",
	}

	result, err := provider.Send(notification)

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "simulated-fcm", result.ProviderName)
	assert.Contains(t, result.MessageID, "push_")
}

func TestPushProvider_Send_EmptyRecipient(t *testing.T) {
	provider := NewSimulatedPushProvider(newTestLogger())

	notification := &models.Notification{
		ID:        "notif-1",
		Recipient: "",
	}

	result, err := provider.Send(notification)

	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "device token is required")
}

func TestPushProvider_Channel(t *testing.T) {
	provider := NewSimulatedPushProvider(newTestLogger())
	assert.Equal(t, models.ChannelPush, provider.Channel())
}

// === Utility Tests ===

func TestGenerateMessageID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		id := generateMessageID("test")
		assert.False(t, seen[id], "duplicate message ID: %s", id)
		seen[id] = true
		assert.Contains(t, id, "test_")
	}
}

func TestGenerateMessageID_Prefix(t *testing.T) {
	emailID := generateMessageID("email")
	smsID := generateMessageID("sms")
	pushID := generateMessageID("push")

	assert.Contains(t, emailID, "email_")
	assert.Contains(t, smsID, "sms_")
	assert.Contains(t, pushID, "push_")
}
