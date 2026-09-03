package service

import (
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

// SMTPEmailProvider sends email over plain SMTP. One implementation covers
// every vendor that speaks SMTP — Brevo, Mailjet, SMTP2GO, Elastic Email,
// Zoho ZeptoMail, Resend and Amazon SES — so switching or adding a vendor is
// configuration rather than code. Name is carried through to logs and
// ProviderResult so a failover chain reports which hop actually delivered.
type SMTPEmailProvider struct {
	cfg    SMTPConfig
	logger *logrus.Logger
	// dial is swapped in tests; production uses dialSMTP.
	dial func(SMTPConfig) (smtpClient, error)
}

// SMTPConfig describes one SMTP relay.
type SMTPConfig struct {
	// Name labels this relay in logs and results, e.g. "brevo". Defaults to
	// "smtp".
	Name     string
	Host     string
	Port     int
	Username string
	Password string

	FromEmail string
	FromName  string

	// ImplicitTLS dials TLS directly, which is what port 465 expects. Leave
	// false for the far more common submission port 587, which connects in
	// the clear and upgrades via STARTTLS.
	ImplicitTLS bool

	// Timeout bounds the whole conversation. A relay that accepts the TCP
	// connection and then stalls would otherwise hold the goroutine open
	// indefinitely.
	Timeout time.Duration
}

// smtpClient is the slice of *smtp.Client this provider uses, so the send path
// can be tested without a relay.
type smtpClient interface {
	Auth(smtp.Auth) error
	Mail(string) error
	Rcpt(string) error
	Data() (dataWriter, error)
	Quit() error
	Close() error
}

type dataWriter interface {
	Write([]byte) (int, error)
	Close() error
}

const defaultSMTPTimeout = 15 * time.Second

// NewSMTPEmailProvider builds a provider for one SMTP relay.
func NewSMTPEmailProvider(cfg SMTPConfig, logger *logrus.Logger) *SMTPEmailProvider {
	if cfg.Name == "" {
		cfg.Name = "smtp"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultSMTPTimeout
	}
	return &SMTPEmailProvider{cfg: cfg, logger: logger, dial: dialSMTP}
}

func (p *SMTPEmailProvider) Channel() models.Channel {
	return models.ChannelEmail
}

// Name reports the configured relay label.
func (p *SMTPEmailProvider) Name() string {
	return p.cfg.Name
}

func (p *SMTPEmailProvider) Send(notification *models.Notification) (*ProviderResult, error) {
	if notification.Recipient == "" {
		return &ProviderResult{
			ProviderName: p.cfg.Name,
			Success:      false,
			Error:        "recipient email is required",
		}, nil
	}
	if p.cfg.Host == "" {
		return &ProviderResult{
			ProviderName: p.cfg.Name,
			Success:      false,
			Error:        "smtp host is not configured",
		}, nil
	}

	messageID := generateMessageID(p.cfg.Name)
	msg := p.buildMessage(notification, messageID)

	client, err := p.dial(p.cfg)
	if err != nil {
		return &ProviderResult{
			ProviderName: p.cfg.Name,
			Success:      false,
			Error:        fmt.Sprintf("smtp connect failed: %v", err),
		}, nil
	}
	defer func() { _ = client.Close() }()

	if p.cfg.Username != "" {
		auth := smtp.PlainAuth("", p.cfg.Username, p.cfg.Password, p.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return &ProviderResult{
				ProviderName: p.cfg.Name,
				Success:      false,
				Error:        fmt.Sprintf("smtp auth failed: %v", err),
			}, nil
		}
	}

	if err := client.Mail(p.cfg.FromEmail); err != nil {
		return &ProviderResult{
			ProviderName: p.cfg.Name,
			Success:      false,
			Error:        fmt.Sprintf("smtp MAIL FROM failed: %v", err),
		}, nil
	}
	if err := client.Rcpt(notification.Recipient); err != nil {
		return &ProviderResult{
			ProviderName: p.cfg.Name,
			Success:      false,
			Error:        fmt.Sprintf("smtp RCPT TO failed: %v", err),
		}, nil
	}

	w, err := client.Data()
	if err != nil {
		return &ProviderResult{
			ProviderName: p.cfg.Name,
			Success:      false,
			Error:        fmt.Sprintf("smtp DATA failed: %v", err),
		}, nil
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return &ProviderResult{
			ProviderName: p.cfg.Name,
			Success:      false,
			Error:        fmt.Sprintf("smtp write failed: %v", err),
		}, nil
	}
	// Closing the data writer is what commits the message; a relay rejects
	// here (size, spam, policy) so the error matters more than most.
	if err := w.Close(); err != nil {
		return &ProviderResult{
			ProviderName: p.cfg.Name,
			Success:      false,
			Error:        fmt.Sprintf("smtp message rejected: %v", err),
		}, nil
	}

	// Quit failing after a committed message is not a delivery failure, so it
	// is logged rather than surfaced.
	if err := client.Quit(); err != nil {
		p.logger.WithError(err).WithField("provider", p.cfg.Name).
			Debug("SMTP QUIT failed after the message was accepted")
	}

	p.logger.WithFields(logrus.Fields{
		"provider":   p.cfg.Name,
		"to":         notification.Recipient,
		"subject":    notification.Subject,
		"message_id": messageID,
	}).Info("Email sent via SMTP")

	return &ProviderResult{
		ProviderName: p.cfg.Name,
		MessageID:    messageID,
		Success:      true,
	}, nil
}

// buildMessage assembles the RFC 5322 message. Subject and sender name are
// RFC 2047 encoded: the templates deliberately emit Bangla and the ৳ sign, and
// raw non-ASCII in a header is not legal and gets mangled in transit.
func (p *SMTPEmailProvider) buildMessage(notification *models.Notification, messageID string) string {
	from := p.cfg.FromEmail
	if p.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("utf-8", p.cfg.FromName), p.cfg.FromEmail)
	}

	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + notification.Recipient + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", notification.Subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString(fmt.Sprintf("X-Notification-Id: %s\r\n", messageID))
	b.WriteString("\r\n")
	// Bare newlines are illegal in SMTP data and some relays reject or rewrite
	// them; the HTML templates are written with \n.
	b.WriteString(normalizeCRLF(notification.Body))
	return b.String()
}

