package service

import (
	"errors"
	"net/smtp"
	"strings"
	"testing"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

func quietLog() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

// scriptedProvider is a NotificationProvider with a pre-decided outcome, so a
// failover chain can be driven through every branch.
type scriptedProvider struct {
	name string
	// one of: succeed, softFail, hardErr, nilResult
	mode  string
	calls int
}

func (p *scriptedProvider) Channel() models.Channel { return models.ChannelEmail }
func (p *scriptedProvider) Name() string            { return p.name }

func (p *scriptedProvider) Send(n *models.Notification) (*ProviderResult, error) {
	p.calls++
	switch p.mode {
	case "succeed":
		return &ProviderResult{ProviderName: p.name, MessageID: p.name + "-msg", Success: true}, nil
	case "softFail":
		return &ProviderResult{ProviderName: p.name, Success: false, Error: "quota exceeded"}, nil
	case "hardErr":
		return nil, errors.New("connection refused")
	case "nilResult":
		return nil, nil
	}
	panic("unknown mode " + p.mode)
}

// unnamedProvider has no Name method, so the chain must still label it.
type unnamedProvider struct{ mode string }

func (p *unnamedProvider) Channel() models.Channel { return models.ChannelEmail }
func (p *unnamedProvider) Send(n *models.Notification) (*ProviderResult, error) {
	if p.mode == "succeed" {
		return &ProviderResult{Success: true}, nil
	}
	return &ProviderResult{Success: false, Error: "nope"}, nil
}

func testEmail() *models.Notification {
	return &models.Notification{
		Recipient: "buyer@example.test",
		Subject:   "Reset your password",
		Body:      "<p>Click the link</p>",
	}
}

// ---------------------------------------------------------------- failover

// A single provider needs no wrapper — wrapping would only add a layer of
// logging and indirection.
func TestNewFailoverProviderUnwrapsSingleAndEmpty(t *testing.T) {
	only := &scriptedProvider{name: "brevo", mode: "succeed"}

	if got := NewFailoverProvider(models.ChannelEmail, quietLog(), only); got != only {
		t.Errorf("single provider = %#v, want it returned unwrapped", got)
	}
	if got := NewFailoverProvider(models.ChannelEmail, quietLog()); got != nil {
		t.Errorf("no providers = %#v, want nil", got)
	}
}

// The primary handles traffic and nothing else is touched.
func TestFailoverUsesPrimaryAndStopsThere(t *testing.T) {
	primary := &scriptedProvider{name: "brevo", mode: "succeed"}
	backup := &scriptedProvider{name: "sendgrid", mode: "succeed"}
	chain := NewFailoverProvider(models.ChannelEmail, quietLog(), primary, backup)

	res, err := chain.Send(testEmail())

	if err != nil {
		t.Fatalf("Send = %v", err)
	}
	if !res.Success || res.ProviderName != "brevo" {
		t.Errorf("result = %+v, want success from brevo", res)
	}
	if backup.calls != 0 {
		t.Errorf("backup was called %d times, want 0", backup.calls)
	}
}

// This is the case the chain exists for: a vendor hits its free-tier quota and
// the next one absorbs the send.
func TestFailoverMovesPastSoftFailure(t *testing.T) {
	primary := &scriptedProvider{name: "brevo", mode: "softFail"}
	backup := &scriptedProvider{name: "sendgrid", mode: "succeed"}
	chain := NewFailoverProvider(models.ChannelEmail, quietLog(), primary, backup)

	res, err := chain.Send(testEmail())

	if err != nil {
		t.Fatalf("Send = %v", err)
	}
	if !res.Success || res.ProviderName != "sendgrid" {
		t.Errorf("result = %+v, want success from sendgrid", res)
	}
	if primary.calls != 1 || backup.calls != 1 {
		t.Errorf("calls: primary=%d backup=%d, want 1 and 1", primary.calls, backup.calls)
	}
}

// A provider can fail three distinct ways; all must advance the chain rather
// than abort it.
func TestFailoverTreatsEveryFailureShapeAsRetryable(t *testing.T) {
	for _, mode := range []string{"softFail", "hardErr", "nilResult"} {
		t.Run(mode, func(t *testing.T) {
			primary := &scriptedProvider{name: "brevo", mode: mode}
			backup := &scriptedProvider{name: "sendgrid", mode: "succeed"}
			chain := NewFailoverProvider(models.ChannelEmail, quietLog(), primary, backup)

			res, err := chain.Send(testEmail())

			if err != nil {
				t.Fatalf("Send = %v", err)
			}
			if res == nil || !res.Success {
				t.Fatalf("result = %+v, want the backup to deliver", res)
			}
			if backup.calls != 1 {
				t.Errorf("backup calls = %d, want 1", backup.calls)
			}
		})
	}
}

// Exhaustion must report every reason, not just the last: with a chain, one
// error string hides which vendors were tried and why each refused.
func TestFailoverReportsAllFailuresWhenExhausted(t *testing.T) {
	a := &scriptedProvider{name: "brevo", mode: "softFail"}
	b := &scriptedProvider{name: "sendgrid", mode: "hardErr"}
	chain := NewFailoverProvider(models.ChannelEmail, quietLog(), a, b)

	res, err := chain.Send(testEmail())

	if err != nil {
		t.Fatalf("Send = %v, want the failure reported in the result", err)
	}
	if res.Success {
		t.Fatal("result reports success though every provider failed")
	}
	if !strings.Contains(res.Error, "brevo") || !strings.Contains(res.Error, "sendgrid") {
		t.Errorf("error %q should name both providers", res.Error)
	}
	if !strings.Contains(res.Error, "quota exceeded") || !strings.Contains(res.Error, "connection refused") {
		t.Errorf("error %q should carry both underlying reasons", res.Error)
	}
	if !strings.Contains(res.Error, "all 2 providers failed") {
		t.Errorf("error %q should state how many were tried", res.Error)
	}
}

// Every provider in the chain is tried, not just the first two.
func TestFailoverWalksTheWholeChain(t *testing.T) {
	a := &scriptedProvider{name: "brevo", mode: "softFail"}
	b := &scriptedProvider{name: "mailjet", mode: "softFail"}
	c := &scriptedProvider{name: "simulated", mode: "succeed"}
	chain := NewFailoverProvider(models.ChannelEmail, quietLog(), a, b, c)

	res, _ := chain.Send(testEmail())

	if !res.Success || res.ProviderName != "simulated" {
		t.Errorf("result = %+v, want the third provider to deliver", res)
	}
	if a.calls != 1 || b.calls != 1 || c.calls != 1 {
		t.Errorf("calls = %d/%d/%d, want each tried once", a.calls, b.calls, c.calls)
	}
}

func TestFailoverLabelsProviderWithoutAName(t *testing.T) {
	chain := NewFailoverProvider(models.ChannelEmail, quietLog(),
		&unnamedProvider{mode: "fail"}, &unnamedProvider{mode: "fail"})

	res, _ := chain.Send(testEmail())

	if !strings.Contains(res.Error, "provider-1") || !strings.Contains(res.Error, "provider-2") {
		t.Errorf("error %q should fall back to positional labels", res.Error)
	}
}

func TestFailoverExposesChannelAndLength(t *testing.T) {
	chain := NewFailoverProvider(models.ChannelEmail, quietLog(),
		&scriptedProvider{name: "a", mode: "succeed"},
		&scriptedProvider{name: "b", mode: "succeed"})

	if chain.Channel() != models.ChannelEmail {
		t.Errorf("channel = %v, want email", chain.Channel())
	}
	if fp, ok := chain.(*FailoverProvider); !ok || fp.Len() != 2 {
		t.Errorf("chain = %#v, want a FailoverProvider of length 2", chain)
	}
}

// ------------------------------------------------------------------- SMTP

// fakeSMTPClient records the conversation so the send path can be asserted
// without a relay.
type fakeSMTPClient struct {
	authed   bool
	from     string
	rcpts    []string
	data     strings.Builder
	quit     bool
	closed   bool
	failAt   string // "auth" | "mail" | "rcpt" | "data" | "write" | "commit"
	dataOpen bool
}

func (c *fakeSMTPClient) Auth(smtp.Auth) error {
	if c.failAt == "auth" {
		return errors.New("535 bad credentials")
	}
	c.authed = true
	return nil
}
func (c *fakeSMTPClient) Mail(from string) error {
	if c.failAt == "mail" {
		return errors.New("550 sender rejected")
	}
	c.from = from
	return nil
}
func (c *fakeSMTPClient) Rcpt(to string) error {
	if c.failAt == "rcpt" {
		return errors.New("550 recipient rejected")
	}
	c.rcpts = append(c.rcpts, to)
	return nil
}
func (c *fakeSMTPClient) Data() (dataWriter, error) {
	if c.failAt == "data" {
		return nil, errors.New("451 try later")
	}
	c.dataOpen = true
	return &fakeDataWriter{c: c}, nil
}
func (c *fakeSMTPClient) Quit() error  { c.quit = true; return nil }
func (c *fakeSMTPClient) Close() error { c.closed = true; return nil }

type fakeDataWriter struct{ c *fakeSMTPClient }

func (w *fakeDataWriter) Write(p []byte) (int, error) {
	if w.c.failAt == "write" {
		return 0, errors.New("broken pipe")
	}
	return w.c.data.Write(p)
}
func (w *fakeDataWriter) Close() error {
	if w.c.failAt == "commit" {
		return errors.New("552 message too large")
	}
	return nil
}

func newTestSMTPProvider(client *fakeSMTPClient, dialErr error) *SMTPEmailProvider {
	p := NewSMTPEmailProvider(SMTPConfig{
		Name:      "brevo",
		Host:      "smtp-relay.brevo.com",
		Port:      587,
		Username:  "user",
		Password:  "pass",
		FromEmail: "noreply@saajan.test",
		FromName:  "Saajan Store",
	}, quietLog())
	p.dial = func(SMTPConfig) (smtpClient, error) {
		if dialErr != nil {
			return nil, dialErr
		}
		return client, nil
	}
	return p
}

func TestSMTPSendHappyPath(t *testing.T) {
	client := &fakeSMTPClient{}
	p := newTestSMTPProvider(client, nil)

	res, err := p.Send(testEmail())

	if err != nil {
		t.Fatalf("Send = %v", err)
	}
	if !res.Success || res.ProviderName != "brevo" {
		t.Errorf("result = %+v, want success labelled brevo", res)
	}
	if !client.authed {
		t.Error("provider did not authenticate")
	}
	if client.from != "noreply@saajan.test" {
		t.Errorf("MAIL FROM = %q, want the configured sender", client.from)
	}
	if len(client.rcpts) != 1 || client.rcpts[0] != "buyer@example.test" {
		t.Errorf("RCPT TO = %v, want the recipient", client.rcpts)
	}
	if !client.quit {
		t.Error("provider did not QUIT after a committed message")
	}
	if !client.closed {
		t.Error("connection was not closed")
	}
}

// A recipient-less notification must be rejected before any connection is
// opened.
func TestSMTPRejectsMissingRecipientWithoutDialing(t *testing.T) {
	p := NewSMTPEmailProvider(SMTPConfig{Name: "brevo", Host: "h", Port: 587}, quietLog())
	dialed := false
	p.dial = func(SMTPConfig) (smtpClient, error) {
		dialed = true
		return &fakeSMTPClient{}, nil
	}

	res, err := p.Send(&models.Notification{Subject: "s", Body: "b"})

	if err != nil {
		t.Fatalf("Send = %v", err)
	}
	if res.Success {
		t.Error("result reports success with no recipient")
	}
	if dialed {
		t.Error("provider dialled the relay despite having no recipient")
	}
}

func TestSMTPRejectsMissingHost(t *testing.T) {
	p := NewSMTPEmailProvider(SMTPConfig{Name: "brevo"}, quietLog())

	res, err := p.Send(testEmail())

	if err != nil {
		t.Fatalf("Send = %v", err)
	}
	if res.Success || !strings.Contains(res.Error, "host") {
		t.Errorf("result = %+v, want an unconfigured-host failure", res)
	}
}

// Every stage of the SMTP conversation can fail, and each must surface as a
// soft failure so a failover chain can move on rather than abort.
func TestSMTPFailuresAreRetryableSoftFailures(t *testing.T) {
	stages := []struct {
		failAt string
		want   string
	}{
		{failAt: "auth", want: "auth failed"},
		{failAt: "mail", want: "MAIL FROM failed"},
		{failAt: "rcpt", want: "RCPT TO failed"},
		{failAt: "data", want: "DATA failed"},
		{failAt: "write", want: "write failed"},
		{failAt: "commit", want: "message rejected"},
	}

	for _, s := range stages {
		t.Run(s.failAt, func(t *testing.T) {
			p := newTestSMTPProvider(&fakeSMTPClient{failAt: s.failAt}, nil)

			res, err := p.Send(testEmail())

			if err != nil {
				t.Fatalf("Send returned a hard error %v; a chain could not fall through", err)
			}
			if res.Success {
				t.Fatalf("result reports success despite failing at %s", s.failAt)
			}
			if !strings.Contains(res.Error, s.want) {
				t.Errorf("error %q should mention %q", res.Error, s.want)
			}
		})
	}
}

func TestSMTPDialFailureIsASoftFailure(t *testing.T) {
	p := newTestSMTPProvider(nil, errors.New("no route to host"))

	res, err := p.Send(testEmail())

	if err != nil {
		t.Fatalf("Send = %v, want a soft failure", err)
	}
	if res.Success || !strings.Contains(res.Error, "connect failed") {
		t.Errorf("result = %+v, want a connect failure", res)
	}
}

// The whole point of the chain: an SMTP relay that is down must hand off.
func TestSMTPFailureFallsThroughAChain(t *testing.T) {
	down := newTestSMTPProvider(nil, errors.New("no route to host"))
	backup := &scriptedProvider{name: "sendgrid", mode: "succeed"}
	chain := NewFailoverProvider(models.ChannelEmail, quietLog(), down, backup)

	res, _ := chain.Send(testEmail())

	if !res.Success || res.ProviderName != "sendgrid" {
		t.Errorf("result = %+v, want the backup to deliver", res)
	}
}

// Headers must be RFC 2047 encoded: the templates emit Bangla and the taka
// sign, and raw non-ASCII in a header is illegal and gets mangled in transit.
func TestSMTPEncodesNonASCIIHeaders(t *testing.T) {
	client := &fakeSMTPClient{}
	p := NewSMTPEmailProvider(SMTPConfig{
		Name: "brevo", Host: "h", Port: 587,
		FromEmail: "noreply@saajan.test", FromName: "সাজান স্টোর",
	}, quietLog())
	p.dial = func(SMTPConfig) (smtpClient, error) { return client, nil }

	n := testEmail()
	n.Subject = "আপনার অর্ডার ৳1,799"

	if _, err := p.Send(n); err != nil {
		t.Fatalf("Send = %v", err)
	}

	msg := client.data.String()
	subject := headerLine(msg, "Subject:")
	if subject == "" {
		t.Fatalf("no Subject header in:\n%s", msg)
	}
	if strings.Contains(subject, "৳") || strings.Contains(subject, "আপনার") {
		t.Errorf("Subject carries raw non-ASCII: %q", subject)
	}
	if !strings.Contains(subject, "=?utf-8?") {
		t.Errorf("Subject %q is not RFC 2047 encoded", subject)
	}
	if from := headerLine(msg, "From:"); !strings.Contains(from, "=?utf-8?") {
		t.Errorf("From %q should encode the non-ASCII display name", from)
	}
	// An ASCII-only subject should stay readable rather than be encoded.
	if !strings.Contains(msg, "Content-Type: text/html") {
		t.Errorf("message should declare an HTML body:\n%s", msg)
	}
}

func TestSMTPLeavesASCIISubjectReadable(t *testing.T) {
	client := &fakeSMTPClient{}
	p := newTestSMTPProvider(client, nil)

	if _, err := p.Send(testEmail()); err != nil {
		t.Fatalf("Send = %v", err)
	}

	if subject := headerLine(client.data.String(), "Subject:"); subject != "Subject: Reset your password" {
		t.Errorf("subject line = %q, want it left as plain text", subject)
	}
}

// Bare newlines are illegal in SMTP data; the HTML templates use \n.
func TestSMTPNormalisesLineEndings(t *testing.T) {
	client := &fakeSMTPClient{}
	p := newTestSMTPProvider(client, nil)

	n := testEmail()
	n.Body = "line one\nline two\r\nline three\n"

	if _, err := p.Send(n); err != nil {
		t.Fatalf("Send = %v", err)
	}

	body := client.data.String()
	if strings.Contains(body, "\r\r\n") {
		t.Error("existing CRLF was doubled")
	}
	// No \n may appear without a preceding \r.
	for i, r := range body {
		if r == '\n' && (i == 0 || body[i-1] != '\r') {
			t.Fatalf("bare newline at offset %d", i)
		}
	}
}

func headerLine(msg, prefix string) string {
	for _, line := range strings.Split(msg, "\r\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
		if line == "" {
			break // end of headers
		}
	}
	return ""
}
