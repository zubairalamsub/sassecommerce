package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TwoFactorSecret stores a user's TOTP secret and 2FA state.
//
// Secrets are stored encrypted (AES-GCM). The repository handles encryption
// transparently; callers see/set plaintext.
type TwoFactorSecret struct {
	ID          string     `gorm:"primaryKey" json:"id"`
	UserID      string     `gorm:"uniqueIndex;not null" json:"user_id"`
	TenantID    string     `gorm:"index;not null" json:"tenant_id"`
	Secret      string     `gorm:"column:secret;type:text" json:"-"` // confirmed secret (encrypted)
	Pending     string     `gorm:"column:pending;type:text" json:"-"` // pending secret during setup (encrypted)
	Enabled     bool       `gorm:"default:false" json:"enabled"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (TwoFactorSecret) TableName() string { return "two_factor_secrets" }

func (t *TwoFactorSecret) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

// TwoFactorBackupCode is a single-use recovery code (stored hashed).
type TwoFactorBackupCode struct {
	ID        string     `gorm:"primaryKey" json:"id"`
	UserID    string     `gorm:"index;not null" json:"user_id"`
	CodeHash  string     `gorm:"not null" json:"-"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (TwoFactorBackupCode) TableName() string { return "two_factor_backup_codes" }

func (b *TwoFactorBackupCode) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

// === Request/response DTOs ===

// TwoFactorSetupResponse is returned by POST /auth/2fa/setup. The user scans
// QRCodeImage (a data URL) into their authenticator app and confirms with a code.
type TwoFactorSetupResponse struct {
	Secret          string `json:"secret"`            // base32 secret for manual entry
	ProvisioningURI string `json:"provisioning_uri"`  // otpauth:// URI for QR rendering
	QRCodeImage     string `json:"qr_code_image"`     // data:image/png;base64,...
}

type TwoFactorConfirmRequest struct {
	Code string `json:"code" binding:"required"`
}

// TwoFactorConfirmResponse returns the backup codes (shown once).
type TwoFactorConfirmResponse struct {
	BackupCodes []string `json:"backup_codes"`
}

type TwoFactorDisableRequest struct {
	Password string `json:"password" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

// TwoFactorVerifyRequest is the body for POST /auth/2fa/verify (login completion).
type TwoFactorVerifyRequest struct {
	ChallengeToken string `json:"challenge_token" binding:"required"`
	Code           string `json:"code" binding:"required"`
}

// TwoFactorChallenge is embedded in LoginResponse when 2FA is required.
type TwoFactorChallenge struct {
	Required       bool   `json:"required"`
	ChallengeToken string `json:"challenge_token,omitempty"`
}
