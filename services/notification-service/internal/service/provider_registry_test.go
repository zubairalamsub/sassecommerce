package service

import (
	"strings"
	"testing"
)

// envMap turns a map into the lookup func BuildEmailChain takes, so tests
// never touch the real process environment.
func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func brevoCreds() map[string]string {
	return map[string]string{
		"BREVO_SMTP_USER":     "brevo-user",
		"BREVO_SMTP_PASSWORD": "brevo-pass",
	}
}

func TestBuildEmailChainOrdersProvidersAsSpecified(t *testing.T) {
	env := brevoCreds()
	env["SENDGRID_API_KEY"] = "sg-key"

	chain, names := BuildEmailChain("brevo,sendgrid,simulated", envMap(env), quietLog())

	if chain == nil {
		t.Fatal("chain = nil, want three providers")
	}
	want := []string{"brevo", "sendgrid", "simulated"}
	if len(names) != len(want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
	fp, ok := chain.(*FailoverProvider)
	if !ok {
		t.Fatalf("chain = %T, want *FailoverProvider", chain)
	}
	if fp.Len() != 3 {
		t.Errorf("chain length = %d, want 3", fp.Len())
	}
}

// A named provider missing its credentials is skipped rather than failing
// startup — losing a fallback must not stop the service booting on the
// providers that are configured.
func TestBuildEmailChainSkipsUnconfiguredProviders(t *testing.T) {
	// brevo has no credentials; only simulated can be built.
	chain, names := BuildEmailChain("brevo,simulated", envMap(map[string]string{}), quietLog())

	if chain == nil {
		t.Fatal("chain = nil, want simulated to survive")
	}
	if len(names) != 1 || names[0] != "simulated" {
		t.Fatalf("names = %v, want just [simulated]", names)
	}
	// One survivor means no wrapper.
	if _, ok := chain.(*FailoverProvider); ok {
		t.Error("a single surviving provider should not be wrapped in a chain")
	}
}

func TestBuildEmailChainReturnsNilWhenNothingIsConfigured(t *testing.T) {
	chain, names := BuildEmailChain("brevo,sendgrid", envMap(map[string]string{}), quietLog())

	if chain != nil {
		t.Errorf("chain = %#v, want nil so the caller can decide", chain)
	}
	if len(names) != 0 {
		t.Errorf("names = %v, want empty", names)
	}
}

func TestBuildEmailChainIgnoresBlanksAndCase(t *testing.T) {
	chain, names := BuildEmailChain("  SIMULATED , ,  ", envMap(map[string]string{}), quietLog())

	if chain == nil || len(names) != 1 || names[0] != "simulated" {
		t.Errorf("chain=%v names=%v, want a single simulated provider", chain, names)
	}
}

func TestBuildEmailChainRejectsUnknownProvider(t *testing.T) {
	chain, names := BuildEmailChain("carrierpigeon", envMap(map[string]string{}), quietLog())

	if chain != nil || len(names) != 0 {
		t.Errorf("chain=%v names=%v, want an unknown provider to be skipped", chain, names)
	}
}

// Presets exist so configuring a known vendor needs only credentials.
func TestPresetSuppliesHostAndPort(t *testing.T) {
	provider, err := buildEmailProvider("brevo", envMap(brevoCreds()), quietLog())
	if err != nil {
		t.Fatalf("buildEmailProvider(brevo) = %v", err)
	}

	smtpProvider, ok := provider.(*SMTPEmailProvider)
	if !ok {
		t.Fatalf("provider = %T, want *SMTPEmailProvider", provider)
	}
	if smtpProvider.cfg.Host != "smtp-relay.brevo.com" || smtpProvider.cfg.Port != 587 {
		t.Errorf("relay = %s:%d, want brevo's defaults", smtpProvider.cfg.Host, smtpProvider.cfg.Port)
	}
	if smtpProvider.Name() != "brevo" {
		t.Errorf("name = %q, want brevo", smtpProvider.Name())
	}
	if smtpProvider.cfg.ImplicitTLS {
		t.Error("port 587 should upgrade via STARTTLS, not dial implicit TLS")
	}
}

func TestPresetHostAndPortCanBeOverridden(t *testing.T) {
	env := brevoCreds()
	env["BREVO_SMTP_HOST"] = "relay.internal"
	env["BREVO_SMTP_PORT"] = "465"

	provider, err := buildEmailProvider("brevo", envMap(env), quietLog())
	if err != nil {
		t.Fatalf("buildEmailProvider = %v", err)
	}

	cfg := provider.(*SMTPEmailProvider).cfg
	if cfg.Host != "relay.internal" || cfg.Port != 465 {
		t.Errorf("relay = %s:%d, want the overrides", cfg.Host, cfg.Port)
	}
	// 465 is implicit TLS by convention.
	if !cfg.ImplicitTLS {
		t.Error("port 465 should dial implicit TLS")
	}
}

// Several relays must be configurable side by side, each under its own prefix.
func TestPerProviderCredentialsAreIndependent(t *testing.T) {
	env := map[string]string{
		"BREVO_SMTP_USER":       "brevo-user",
		"BREVO_SMTP_PASSWORD":   "brevo-pass",
		"MAILJET_SMTP_USER":     "mailjet-user",
		"MAILJET_SMTP_PASSWORD": "mailjet-pass",
	}

	brevo, err := buildEmailProvider("brevo", envMap(env), quietLog())
	if err != nil {
		t.Fatalf("brevo: %v", err)
	}
	mailjet, err := buildEmailProvider("mailjet", envMap(env), quietLog())
	if err != nil {
		t.Fatalf("mailjet: %v", err)
	}

	if got := brevo.(*SMTPEmailProvider).cfg.Username; got != "brevo-user" {
		t.Errorf("brevo user = %q, want brevo-user", got)
	}
	if got := mailjet.(*SMTPEmailProvider).cfg.Username; got != "mailjet-user" {
		t.Errorf("mailjet user = %q, want mailjet-user", got)
	}
	if brevo.(*SMTPEmailProvider).cfg.Host == mailjet.(*SMTPEmailProvider).cfg.Host {
		t.Error("the two relays resolved to the same host")
	}
}

// Generic SMTP_* names cover the single-relay case.
func TestGenericSMTPCredentialsActAsFallback(t *testing.T) {
	env := map[string]string{
		"SMTP_USERNAME": "generic-user",
		"SMTP_PASSWORD": "generic-pass",
	}

	provider, err := buildEmailProvider("brevo", envMap(env), quietLog())
	if err != nil {
		t.Fatalf("buildEmailProvider = %v", err)
	}
	if got := provider.(*SMTPEmailProvider).cfg.Username; got != "generic-user" {
		t.Errorf("username = %q, want the generic SMTP_USERNAME", got)
	}
}

func TestExplicitSMTPEntryRequiresAHost(t *testing.T) {
	env := map[string]string{"SMTP_USER": "u", "SMTP_PASSWORD": "p"}

	if _, err := buildEmailProvider("smtp", envMap(env), quietLog()); err == nil {
		t.Fatal("expected an error when SMTP_HOST is unset for the explicit entry")
	}

	env["SMTP_HOST"] = "relay.example.test"
	provider, err := buildEmailProvider("smtp", envMap(env), quietLog())
	if err != nil {
		t.Fatalf("buildEmailProvider = %v", err)
	}
	cfg := provider.(*SMTPEmailProvider).cfg
	if cfg.Host != "relay.example.test" || cfg.Port != 587 {
		t.Errorf("relay = %s:%d, want the host with the default submission port", cfg.Host, cfg.Port)
	}
}

// SES is region-scoped, so it deliberately ships without a default host.
func TestSESRequiresAnExplicitHost(t *testing.T) {
	env := map[string]string{"SES_SMTP_USER": "u", "SES_SMTP_PASSWORD": "p"}

	if _, err := buildEmailProvider("ses", envMap(env), quietLog()); err == nil {
		t.Fatal("expected SES to require an explicit regional host")
	}

	env["SES_SMTP_HOST"] = "email-smtp.ap-southeast-1.amazonaws.com"
	if _, err := buildEmailProvider("ses", envMap(env), quietLog()); err != nil {
		t.Errorf("buildEmailProvider(ses) = %v, want success once the host is given", err)
	}
}

func TestCredentialsAreMandatoryForSMTPProviders(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
	}{
		{name: "no user", env: map[string]string{"BREVO_SMTP_PASSWORD": "p"}},
		{name: "no password", env: map[string]string{"BREVO_SMTP_USER": "u"}},
		{name: "neither", env: map[string]string{}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := buildEmailProvider("brevo", envMap(c.env), quietLog()); err == nil {
				t.Error("expected missing credentials to be an error")
			}
		})
	}
}

