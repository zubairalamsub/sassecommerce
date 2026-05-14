package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

func newSilentLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func TestNormalizeBDMSISDN(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"+8801712345678", "8801712345678"},
		{"8801712345678", "8801712345678"},
		{"01712345678", "8801712345678"},
		{"017-1234-5678", "8801712345678"},
		{" +880 1712 345 678 ", "8801712345678"},
		{"+15551234567", "15551234567"}, // non-BD passes through (without +)
	}
	for _, tc := range cases {
		if got := normalizeBDMSISDN(tc.in); got != tc.want {
			t.Errorf("normalizeBDMSISDN(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSSLWireless_Send_Success(t *testing.T) {
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json content-type, got %s", ct)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"SUCCESS","status_code":200,"error_message":"",
			"smsinfo":[{"sms_status":"SUCCESS","msisdn":"8801712345678","csms_id":"abc","reference_id":"REF-123"}]
		}`))
	}))
	defer srv.Close()

	p := NewSSLWirelessSMSProvider(SSLWirelessConfig{
		APIToken: "tok",
		SID:      "BRAND",
		Endpoint: srv.URL,
	}, newSilentLogger())

	res, err := p.Send(&models.Notification{
		ID:        "abc",
		Recipient: "01712345678",
		Subject:   "Verify",
		Body:      "Your code is 1234",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success, got %+v", res)
	}
	if res.MessageID != "REF-123" {
		t.Errorf("expected message id REF-123, got %s", res.MessageID)
	}

	if capturedBody["api_token"] != "tok" {
		t.Errorf("expected api_token=tok, got %v", capturedBody["api_token"])
	}
	if capturedBody["sid"] != "BRAND" {
		t.Errorf("expected sid=BRAND, got %v", capturedBody["sid"])
	}
	if capturedBody["msisdn"] != "8801712345678" {
		t.Errorf("expected normalized msisdn, got %v", capturedBody["msisdn"])
	}
	if got := capturedBody["sms"].(string); !strings.Contains(got, "Verify") || !strings.Contains(got, "1234") {
		t.Errorf("expected combined subject+body in sms, got %q", got)
	}
}

func TestSSLWireless_Send_FailureFromAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"FAILED","status_code":400,"error_message":"Invalid msisdn"}`))
	}))
	defer srv.Close()

	p := NewSSLWirelessSMSProvider(SSLWirelessConfig{APIToken: "t", SID: "X", Endpoint: srv.URL}, newSilentLogger())
	res, err := p.Send(&models.Notification{Recipient: "01700000000", Body: "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Error("expected failure")
	}
	if !strings.Contains(res.Error, "Invalid msisdn") {
		t.Errorf("expected error to mention API message, got %q", res.Error)
	}
}

func TestSSLWireless_Send_RejectsEmptyRecipient(t *testing.T) {
	p := NewSSLWirelessSMSProvider(SSLWirelessConfig{APIToken: "t", SID: "X"}, newSilentLogger())
	res, err := p.Send(&models.Notification{Recipient: "", Body: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Error("expected failure for empty recipient")
	}
}

func TestSSLWireless_TruncatesLongBody(t *testing.T) {
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		_, _ = w.Write([]byte(`{"status":"SUCCESS","status_code":200}`))
	}))
	defer srv.Close()

	p := NewSSLWirelessSMSProvider(SSLWirelessConfig{APIToken: "t", SID: "X", Endpoint: srv.URL}, newSilentLogger())
	long := strings.Repeat("a", 1500)
	if _, err := p.Send(&models.Notification{Recipient: "01700000000", Body: long}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := capturedBody["sms"].(string)
	if len(got) > 1000 {
		t.Errorf("expected sms truncated to 1000 chars, got %d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected truncated body to end with ..., got suffix %q", got[len(got)-10:])
	}
}
