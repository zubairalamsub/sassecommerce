package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

// TwilioSMSProvider sends SMS via the Twilio REST API
type TwilioSMSProvider struct {
	accountSID string
	authToken  string
	fromNumber string
	logger     *logrus.Logger
	client     *http.Client
}

// TwilioConfig holds configuration for the Twilio provider
type TwilioConfig struct {
	AccountSID string
	AuthToken  string
	FromNumber string
}

func NewTwilioSMSProvider(config TwilioConfig, logger *logrus.Logger) NotificationProvider {
	return &TwilioSMSProvider{
		accountSID: config.AccountSID,
		authToken:  config.AuthToken,
		fromNumber: config.FromNumber,
		logger:     logger,
		client:     &http.Client{},
	}
}

func (p *TwilioSMSProvider) Channel() models.Channel {
	return models.ChannelSMS
}

func (p *TwilioSMSProvider) Send(notification *models.Notification) (*ProviderResult, error) {
	if notification.Recipient == "" {
		return &ProviderResult{
			ProviderName: "twilio",
			Success:      false,
			Error:        "recipient phone number is required",
		}, nil
	}

	apiURL := fmt.Sprintf(
		"https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json",
		p.accountSID,
	)

	// Build SMS body: subject + body combined for SMS
	smsBody := notification.Body
	if notification.Subject != "" {
		smsBody = notification.Subject + ": " + notification.Body
	}
	// Twilio SMS max is 1600 chars
	if len(smsBody) > 1600 {
		smsBody = smsBody[:1597] + "..."
	}

	data := url.Values{}
	data.Set("To", notification.Recipient)
	data.Set("From", p.fromNumber)
	data.Set("Body", smsBody)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create Twilio request: %w", err)
	}

	req.SetBasicAuth(p.accountSID, p.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return &ProviderResult{
			ProviderName: "twilio",
			Success:      false,
			Error:        fmt.Sprintf("Twilio API request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var twilioResp struct {
		SID          string `json:"sid"`
		ErrorCode    int    `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	json.Unmarshal(respBody, &twilioResp)

	messageID := twilioResp.SID
	if messageID == "" {
		messageID = generateMessageID("twilio")
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		p.logger.WithFields(logrus.Fields{
			"provider":   "twilio",
			"to":         notification.Recipient,
			"message_id": messageID,
		}).Info("SMS sent via Twilio")

		return &ProviderResult{
			ProviderName: "twilio",
			MessageID:    messageID,
			Success:      true,
		}, nil
	}

	errMsg := fmt.Sprintf("Twilio API returned status %d: %s", resp.StatusCode, twilioResp.ErrorMessage)

	p.logger.WithFields(logrus.Fields{
		"provider":   "twilio",
		"to":         notification.Recipient,
		"status":     resp.StatusCode,
		"error_code": twilioResp.ErrorCode,
		"error":      errMsg,
	}).Error("Failed to send SMS via Twilio")

	return &ProviderResult{
		ProviderName: "twilio",
		MessageID:    messageID,
		Success:      false,
		Error:        errMsg,
	}, nil
}