func TestNonNumericPortIsRejected(t *testing.T) {
	env := brevoCreds()
	env["BREVO_SMTP_PORT"] = "five-eight-seven"

	_, err := buildEmailProvider("brevo", envMap(env), quietLog())

	if err == nil {
		t.Fatal("expected a non-numeric port to be rejected")
	}
	if !strings.Contains(err.Error(), "PORT") {
		t.Errorf("error %q should name the offending variable", err)
	}
}

// The shared EMAIL_FROM_* pair should apply to every provider so the sender
// identity is set once.
func TestSharedFromAddressAppliesAcrossProviders(t *testing.T) {
	env := brevoCreds()
	env["SENDGRID_API_KEY"] = "sg"
	env["EMAIL_FROM_ADDRESS"] = "shop@saajan.test"
	env["EMAIL_FROM_NAME"] = "Saajan"

	brevo, err := buildEmailProvider("brevo", envMap(env), quietLog())
	if err != nil {
		t.Fatalf("brevo: %v", err)
	}
	if got := brevo.(*SMTPEmailProvider).cfg.FromEmail; got != "shop@saajan.test" {
		t.Errorf("brevo from = %q, want the shared address", got)
	}

	sg, err := buildEmailProvider("sendgrid", envMap(env), quietLog())
	if err != nil {
		t.Fatalf("sendgrid: %v", err)
	}
	if got := sg.(*SendGridEmailProvider).fromEmail; got != "shop@saajan.test" {
		t.Errorf("sendgrid from = %q, want the shared address", got)
	}
}

func TestEmailProviderNamesCoversPresetsAndBuiltins(t *testing.T) {
	names := EmailProviderNames()
	joined := strings.Join(names, ",")

	for _, want := range []string{"sendgrid", "smtp", "simulated", "brevo", "mailjet", "resend", "ses"} {
		if !strings.Contains(joined, want) {
			t.Errorf("EmailProviderNames() is missing %q: %v", want, names)
		}
	}
}
