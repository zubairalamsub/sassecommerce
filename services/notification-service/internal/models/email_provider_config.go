package models

import "time"

// PlatformScope is the tenant id under which the platform-wide default
// provider chain is stored. It reuses the "_platform" sentinel the media
// storage already uses for super_admin-owned resources, and is not a legal
// tenant id, so a real tenant can never collide with it.
const PlatformScope = "_platform"

// EmailProviderConfig is one configured relay in a chain. Chains resolve
// tenant-first: a tenant with any enabled provider uses its own, otherwise it
// inherits the platform default. Priority orders the chain, low first.
//
// Secret holds the vendor's password / API secret and is encrypted at rest.
// It is never returned by the API — responses carry SecretSet and SecretHint
// instead, so an operator can see that a credential exists without it being
// readable back out of the system.
type EmailProviderConfig struct {
	ID       string `bson:"_id,omitempty" json:"id"`
	TenantID string `bson:"tenant_id" json:"tenant_id"`
	// Provider is a key from service.EmailProviderNames() — a vendor preset
	// ("mailjet", "brevo"), "sendgrid", explicit "smtp", or "simulated".
	Provider string `bson:"provider" json:"provider"`
	Enabled  bool   `bson:"enabled" json:"enabled"`
	Priority int    `bson:"priority" json:"priority"`

	// Host/Port are optional for a preset vendor and required for "smtp".
	Host string `bson:"host,omitempty" json:"host,omitempty"`
	Port int    `bson:"port,omitempty" json:"port,omitempty"`

	Username string `bson:"username,omitempty" json:"username,omitempty"`
	// Secret is the encrypted credential. The bson tag keeps it out of any
	// struct that is marshalled to the client, and the json tag is "-" so an
	// accidental direct encode cannot leak it.
	Secret string `bson:"secret_enc,omitempty" json:"-"`

	FromEmail string `bson:"from_email,omitempty" json:"from_email,omitempty"`
	FromName  string `bson:"from_name,omitempty" json:"from_name,omitempty"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// EmailProviderConfigResponse is the client-facing shape. It deliberately has
// no field that can carry the secret.
type EmailProviderConfigResponse struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Provider string `json:"provider"`
	Enabled  bool   `json:"enabled"`
	Priority int    `json:"priority"`

	Host      string `json:"host,omitempty"`
	Port      int    `json:"port,omitempty"`
	Username  string `json:"username,omitempty"`
	FromEmail string `json:"from_email,omitempty"`
	FromName  string `json:"from_name,omitempty"`

	// SecretSet reports whether a credential is stored; SecretHint shows only
	// the last four characters so an operator can tell two keys apart without
	// the value being recoverable.
	SecretSet  bool   `json:"secret_set"`
	SecretHint string `json:"secret_hint,omitempty"`

	// Inherited marks a row served from the platform default rather than
	// configured by this tenant, so the UI can show it read-only.
	Inherited bool `json:"inherited"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse masks the secret.
func (c *EmailProviderConfig) ToResponse(inherited bool) *EmailProviderConfigResponse {
	return &EmailProviderConfigResponse{
		ID:         c.ID,
		TenantID:   c.TenantID,
		Provider:   c.Provider,
		Enabled:    c.Enabled,
		Priority:   c.Priority,
		Host:       c.Host,
		Port:       c.Port,
		Username:   c.Username,
		FromEmail:  c.FromEmail,
		FromName:   c.FromName,
		SecretSet:  c.Secret != "",
		SecretHint: c.SecretHintValue(),
		Inherited:  inherited,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

// SecretHintValue returns a non-reversible hint for a stored secret. It is
// derived from the ciphertext, never the plaintext, so producing the hint
// never requires decrypting.
func (c *EmailProviderConfig) SecretHintValue() string {
	const hintLen = 4
	if len(c.Secret) <= hintLen {
		return ""
	}
	return "…" + c.Secret[len(c.Secret)-hintLen:]
}

// UpsertEmailProviderRequest is the write shape. Secret is optional on update:
// omitting it keeps the stored credential, so an operator can change the port
// or toggle Enabled without re-entering the key.
type UpsertEmailProviderRequest struct {
	Provider  string `json:"provider" binding:"required"`
	Enabled   *bool  `json:"enabled"`
	Priority  *int   `json:"priority"`
	Host      string `json:"host"`
	Port      *int   `json:"port"`
	Username  string `json:"username"`
	Secret    string `json:"secret"`
	FromEmail string `json:"from_email"`
	FromName  string `json:"from_name"`
}

// TestEmailProviderRequest asks the service to send a probe message through a
// single configured provider.
type TestEmailProviderRequest struct {
	Provider string `json:"provider" binding:"required"`
	To       string `json:"to" binding:"required,email"`
}
