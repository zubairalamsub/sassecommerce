package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ecommerce/notification-service/internal/models"
)

// mailgunStub captures what the provider posts so the request can be asserted
// without contacting Mailgun.
type mailgunStub struct {
	server *httptest.Server

	path     string
	authUser string
	authPass string
	form     map[string]string

	status int
	body   string
}

func newMailgunStub(status int, body string) *mailgunStub {
	s := &mailgunStub{status: status, body: body, form: map[string]string{}}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.path = r.URL.Path
		s.authUser, s.authPass, _ = r.BasicAuth()
		_ = r.ParseForm()
		for k := range r.PostForm {
			s.form[k] = r.PostForm.Get(k)
		}
		w.WriteHeader(s.status)
		_, _ = w.Write([]byte(s.body))
	}))
	return s
}

func (s *mailgunStub) close() { s.server.Close() }

func newStubbedMailgun(t *testing.T, stub *mailgunStub, cfg MailgunConfig) *MailgunAPIProvider {
	t.Helper()
	p := NewMailgunAPIProvider(cfg, quietLog())
	p.baseURL = stub.server.URL
	return p
}

func mailgunEmail() *models.Notification {
	return &models.Notification{
		Recipient: "buyer@example.test",
		Subject:   "Order confirmed",
		Body:      "<p>Thanks for your order</p>",
	}
}

func TestMailgunAPISendHappyPath(t *testing.T) {
	stub := newMailgunStub(http.StatusOK, `{"id":"<20260904.1@mg.saajan.test>","message":"Queued. Thank you."}`)
	defer stub.close()

	p := newStubbedMailgun(t, stub, MailgunConfig{
		APIKey:    "key-abc123",
		Domain:    "mg.saajan.test",
		FromEmail: "noreply@mg.saajan.test",
		FromName:  "Saajan Store",
	})

	res, err := p.Send(mailgunEmail())

	if err != nil {
		t.Fatalf("Send = %v", err)
	}
	if !res.Success {
		t.Fatalf("result = %+v, want success", res)
	}
	// Mailgun's own id must be preserved so a message can be traced in their
	// dashboard rather than only in our logs.
	if res.MessageID != "<20260904.1@mg.saajan.test>" {
		t.Errorf("message id = %q, want Mailgun's own id", res.MessageID)
	}
	if res.ProviderName != "mailgun_api" {
		t.Errorf("provider name = %q, want mailgun_api", res.ProviderName)
	}
}

// The domain is part of the URL path, so getting it wrong is a 404 rather than
// a helpful error.
func TestMailgunAPIPostsToTheDomainEndpoint(t *testing.T) {
	stub := newMailgunStub(http.StatusOK, `{"id":"1"}`)
	defer stub.close()

	p := newStubbedMailgun(t, stub, MailgunConfig{
		APIKey: "key-abc123", Domain: "mg.saajan.test", FromEmail: "noreply@mg.saajan.test",
	})

	if _, err := p.Send(mailgunEmail()); err != nil {
		t.Fatalf("Send = %v", err)
	}

	if stub.path != "/v3/mg.saajan.test/messages" {
		t.Errorf("path = %q, want /v3/<domain>/messages", stub.path)
	}
}

// Mailgun authenticates with HTTP Basic, username literally "api".
func TestMailgunAPIUsesBasicAuthWithApiUsername(t *testing.T) {
	stub := newMailgunStub(http.StatusOK, `{"id":"1"}`)
	defer stub.close()

	p := newStubbedMailgun(t, stub, MailgunConfig{
		APIKey: "key-abc123", Domain: "mg.saajan.test", FromEmail: "noreply@mg.saajan.test",
	})

	if _, err := p.Send(mailgunEmail()); err != nil {
		t.Fatalf("Send = %v", err)
	}

	if stub.authUser != "api" {
		t.Errorf("basic auth user = %q, want the literal \"api\"", stub.authUser)
	}
	if stub.authPass != "key-abc123" {
		t.Errorf("basic auth password = %q, want the API key", stub.authPass)
	}
	// Guard against the key being sent somewhere it should not be.
	if got := stub.form["api_key"]; got != "" {
		t.Errorf("API key leaked into the form body as %q", got)
	}
}

