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

// FCMPushProvider sends push notifications via Firebase Cloud Messaging HTTP v1 API
type FCMPushProvider struct {
	serverKey string
	projectID string
	logger    *logrus.Logger
	client    *http.Client
}

// FCMConfig holds configuration for the FCM provider
type FCMConfig struct {
	// ServerKey is the FCM server key (for legacy HTTP API)
	ServerKey string
	// ProjectID is the Firebase project ID (for v1 API)
	ProjectID string
}

func NewFCMPushProvider(config FCMConfig, logger *logrus.Logger) NotificationProvider {
	return &FCMPushProvider{
		serverKey: config.ServerKey,
		projectID: config.ProjectID,
		logger:    logger,
		client:    &http.Client{},
	}
}

func (p *FCMPushProvider) Channel() models.Channel {
	return models.ChannelPush
}

func (p *FCMPushProvider) Send(notification *models.Notification) (*ProviderResult, error) {
	if notification.Recipient == "" {
		return &ProviderResult{
			ProviderName: "fcm",
			Success:      false,
			Error:        "device token is required",
		}, nil
	}

	// Use FCM legacy HTTP API (simpler, no OAuth required)
	payload := map[string]interface{}{
		"to": notification.Recipient,
		"notification": map[string]string{
			"title": notification.Subject,
			"body":  notification.Body,
		},
	}

	// Add metadata as data payload if available
	if notification.Metadata != nil {
		dataPayload := make(map[string]string)
		for k, v := range notification.Metadata {
			dataPayload[k] = fmt.Sprintf("%v", v)
		}
		if notification.ReferenceID != "" {
			dataPayload["reference_id"] = notification.ReferenceID
			dataPayload["reference_type"] = notification.ReferenceType
		}
		payload["data"] = dataPayload
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal FCM request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://fcm.googleapis.com/fcm/send", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create FCM request: %w", err)
	}

	req.Header.Set("Authorization", "key="+p.serverKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return &ProviderResult{
			ProviderName: "fcm",
			Success:      false,
			Error:        fmt.Sprintf("FCM API request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var fcmResp struct {
		MulticastID int64 `json:"multicast_id"`
		Success     int   `json:"success"`
		Failure     int   `json:"failure"`
		Results     []struct {
			MessageID string `json:"message_id"`
			Error     string `json:"error"`
		} `json:"results"`
	}
	json.Unmarshal(respBody, &fcmResp)

	messageID := generateMessageID("fcm")
	if len(fcmResp.Results) > 0 && fcmResp.Results[0].MessageID != "" {
		messageID = fcmResp.Results[0].MessageID
	}

	if resp.StatusCode == http.StatusOK && fcmResp.Success > 0 {
		tokenPreview := notification.Recipient
		if len(tokenPreview) > 20 {
			tokenPreview = tokenPreview[:20] + "..."
		}

		p.logger.WithFields(logrus.Fields{
			"provider":   "fcm",
			"to":         tokenPreview,
			"message_id": messageID,
		}).Info("Push notification sent via FCM")

		return &ProviderResult{
			ProviderName: "fcm",
			MessageID:    messageID,
			Success:      true,
		}, nil
	}

	errMsg := "FCM delivery failed"
	if len(fcmResp.Results) > 0 && fcmResp.Results[0].Error != "" {
		errMsg = fmt.Sprintf("FCM error: %s", fcmResp.Results[0].Error)
	}

	p.logger.WithFields(logrus.Fields{
		"provider": "fcm",
		"status":   resp.StatusCode,
		"error":    errMsg,
	}).Error("Failed to send push notification via FCM")

	return &ProviderResult{
		ProviderName: "fcm",
		MessageID:    messageID,
		Success:      false,
		Error:        errMsg,
	}, nil
}
