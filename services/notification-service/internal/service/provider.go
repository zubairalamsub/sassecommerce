package service

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

// NotificationProvider abstracts the delivery of notifications via a specific channel
type NotificationProvider interface {
	Send(notification *models.Notification) (*ProviderResult, error)
	Channel() models.Channel
}

// ProviderResult holds the result of a send operation
type ProviderResult struct {
	ProviderName string
	MessageID    string
	Success      bool
	Error        string
}

// === Simulated Email Provider ===

type SimulatedEmailProvider struct {
	logger *logrus.Logger
}

func NewSimulatedEmailProvider(logger *logrus.Logger) NotificationProvider {
	return &SimulatedEmailProvider{logger: logger}
}

func (p *SimulatedEmailProvider) Channel() models.Channel {
	return models.ChannelEmail
}

func (p *SimulatedEmailProvider) Send(notification *models.Notification) (*ProviderResult, error) {
	if notification.Recipient == "" {
		return &ProviderResult{
			ProviderName: "simulated-sendgrid",
			Success:      false,
			Error:        "recipient email is required",
		}, nil
	}

	messageID := generateMessageID("email")

	p.logger.WithFields(logrus.Fields{
		"provider":   "simulated-sendgrid",
		"to":         notification.Recipient,
		"subject":    notification.Subject,
		"message_id": messageID,
	}).Info("Email sent (simulated)")

	return &ProviderResult{
		ProviderName: "simulated-sendgrid",
		MessageID:    messageID,
		Success:      true,
	}, nil
}

// === Simulated SMS Provider ===

type SimulatedSMSProvider struct {
	logger *logrus.Logger
}

func NewSimulatedSMSProvider(logger *logrus.Logger) NotificationProvider {
	return &SimulatedSMSProvider{logger: logger}
}

func (p *SimulatedSMSProvider) Channel() models.Channel {
	return models.ChannelSMS
}

func (p *SimulatedSMSProvider) Send(notification *models.Notification) (*ProviderResult, error) {
	if notification.Recipient == "" {
		return &ProviderResult{
			ProviderName: "simulated-twilio",
			Success:      false,
			Error:        "recipient phone number is required",
		}, nil
	}

	messageID := generateMessageID("sms")

	p.logger.WithFields(logrus.Fields{
		"provider":   "simulated-twilio",
		"to":         notification.Recipient,
		"message_id": messageID,
	}).Info("SMS sent (simulated)")

	return &ProviderResult{
		ProviderName: "simulated-twilio",
		MessageID:    messageID,
		Success:      true,
	}, nil
}

// === Simulated Push Provider ===

type SimulatedPushProvider struct {
	logger *logrus.Logger
}

func NewSimulatedPushProvider(logger *logrus.Logger) NotificationProvider {
	return &SimulatedPushProvider{logger: logger}
}

func (p *SimulatedPushProvider) Channel() models.Channel {
	return models.ChannelPush
}

func (p *SimulatedPushProvider) Send(notification *models.Notification) (*ProviderResult, error) {
	if notification.Recipient == "" {
		return &ProviderResult{
			ProviderName: "simulated-fcm",
			Success:      false,
			Error:        "device token is required",
		}, nil
	}

	messageID := generateMessageID("push")

	p.logger.WithFields(logrus.Fields{
		"provider":   "simulated-fcm",
		"to":         notification.Recipient[:min(20, len(notification.Recipient))] + "...",
		"message_id": messageID,
	}).Info("Push notification sent (simulated)")

	return &ProviderResult{
		ProviderName: "simulated-fcm",
		MessageID:    messageID,
		Success:      true,
	}, nil
}

func generateMessageID(prefix string) string {
	n, _ := rand.Int(rand.Reader, big.NewInt(999999999))
	return fmt.Sprintf("%s_%09d", prefix, n.Int64())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