func TestMailgunAPISendsTheMessageFields(t *testing.T) {
	stub := newMailgunStub(http.StatusOK, `{"id":"1"}`)
	defer stub.close()

	p := newStubbedMailgun(t, stub, MailgunConfig{
		APIKey: "k", Domain: "mg.saajan.test",
		FromEmail: "noreply@mg.saajan.test", FromName: "Saajan Store",
	})

	if _, err := p.Send(mailgunEmail()); err != nil {
		t.Fatalf("Send = %v", err)
	}

	if got := stub.form["to"]; got != "buyer@example.test" {
		t.Errorf("to = %q", got)
	}
	if got := stub.form["subject"]; got != "Order confirmed" {
		t.Errorf("subject = %q", got)
	}
	if got := stub.form["html"]; got != "<p>Thanks for your order</p>" {
		t.Errorf("html = %q, want the body sent as HTML", got)
	}
	if got := stub.form["from"]; got != `"Saajan Store" <noreply@mg.saajan.test>` {
		t.Errorf("from = %q, want a quoted display name", got)
	}
}

// A comma in the display name would split the address if it were not quoted.
func TestMailgunAPIQuotesAwkwardDisplayNames(t *testing.T) {
	stub := newMailgunStub(http.StatusOK, `{"id":"1"}`)
	defer stub.close()

	p := newStubbedMailgun(t, stub, MailgunConfig{
		APIKey: "k", Domain: "d.test",
		FromEmail: "noreply@d.test", FromName: `Saajan, Inc. "Store"`,
	})

	if _, err := p.Send(mailgunEmail()); err != nil {
		t.Fatalf("Send = %v", err)
	}

	from := stub.form["from"]
	if !strings.HasPrefix(from, `"`) || !strings.HasSuffix(from, "<noreply@d.test>") {
		t.Errorf("from = %q, want the name quoted and the address in angle brackets", from)
	}
	if !strings.Contains(from, `\"Store\"`) {
		t.Errorf("from = %q, want inner quotes escaped", from)
	}
}

func TestMailgunAPIOmitsDisplayNameWhenUnset(t *testing.T) {
	stub := newMailgunStub(http.StatusOK, `{"id":"1"}`)
	defer stub.close()

	p := newStubbedMailgun(t, stub, MailgunConfig{APIKey: "k", Domain: "d.test", FromEmail: "noreply@d.test"})

	if _, err := p.Send(mailgunEmail()); err != nil {
		t.Fatalf("Send = %v", err)
	}

	if got := stub.form["from"]; got != "noreply@d.test" {
		t.Errorf("from = %q, want the bare address", got)
	}
}

// Mailgun explains a rejection in a JSON "message" field; surfacing it is the
// difference between a usable error and "it didn't work".
func TestMailgunAPISurfacesRejectionReason(t *testing.T) {
	stub := newMailgunStub(http.StatusUnauthorized, `{"message":"Forbidden"}`)
	defer stub.close()

	p := newStubbedMailgun(t, stub, MailgunConfig{APIKey: "wrong", Domain: "d.test", FromEmail: "a@d.test"})

	res, err := p.Send(mailgunEmail())

	if err != nil {
		t.Fatalf("Send = %v, want a soft failure so a chain can fall through", err)
	}
	if res.Success {
		t.Fatal("result reports success on a 401")
	}
	if !strings.Contains(res.Error, "401") || !strings.Contains(res.Error, "Forbidden") {
		t.Errorf("error = %q, want the status and Mailgun's reason", res.Error)
	}
}

// A proxy or gateway can return non-JSON; the body must not be swallowed.
func TestMailgunAPIFallsBackToRawBodyOnNonJSONError(t *testing.T) {
	stub := newMailgunStub(http.StatusBadGateway, `<html>gateway exploded</html>`)
	defer stub.close()

	p := newStubbedMailgun(t, stub, MailgunConfig{APIKey: "k", Domain: "d.test", FromEmail: "a@d.test"})

	res, _ := p.Send(mailgunEmail())

	if res.Success {
		t.Fatal("result reports success on a 502")
	}
	if !strings.Contains(res.Error, "gateway exploded") {
		t.Errorf("error = %q, want the raw body preserved", res.Error)
	}
}

func TestMailgunAPIValidatesBeforeCalling(t *testing.T) {
	cases := []struct {
		name string
		cfg  MailgunConfig
		note *models.Notification
		want string
	}{
		{
			name: "no recipient",
			cfg:  MailgunConfig{APIKey: "k", Domain: "d.test", FromEmail: "a@d.test"},
			note: &models.Notification{Subject: "s", Body: "b"},
			want: "recipient",
		},
		{
			name: "no api key",
			cfg:  MailgunConfig{Domain: "d.test", FromEmail: "a@d.test"},
			note: mailgunEmail(),
			want: "api key",
		},
		{
			name: "no domain and none derivable",
			cfg:  MailgunConfig{APIKey: "k", FromEmail: "not-an-address"},
			note: mailgunEmail(),
			want: "domain",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stub := newMailgunStub(http.StatusOK, `{"id":"1"}`)
			defer stub.close()
			p := newStubbedMailgun(t, stub, c.cfg)

			res, err := p.Send(c.note)

			if err != nil {
				t.Fatalf("Send = %v, want a soft failure", err)
			}
			if res.Success {
				t.Fatalf("result reports success for %s", c.name)
			}
			if !strings.Contains(res.Error, c.want) {
				t.Errorf("error = %q, want it to mention %q", res.Error, c.want)
			}
			if stub.path != "" {
				t.Errorf("provider called Mailgun despite invalid config (path %q)", stub.path)
			}
		})
	}
}

