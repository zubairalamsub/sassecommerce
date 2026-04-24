package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

// SendGridEmailProvider sends emails via the SendGrid v3 API
type SendGridEmailProvider struct {
	apiKey    string
	fromEmail string
	fromName  string
	logger    *logrus.Logger
	client    *http.Client
}

// SendGridConfig holds configuration for the SendGrid provider
type SendGridConfig struct {
	APIKey    string
	FromEmail string
	FromName  string
}

func NewSendGridEmailProvider(config SendGridConfig, logger *logrus.Logger) NotificationProvider {
	return &SendGridEmailProvider{
		apiKey:    config.APIKey,
		fromEmail: config.FromEmail,
		fromName:  config.FromName,
		logger:    logger,
		client:    &http.Client{},
	}
}

func (p *SendGridEmailProvider) Channel() models.Channel {
	return models.ChannelEmail
}

func (p *SendGridEmailProvider) Send(notification *models.Notification) (*ProviderResult, error) {
	if notification.Recipient == "" {
		return &ProviderResult{
			ProviderName: "sendgrid",
			Success:      false,
			Error:        "recipient email is required",
		}, nil
	}

	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": []map[string]string{
					{"email": notification.Recipient},
				},
				"subject": notification.Subject,
			},
		},
		"from": map[string]string{
			"email": p.fromEmail,
			"name":  p.fromName,
		},
		"content": []map[string]string{
			{
				"type":  "text/html",
				"value": notification.Body,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SendGrid request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create SendGrid request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return &ProviderResult{
			ProviderName: "sendgrid",
			Success:      false,
			Error:        fmt.Sprintf("SendGrid API request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	messageID := resp.Header.Get("X-Message-Id")
	if messageID == "" {
		messageID = generateMessageID("sg")
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		p.logger.WithFields(logrus.Fields{
			"provider":   "sendgrid",
			"to":         notification.Recipient,
			"subject":    notification.Subject,
			"message_id": messageID,
			"status":     resp.StatusCode,
		}).Info("Email sent via SendGrid")

		return &ProviderResult{
			ProviderName: "sendgrid",
			MessageID:    messageID,
			Success:      true,
		}, nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	errMsg := fmt.Sprintf("SendGrid API returned status %d: %s", resp.StatusCode, string(respBody))

	p.logger.WithFields(logrus.Fields{
		"provider": "sendgrid",
		"to":       notification.Recipient,
		"status":   resp.StatusCode,
		"error":    errMsg,
	}).Error("Failed to send email via SendGrid")

	return &ProviderResult{
		ProviderName: "sendgrid",
		MessageID:    messageID,
		Success:      false,
		Error:        errMsg,
	}, nil
}
