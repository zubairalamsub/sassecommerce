package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

// smtpPreset holds the relay coordinates for a known vendor, so configuring it
// needs only credentials. Every one of these speaks ordinary SMTP, which is
// why a single provider implementation covers all of them.
type smtpPreset struct {
	host string
	port int
}

// smtpPresets are the vendors with a usable free tier at the time of writing.
// Anything not listed still works through the explicit "smtp" entry, and a
// preset's host/port can always be overridden per provider.
var smtpPresets = map[string]smtpPreset{
	"brevo":        {host: "smtp-relay.brevo.com", port: 587},
	"mailjet":      {host: "in-v3.mailjet.com", port: 587},
	"resend":       {host: "smtp.resend.com", port: 587},
	"smtp2go":      {host: "mail.smtp2go.com", port: 2525},
	"elasticemail": {host: "smtp.elasticemail.com", port: 2525},
	"zeptomail":    {host: "smtp.zeptomail.com", port: 587},
	"mailgun":      {host: "smtp.mailgun.org", port: 587},
	"postmark":     {host: "smtp.postmarkapp.com", port: 587},
	// SES is region-scoped, so its host must be given explicitly via
	// SES_SMTP_HOST; the port is the standard submission one.
	"ses": {host: "", port: 587},
}

// MailgunAPIProviderKey is the provider key for Mailgun's HTTP API. It is
// separate from the SMTP "mailgun" entry because the two take different
// credentials — the API key here, a per-domain postmaster@ login there — and
// collapsing them would silently break whichever was configured first.
const MailgunAPIProviderKey = "mailgun_api"

// EmailProviderNames lists every provider key BuildEmailChain accepts.
func EmailProviderNames() []string {
	names := []string{"sendgrid", "smtp", "simulated", MailgunAPIProviderKey}
	for name := range smtpPresets {
		names = append(names, name)
	}
	return names
}

// BuildEmailChain turns an ordered, comma-separated spec such as
// "brevo,sendgrid,simulated" into a single NotificationProvider. Providers are
// attempted left to right, so the first entry carries normal traffic and the
// rest are fallbacks.
//
// env is injected rather than read from the process so this is testable and so
// callers can source configuration from somewhere other than the environment.
//
// A provider named in the spec but missing its credentials is skipped with a
// warning rather than failing startup: losing a fallback should not stop the
// service from booting on the providers that are configured. An empty spec, or
// one where nothing is configured, returns nil so the caller can decide.
func BuildEmailChain(spec string, env func(string) string, logger *logrus.Logger) (NotificationProvider, []string) {
	entries := splitSpec(spec)
	built := make([]NotificationProvider, 0, len(entries))
	names := make([]string, 0, len(entries))

	for _, entry := range entries {
		provider, err := buildEmailProvider(entry, env, logger)
		if err != nil {
			logger.WithError(err).WithField("provider", entry).
				Warn("Skipping email provider: not configured")
			continue
		}
		built = append(built, provider)
		names = append(names, entry)
	}

	if len(built) == 0 {
		return nil, nil
	}
	return NewFailoverProvider(models.ChannelEmail, logger, built...), names
}

func buildEmailProvider(name string, env func(string) string, logger *logrus.Logger) (NotificationProvider, error) {
	switch name {
	case "simulated":
		return NewSimulatedEmailProvider(logger), nil

	case "sendgrid":
		key := env("SENDGRID_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("SENDGRID_API_KEY is not set")
		}
		return NewSendGridEmailProvider(SendGridConfig{
			APIKey:    key,
			FromEmail: firstNonEmpty(env("SENDGRID_FROM_EMAIL"), env("EMAIL_FROM_ADDRESS"), "noreply@saajan.com"),
			FromName:  firstNonEmpty(env("SENDGRID_FROM_NAME"), env("EMAIL_FROM_NAME"), "Saajan Store"),
		}, logger), nil

	case MailgunAPIProviderKey:
		key := env("MAILGUN_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("MAILGUN_API_KEY is not set")
		}
		from := firstNonEmpty(env("MAILGUN_FROM_ADDRESS"), env("EMAIL_FROM_ADDRESS"), "noreply@saajan.com")
		return NewMailgunAPIProvider(MailgunConfig{
			APIKey: key,
			// Falls back to the from address's domain, which is what Mailgun
			// requires the posting domain to match anyway.
			Domain:    firstNonEmpty(env("MAILGUN_DOMAIN"), domainFromAddress(from)),
			APIHost:   env("MAILGUN_API_HOST"),
			FromEmail: from,
			FromName:  firstNonEmpty(env("MAILGUN_FROM_NAME"), env("EMAIL_FROM_NAME"), "Saajan Store"),
		}, logger), nil

	default:
		return buildSMTPProvider(name, env, logger)
	}
}

