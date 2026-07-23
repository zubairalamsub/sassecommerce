package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

// SSLWirelessSMSProvider sends SMS via SSL Wireless (smsplus.sslwireless.com),
// the most common SMS gateway in Bangladesh.
//
// API reference: https://sslwireless.com/sms-marketing/ — v3 JSON endpoint.
type SSLWirelessSMSProvider struct {
	apiToken string
	sid      string
	endpoint string
	logger   *logrus.Logger
	client   *http.Client
}

// SSLWirelessConfig holds configuration for the SSL Wireless provider.
type SSLWirelessConfig struct {
	// APIToken is the api_token issued by SSL Wireless.
	APIToken string
	// SID is the approved Sender ID / brand mask.
	SID string
	// Endpoint optionally overrides the default v3 send URL (useful in tests).
	Endpoint string
}

const sslWirelessDefaultEndpoint = "https://smsplus.sslwireless.com/api/v3/send-sms"

func NewSSLWirelessSMSProvider(config SSLWirelessConfig, logger *logrus.Logger) NotificationProvider {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = sslWirelessDefaultEndpoint
	}
	return &SSLWirelessSMSProvider{
		apiToken: config.APIToken,
		sid:      config.SID,
		endpoint: endpoint,
		logger:   logger,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *SSLWirelessSMSProvider) Channel() models.Channel {
	return models.ChannelSMS
}

func (p *SSLWirelessSMSProvider) Send(notification *models.Notification) (*ProviderResult, error) {
	if notification.Recipient == "" {
		return &ProviderResult{
			ProviderName: "sslwireless",
			Success:      false,
			Error:        "recipient phone number is required",
		}, nil
	}

	msisdn := normalizeBDMSISDN(notification.Recipient)

	smsBody := notification.Body
	if notification.Subject != "" {
		smsBody = notification.Subject + ": " + notification.Body
	}
	// SSL Wireless caps a single SMS chain at ~1000 chars; trim defensively.
	if len(smsBody) > 1000 {
		smsBody = smsBody[:997] + "..."
	}

	csmsID := notification.ID
	if csmsID == "" {
		csmsID = generateMessageID("ssl")
	}

	payload := map[string]interface{}{
		"api_token": p.apiToken,
		"sid":       p.sid,
		"msisdn":    msisdn,
		"sms":       smsBody,
		"csms_id":   csmsID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SSL Wireless request: %w", err)
	}

	req, err := http.NewRequest("POST", p.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create SSL Wireless request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return &ProviderResult{
			ProviderName: "sslwireless",
			Success:      false,
			Error:        fmt.Sprintf("SSL Wireless request failed: %v", err),
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	var sslResp struct {
		Status       string `json:"status"`
		StatusCode   int    `json:"status_code"`
		ErrorMessage string `json:"error_message"`
		SMSInfo      []struct {
			SMSStatus   string `json:"sms_status"`
			MSISDN      string `json:"msisdn"`
			CSMSID      string `json:"csms_id"`
			ReferenceID string `json:"reference_id"`
		} `json:"smsinfo"`
	}
	_ = json.Unmarshal(respBody, &sslResp)

	messageID := csmsID
	if len(sslResp.SMSInfo) > 0 && sslResp.SMSInfo[0].ReferenceID != "" {
		messageID = sslResp.SMSInfo[0].ReferenceID
	}

	delivered := resp.StatusCode >= 200 && resp.StatusCode < 300 &&
		strings.EqualFold(sslResp.Status, "SUCCESS")

	if delivered {
		p.logger.WithFields(logrus.Fields{
			"provider":   "sslwireless",
			"to":         msisdn,
			"message_id": messageID,
		}).Info("SMS sent via SSL Wireless")
		return &ProviderResult{
			ProviderName: "sslwireless",
			MessageID:    messageID,
			Success:      true,
		}, nil
	}

	errMsg := sslResp.ErrorMessage
	if errMsg == "" {
		errMsg = fmt.Sprintf("SSL Wireless returned status %d: %s", resp.StatusCode, string(respBody))
	}
	p.logger.WithFields(logrus.Fields{
		"provider":    "sslwireless",
		"to":          msisdn,
		"status":      resp.StatusCode,
		"api_status":  sslResp.Status,
		"status_code": sslResp.StatusCode,
		"error":       errMsg,
	}).Error("Failed to send SMS via SSL Wireless")

	return &ProviderResult{
		ProviderName: "sslwireless",
		MessageID:    messageID,
		Success:      false,
		Error:        errMsg,
	}, nil
}

// normalizeBDMSISDN coerces a Bangladesh phone number into the 8801XXXXXXXXX format
// SSL Wireless expects. Inputs like "+8801712345678", "01712345678", "8801712345678"
// all map to "8801712345678". Non-BD numbers pass through unchanged (minus leading +).
func normalizeBDMSISDN(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "+")
	// Strip spaces and dashes that users sometimes type.
	s = strings.NewReplacer(" ", "", "-", "").Replace(s)
	if strings.HasPrefix(s, "880") {
		return s
	}
	if strings.HasPrefix(s, "0") {
		return "880" + strings.TrimPrefix(s, "0")
	}
	return s
}
