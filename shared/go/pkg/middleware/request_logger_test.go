package middleware

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactJSONBody(t *testing.T) {
	sensitive := SensitiveFieldSet(DefaultSensitiveJSONFields)

	tests := []struct {
		name         string
		body         string
		wantRedacted []string // JSON paths whose values must be [REDACTED]
		wantKept     []string // substrings that must survive
	}{
		{
			name:         "login request",
			body:         `{"email":"a@b.com","password":"hunter2"}`,
			wantRedacted: []string{"password"},
			wantKept:     []string{"a@b.com"},
		},
		{
			name:         "change password",
			body:         `{"old_password":"a","new_password":"b"}`,
			wantRedacted: []string{"old_password", "new_password"},
		},
		{
			name:         "login response with tokens",
			body:         `{"token":"eyJhbGciOi...","refresh_token":"abc","user":{"email":"a@b.com"}}`,
			wantRedacted: []string{"token", "refresh_token"},
			wantKept:     []string{"a@b.com"},
		},
		{
			name:         "nested and case-insensitive",
			body:         `{"data":{"Password":"x","items":[{"reset_token":"y"}]}}`,
			wantRedacted: []string{"Password", "reset_token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactJSONBody(tt.body, sensitive)

			var parsed any
			if err := json.Unmarshal([]byte(got), &parsed); err != nil {
				t.Fatalf("result is not valid JSON: %v", err)
			}
			for _, key := range tt.wantRedacted {
				if !strings.Contains(got, `"`+key+`":"[REDACTED]"`) {
					t.Errorf("expected %q to be redacted, got: %s", key, got)
				}
			}
			for _, keep := range tt.wantKept {
				if !strings.Contains(got, keep) {
					t.Errorf("expected %q to be kept, got: %s", keep, got)
				}
			}
		})
	}
}

func TestRedactJSONBodyNonJSON(t *testing.T) {
	sensitive := SensitiveFieldSet(DefaultSensitiveJSONFields)

	for _, body := range []string{"password=hunter2&user=a", "<html>x</html>", `{"truncated": "js`} {
		got := RedactJSONBody(body, sensitive)
		if strings.Contains(got, "hunter2") || got == body {
			t.Errorf("non-JSON body must be omitted, got: %s", got)
		}
	}
}
