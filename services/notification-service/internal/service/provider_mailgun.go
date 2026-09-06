package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

// MailgunAPIProvider sends through Mailgun's HTTP messages API rather than
// SMTP. It exists alongside the SMTP "mailgun" entry because the two take
// different credentials: SMTP wants a per-domain postmaster@ login, while this
// takes the account API key. It is also the option that works where outbound
// port 587 is blocked, which is common on managed hosts.
type MailgunAPIProvider struct {
	cfg    MailgunConfig
	logger *logrus.Logger
	client *http.Client
	// baseURL is overridden in tests; empty means derive it from the config.
	baseURL string
}

// MailgunConfig configures the HTTP API transport.
type MailgunConfig struct {
	// APIKey is the private API key from Mailgun's API Security page. Not the
	// SMTP password.
	APIKey string

	// Domain is the verified sending domain the message is posted under. When
	// empty it is derived from FromEmail, which is what it has to match
	// anyway — Mailgun rejects a From address outside the posting domain.
	Domain string

	// APIHost selects the region: api.mailgun.net (default) or
	// api.eu.mailgun.net for EU accounts. A full https:// URL is also
	// accepted so a proxy can be pointed at.
	APIHost string

	FromEmail string
	FromName  string

	Timeout time.Duration
}

const (
	defaultMailgunHost    = "api.mailgun.net"
	defaultMailgunTimeout = 15 * time.Second
)

// NewMailgunAPIProvider builds the provider.
func NewMailgunAPIProvider(cfg MailgunConfig, logger *logrus.Logger) *MailgunAPIProvider {
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultMailgunTimeout
	}
	if cfg.Domain == "" {
		cfg.Domain = domainFromAddress(cfg.FromEmail)
	}
	return &MailgunAPIProvider{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

func (p *MailgunAPIProvider) Channel() models.Channel { return models.ChannelEmail }

// Name reports this provider's label for failover logging and results.
func (p *MailgunAPIProvider) Name() string { return "mailgun_api" }

// endpoint builds the messages URL for the configured region and domain.
func (p *MailgunAPIProvider) endpoint() string {
	if p.baseURL != "" {
		return fmt.Sprintf("%s/v3/%s/messages", strings.TrimSuffix(p.baseURL, "/"), p.cfg.Domain)
	}
	host := p.cfg.APIHost
	if host == "" {
		host = defaultMailgunHost
	}
	// Accept either a bare host or a full URL, so an EU account can be
	// configured with just "api.eu.mailgun.net".
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	return fmt.Sprintf("%s/v3/%s/messages", strings.TrimSuffix(host, "/"), p.cfg.Domain)
}

func (p *MailgunAPIProvider) Send(notification *models.Notification) (*ProviderResult, error) {
	if notification.Recipient == "" {
		return &ProviderResult{
			ProviderName: p.Name(),
			Success:      false,
			Error:        "recipient email is required",
		}, nil
	}
	if p.cfg.APIKey == "" {
		return &ProviderResult{
			ProviderName: p.Name(),
			Success:      false,
			Error:        "mailgun api key is not configured",
		}, nil
	}
	// Without a domain the URL would be malformed, and Mailgun's 404 for that
	// is opaque — say so plainly instead.
	if p.cfg.Domain == "" {
		return &ProviderResult{
			ProviderName: p.Name(),
			Success:      false,
			Error:        "mailgun sending domain is not set and could not be derived from the from address",
		}, nil
	}

	form := url.Values{}
	form.Set("from", formatAddress(p.cfg.FromName, p.cfg.FromEmail))
	form.Set("to", notification.Recipient)
	form.Set("subject", notification.Subject)
	form.Set("html", notification.Body)

	req, err := http.NewRequest(http.MethodPost, p.endpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create Mailgun request: %w", err)
	}
	// Mailgun authenticates with HTTP Basic, username literally "api".
	req.SetBasicAuth("api", p.cfg.APIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return &ProviderResult{
			ProviderName: p.Name(),
			Success:      false,
			Error:        fmt.Sprintf("mailgun request failed: %v", err),
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Mailgun puts the reason in a JSON "message" field; fall back to the
		// raw body so a proxy's HTML error page is not swallowed silently.
		reason := mailgunMessage(body)
		if reason == "" {
			reason = strings.TrimSpace(string(body))
		}
		return &ProviderResult{
			ProviderName: p.Name(),
			Success:      false,
			Error:        fmt.Sprintf("mailgun rejected the message (HTTP %d): %s", resp.StatusCode, reason),
		}, nil
	}

	messageID := mailgunID(body)
	if messageID == "" {
		messageID = generateMessageID("mg")
	}

	p.logger.WithFields(logrus.Fields{
		"provider":   p.Name(),
		"to":         notification.Recipient,
		"subject":    notification.Subject,
		"message_id": messageID,
	}).Info("Email accepted by Mailgun API")

	return &ProviderResult{
		ProviderName: p.Name(),
		MessageID:    messageID,
		Success:      true,
	}, nil
}

// mailgunResponse is the shape of both success and error bodies.
type mailgunResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func mailgunID(body []byte) string {
	var r mailgunResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return ""
	}
	return r.ID
}

func mailgunMessage(body []byte) string {
	var r mailgunResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return ""
	}
	return r.Message
}

// domainFromAddress returns the part after "@", which for Mailgun is the
// sending domain a message must be posted under.
func domainFromAddress(address string) string {
	at := strings.LastIndex(address, "@")
	if at < 0 || at == len(address)-1 {
		return ""
	}
	return address[at+1:]
}

// formatAddress renders a From header value, quoting the display name so a
// comma or parenthesis in it cannot split the address.
func formatAddress(name, email string) string {
	if name == "" {
		return email
	}
	escaped := strings.ReplaceAll(name, `"`, `\"`)
	return fmt.Sprintf(`"%s" <%s>`, escaped, email)
}
