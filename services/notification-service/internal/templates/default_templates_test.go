package templates

import (
	"strings"
	"testing"
	"text/template"
)

// TestDefaults_AllParseAndRender exercises every starter-pack template through
// the same text/template parser the consumer uses, with sample vars covering
// every field the registry references. A parse failure here would surface as
// a runtime 500 in production, so we catch it during CI instead.
func TestDefaults_AllParseAndRender(t *testing.T) {
	vars := map[string]interface{}{
		"TenantName":      "Saajan",
		"BrandColor":      "#006A4E",
		"FrontendBaseURL": "https://shop.example.com",
		"CustomerName":    "Bob",
		"UserName":        "bob",
		"VerifyURL":       "https://shop.example.com/verify?t=abc",
		"ResetURL":        "https://shop.example.com/reset?t=abc",
		"OrderID":         "ORD-1",
		"Total":           "৳1,250.00",
		"PaymentMethod":   "bKash",
		"TrackingNumber":  "TRK-1",
		"Carrier":         "Pathao",
		"Reason":          "Customer requested",
		"RefundAmount":    "৳500.00",
		"ProductName":     "Sample shirt",
		"SKU":             "SKU-1",
		"CurrentQuantity": 5,
		"Items": []map[string]interface{}{
			{"Name": "Shirt", "Quantity": 2, "Subtotal": "৳800.00"},
		},
	}

	for _, d := range Defaults() {
		t.Run(string(d.Type), func(t *testing.T) {
			subj, err := template.New("s").Option("missingkey=zero").Parse(d.SubjectTemplate)
			if err != nil {
				t.Fatalf("subject parse: %v", err)
			}
			body, err := template.New("b").Option("missingkey=zero").Parse(d.BodyTemplate)
			if err != nil {
				t.Fatalf("body parse: %v", err)
			}

			var sb strings.Builder
			if err := subj.Execute(&sb, vars); err != nil {
				t.Fatalf("subject execute: %v", err)
			}
			if sb.Len() == 0 {
				t.Errorf("subject rendered empty")
			}

			var bb strings.Builder
			if err := body.Execute(&bb, vars); err != nil {
				t.Fatalf("body execute: %v", err)
			}
			out := bb.String()
			// Smoke checks: every email body must contain the chrome we
			// designed in emailShell so the look is consistent.
			if !strings.Contains(out, "<!DOCTYPE html>") {
				t.Errorf("body missing doctype")
			}
			if !strings.Contains(out, "Saajan") {
				t.Errorf("body missing tenant name substitution")
			}
		})
	}
}

// TestDefaults_Count is a tripwire: if someone removes a starter template the
// counter on the admin install dialog ("This will create 10…") must be
// updated to match — this test makes that obvious.
func TestDefaults_Count(t *testing.T) {
	got := len(Defaults())
	if got < 10 {
		t.Errorf("expected at least 10 starter templates, got %d", got)
	}
}