// normalizeCRLF converts lone \n to \r\n without doubling existing \r\n.
func normalizeCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// dialSMTP opens a connection, upgrading to TLS as configured.
func dialSMTP(cfg SMTPConfig) (smtpClient, error) {
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	tlsCfg := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}

	if cfg.ImplicitTLS {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: cfg.Timeout}, "tcp", addr, tlsCfg)
		if err != nil {
			return nil, err
		}
		c, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		return &stdSMTPClient{c}, nil
	}

	conn, err := net.DialTimeout("tcp", addr, cfg.Timeout)
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(time.Now().Add(cfg.Timeout)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	c, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	// Credentials must never cross the wire in the clear, so STARTTLS is
	// required rather than best-effort whenever we are authenticating.
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(tlsCfg); err != nil {
			_ = c.Close()
			return nil, err
		}
	} else if cfg.Username != "" {
		_ = c.Close()
		return nil, errors.New("relay does not support STARTTLS; refusing to send credentials in cleartext")
	}
	return &stdSMTPClient{c}, nil
}

// stdSMTPClient adapts *smtp.Client to the smtpClient interface.
type stdSMTPClient struct{ c *smtp.Client }

func (s *stdSMTPClient) Auth(a smtp.Auth) error { return s.c.Auth(a) }
func (s *stdSMTPClient) Mail(from string) error { return s.c.Mail(from) }
func (s *stdSMTPClient) Rcpt(to string) error   { return s.c.Rcpt(to) }
func (s *stdSMTPClient) Quit() error            { return s.c.Quit() }
func (s *stdSMTPClient) Close() error           { return s.c.Close() }
func (s *stdSMTPClient) Data() (dataWriter, error) {
	w, err := s.c.Data()
	if err != nil {
		return nil, err
	}
	return w, nil
}
