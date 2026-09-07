package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEmailProviderConfig_ToResponse_MasksSecret(t *testing.T) {
	cfg := EmailProviderConfig{
		ID: "cfg-1", TenantID: "tenant-1", Provider: "smtp", Enabled: true, Priority: 1,
		Host: "smtp.example.com", Port: 587, Username: "svc-account",
		Secret:    "super-secret-app-password",
		FromEmail: "noreply@example.com", FromName: "Example Shop",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}

	resp := cfg.ToResponse(false)

	assert.True(t, resp.SecretSet)
	assert.Equal(t, "…word", resp.SecretHint, "hint should be the ciphertext's last 4 characters, prefixed with an ellipsis")
	assert.False(t, resp.Inherited)
	assert.Equal(t, "smtp.example.com", resp.Host)
	assert.Equal(t, 587, resp.Port)
}

func TestEmailProviderConfig_ToResponse_NoSecretMeansUnset(t *testing.T) {
	cfg := EmailProviderConfig{ID: "cfg-1", Provider: "simulated"}

	resp := cfg.ToResponse(true)

	assert.False(t, resp.SecretSet)
	assert.Empty(t, resp.SecretHint)
	assert.True(t, resp.Inherited, "inherited must reflect the caller's argument, since it distinguishes a tenant's own row from the platform default")
}

func TestEmailProviderConfig_SecretHintValue(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		want   string
	}{
		{name: "empty secret has no hint", secret: "", want: ""},
		{name: "secret at the boundary length is hidden entirely", secret: "abcd", want: ""},
		{name: "secret one over the boundary reveals four characters", secret: "abcde", want: "…bcde"},
		{name: "long secret reveals only the last four characters", secret: "sk_live_abcdef1234", want: "…1234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := EmailProviderConfig{Secret: tt.secret}
			assert.Equal(t, tt.want, cfg.SecretHintValue())
		})
	}
}

// The secret must never reach the wire under any JSON encoding of the
// storage-side type, not just through ToResponse. Both the bson tag
// (secret_enc) and the json tag (-) exist specifically to stop that, per the
// type's own doc comment -- this is the regression test for that guarantee,
// including a defense-in-depth check that the literal value never appears
// anywhere in the encoded bytes.
func TestEmailProviderConfig_JSON_NeverSerializesSecret(t *testing.T) {
	cfg := EmailProviderConfig{
		ID: "cfg-1", TenantID: "tenant-1", Provider: "smtp",
		Secret: "super-secret-app-password-XYZ",
	}

	raw, err := json.Marshal(cfg)
	assert.NoError(t, err)

	assert.NotContains(t, string(raw), "super-secret-app-password-XYZ", "the raw secret value must never leak into JSON output")
	assert.NotContains(t, string(raw), "secret", "no key resembling the secret field should appear at all")

	var decoded map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &decoded))
	assert.NotContains(t, decoded, "secret")
	assert.NotContains(t, decoded, "secret_enc")
}

func TestEmailProviderConfigResponse_JSON_WireFieldNames(t *testing.T) {
	cfg := EmailProviderConfig{
		ID: "cfg-1", TenantID: "tenant-1", Provider: "mailjet", Enabled: true, Priority: 1,
		Host: "in-v3.mailjet.com", Port: 587, Username: "api-key", Secret: "topsecret1234",
		FromEmail: "noreply@example.com", FromName: "Example",
	}

	raw, err := json.Marshal(cfg.ToResponse(false))
	assert.NoError(t, err)

	var decoded map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &decoded))

	for _, key := range []string{
		"id", "tenant_id", "provider", "enabled", "priority", "host", "port",
		"username", "from_email", "from_name", "secret_set", "secret_hint",
		"inherited", "created_at", "updated_at",
	} {
		assert.Contains(t, decoded, key, "expected the wire field %q", key)
	}
}

func TestUpsertEmailProviderRequest_BindingValidation(t *testing.T) {
	validate := newBindingValidator()

	assert.Error(t, validate.Struct(UpsertEmailProviderRequest{}), "provider is required")
	assert.NoError(t, validate.Struct(UpsertEmailProviderRequest{Provider: "smtp"}))
}

// Port is optional on update: an operator toggling Enabled should not have to
// resend Host/Port. *int distinguishes "leave the stored port alone" (nil)
// from "set the port to 0" (a pointer to zero) -- a plain int could not
// represent "unspecified" at all, and would silently zero the port on every
// partial update that omits it.
func TestUpsertEmailProviderRequest_PortDistinguishesAbsentFromZero(t *testing.T) {
	var withZeroPort UpsertEmailProviderRequest
	assert.NoError(t, json.Unmarshal([]byte(`{"provider":"smtp","port":0}`), &withZeroPort))
	if assert.NotNil(t, withZeroPort.Port) {
		assert.Equal(t, 0, *withZeroPort.Port)
	}

	var withoutPort UpsertEmailProviderRequest
	assert.NoError(t, json.Unmarshal([]byte(`{"provider":"smtp"}`), &withoutPort))
	assert.Nil(t, withoutPort.Port, "an omitted port must stay nil so the service leaves the stored value alone")
}

func TestTestEmailProviderRequest_RequiresAWellFormedEmail(t *testing.T) {
	validate := newBindingValidator()

	tests := []struct {
		name    string
		req     TestEmailProviderRequest
		wantErr bool
	}{
		{name: "valid request passes", req: TestEmailProviderRequest{Provider: "smtp", To: "ops@example.com"}, wantErr: false},
		{name: "missing provider fails", req: TestEmailProviderRequest{To: "ops@example.com"}, wantErr: true},
		{name: "missing recipient fails", req: TestEmailProviderRequest{Provider: "smtp"}, wantErr: true},
		{name: "malformed email fails", req: TestEmailProviderRequest{Provider: "smtp", To: "not-an-email"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPlatformScope_IsNotAPlausibleTenantID(t *testing.T) {
	// Documents the sentinel's exact value: any change here would silently
	// change which stored config rows count as "the platform default".
	assert.Equal(t, "_platform", PlatformScope)
}