// The domain has to match the from address, so deriving it removes a field an
// operator would otherwise have to keep in sync.
func TestMailgunAPIDerivesDomainFromFromAddress(t *testing.T) {
	p := NewMailgunAPIProvider(MailgunConfig{APIKey: "k", FromEmail: "noreply@mg.saajan.test"}, quietLog())

	if p.cfg.Domain != "mg.saajan.test" {
		t.Errorf("domain = %q, want it derived from the from address", p.cfg.Domain)
	}
}

func TestMailgunAPIExplicitDomainWins(t *testing.T) {
	p := NewMailgunAPIProvider(MailgunConfig{
		APIKey: "k", Domain: "explicit.test", FromEmail: "noreply@derived.test",
	}, quietLog())

	if p.cfg.Domain != "explicit.test" {
		t.Errorf("domain = %q, want the explicit one", p.cfg.Domain)
	}
}

// EU accounts live on a different host; getting this wrong authenticates
// against the wrong region and fails confusingly.
func TestMailgunAPIRegionHost(t *testing.T) {
	cases := []struct {
		name string
		host string
		want string
	}{
		{name: "defaults to US", host: "", want: "https://api.mailgun.net/v3/d.test/messages"},
		{name: "bare EU host", host: "api.eu.mailgun.net", want: "https://api.eu.mailgun.net/v3/d.test/messages"},
		{name: "full URL accepted", host: "https://api.eu.mailgun.net", want: "https://api.eu.mailgun.net/v3/d.test/messages"},
		{name: "trailing slash trimmed", host: "https://api.eu.mailgun.net/", want: "https://api.eu.mailgun.net/v3/d.test/messages"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := NewMailgunAPIProvider(MailgunConfig{APIKey: "k", Domain: "d.test", APIHost: c.host}, quietLog())
			if got := p.endpoint(); got != c.want {
				t.Errorf("endpoint = %q, want %q", got, c.want)
			}
		})
	}
}

// A Mailgun outage must hand off to the next provider like any other failure.
func TestMailgunAPIFailureFallsThroughAChain(t *testing.T) {
	stub := newMailgunStub(http.StatusInternalServerError, `{"message":"server error"}`)
	defer stub.close()

	down := newStubbedMailgun(t, stub, MailgunConfig{APIKey: "k", Domain: "d.test", FromEmail: "a@d.test"})
	backup := &scriptedProvider{name: "mailjet", mode: "succeed"}
	chain := NewFailoverProvider(models.ChannelEmail, quietLog(), down, backup)

	res, _ := chain.Send(mailgunEmail())

	if !res.Success || res.ProviderName != "mailjet" {
		t.Errorf("result = %+v, want the backup to deliver", res)
	}
}

func TestDomainFromAddress(t *testing.T) {
	cases := []struct{ in, want string }{
		{in: "noreply@mg.saajan.test", want: "mg.saajan.test"},
		{in: "a@b@c.test", want: "c.test"},
		{in: "no-at-sign", want: ""},
		{in: "trailing@", want: ""},
		{in: "", want: ""},
	}

	for _, c := range cases {
		if got := domainFromAddress(c.in); got != c.want {
			t.Errorf("domainFromAddress(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// Guards the credential never reaching a log or the URL — it belongs only in
// the Authorization header.
func TestMailgunAPIKeyStaysInTheAuthHeader(t *testing.T) {
	stub := newMailgunStub(http.StatusOK, `{"id":"1"}`)
	defer stub.close()

	const key = "key-super-secret"
	p := newStubbedMailgun(t, stub, MailgunConfig{APIKey: key, Domain: "d.test", FromEmail: "a@d.test"})

	if _, err := p.Send(mailgunEmail()); err != nil {
		t.Fatalf("Send = %v", err)
	}

	if strings.Contains(stub.path, key) {
		t.Error("API key appeared in the URL path")
	}
	for field, value := range stub.form {
		if strings.Contains(value, key) {
			t.Errorf("API key appeared in form field %q", field)
		}
	}
	// It belongs in exactly one place: the basic-auth credential.
	if stub.authPass != key {
		t.Errorf("basic auth password = %q, want the API key", stub.authPass)
	}
}