// buildSMTPProvider configures either a preset vendor or the explicit "smtp"
// entry. Credentials are looked up under the provider's own prefix first
// (BREVO_SMTP_USER) so several relays can be configured side by side, falling
// back to the generic SMTP_* names for a single-relay setup.
func buildSMTPProvider(name string, env func(string) string, logger *logrus.Logger) (NotificationProvider, error) {
	prefix := strings.ToUpper(name) + "_SMTP_"
	preset, isPreset := smtpPresets[name]
	if !isPreset && name != "smtp" {
		return nil, fmt.Errorf("unknown email provider %q (known: %s)", name, strings.Join(EmailProviderNames(), ", "))
	}
	if name == "smtp" {
		prefix = "SMTP_"
	}

	lookup := func(suffix string) string {
		if v := env(prefix + suffix); v != "" {
			return v
		}
		return env("SMTP_" + suffix)
	}

	host := firstNonEmpty(lookup("HOST"), preset.host)
	if host == "" {
		return nil, fmt.Errorf("%sHOST is not set and %q has no default host", prefix, name)
	}

	port := preset.port
	if raw := lookup("PORT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("%sPORT is not a number: %q", prefix, raw)
		}
		port = parsed
	}
	if port == 0 {
		port = 587
	}

	user := firstNonEmpty(lookup("USER"), lookup("USERNAME"))
	password := lookup("PASSWORD")
	if user == "" || password == "" {
		return nil, fmt.Errorf("%sUSER and %sPASSWORD are required", prefix, prefix)
	}

	return NewSMTPEmailProvider(SMTPConfig{
		Name:     name,
		Host:     host,
		Port:     port,
		Username: user,
		Password: password,
		// Port 465 is implicit TLS by convention; everything else upgrades
		// with STARTTLS.
		ImplicitTLS: port == 465,
		FromEmail:   firstNonEmpty(lookup("FROM_ADDRESS"), env("EMAIL_FROM_ADDRESS"), "noreply@saajan.com"),
		FromName:    firstNonEmpty(lookup("FROM_NAME"), env("EMAIL_FROM_NAME"), "Saajan Store"),
	}, logger), nil
}

func splitSpec(spec string) []string {
	parts := strings.Split(spec, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.ToLower(strings.TrimSpace(p)); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// BuildProviderFromConfig turns a stored EmailProviderConfig plus its
// decrypted secret into a live provider. It shares the preset table with the
// environment path, so a vendor configured through the UI and the same vendor
// configured through EMAIL_PROVIDERS resolve to identical relay coordinates.
func BuildProviderFromConfig(cfg *models.EmailProviderConfig, secret string, logger *logrus.Logger) (NotificationProvider, error) {
	name := strings.ToLower(strings.TrimSpace(cfg.Provider))

	switch name {
	case "simulated":
		return NewSimulatedEmailProvider(logger), nil

	case "sendgrid":
		if secret == "" {
			return nil, fmt.Errorf("sendgrid requires an API key")
		}
		return NewSendGridEmailProvider(SendGridConfig{
			APIKey:    secret,
			FromEmail: firstNonEmpty(cfg.FromEmail, "noreply@saajan.com"),
			FromName:  firstNonEmpty(cfg.FromName, "Saajan Store"),
		}, logger), nil

	case MailgunAPIProviderKey:
		if secret == "" {
			return nil, fmt.Errorf("mailgun api requires an API key")
		}
		from := firstNonEmpty(cfg.FromEmail, "noreply@saajan.com")
		// Username doubles as an explicit sending domain here; leaving it
		// blank derives the domain from the from address. Host selects the
		// region (api.eu.mailgun.net for EU accounts).
		domain := firstNonEmpty(cfg.Username, domainFromAddress(from))
		if domain == "" {
			return nil, fmt.Errorf("mailgun api needs a sending domain, or a from address to derive it from")
		}
		return NewMailgunAPIProvider(MailgunConfig{
			APIKey:    secret,
			Domain:    domain,
			APIHost:   cfg.Host,
			FromEmail: from,
			FromName:  firstNonEmpty(cfg.FromName, "Saajan Store"),
		}, logger), nil
	}

	preset, isPreset := smtpPresets[name]
	if !isPreset && name != "smtp" {
		return nil, fmt.Errorf("unknown email provider %q (known: %s)", name, strings.Join(EmailProviderNames(), ", "))
	}

	host := firstNonEmpty(cfg.Host, preset.host)
	if host == "" {
		return nil, fmt.Errorf("provider %q needs an explicit host", name)
	}
	port := cfg.Port
	if port == 0 {
		port = preset.port
	}
	if port == 0 {
		port = 587
	}
	if cfg.Username == "" || secret == "" {
		return nil, fmt.Errorf("provider %q needs a username and secret", name)
	}

	return NewSMTPEmailProvider(SMTPConfig{
		Name:        name,
		Host:        host,
		Port:        port,
		Username:    cfg.Username,
		Password:    secret,
		ImplicitTLS: port == 465,
		FromEmail:   firstNonEmpty(cfg.FromEmail, "noreply@saajan.com"),
		FromName:    firstNonEmpty(cfg.FromName, "Saajan Store"),
	}, logger), nil
}
