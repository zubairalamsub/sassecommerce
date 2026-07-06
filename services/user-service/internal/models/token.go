package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TokenClaims represents JWT claims
type TokenClaims struct {
	UserID   string   `json:"user_id"`
	TenantID string   `json:"tenant_id"`
	Email    string   `json:"email"`
	Role     UserRole `json:"role"`
	jwt.RegisteredClaims
}

// TokenConfig holds JWT configuration
type TokenConfig struct {
	SecretKey      string
	ExpirationTime time.Duration
	Issuer         string
}

// RefreshToken represents a refresh token. One row is one live session:
// the device metadata (user agent, IP) lets users review and revoke
// individual sessions.
type RefreshToken struct {
	ID         string     `gorm:"primaryKey" json:"id"`
	UserID     string     `gorm:"index;not null" json:"user_id"`
	TenantID   string     `gorm:"index" json:"tenant_id"`
	Token      string     `gorm:"uniqueIndex;not null" json:"token"`
	UserAgent  string     `json:"user_agent"`
	IPAddress  string     `json:"ip_address"`
	ExpiresAt  time.Time  `gorm:"not null" json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt time.Time  `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

// RefreshTokenRequest is the body for POST /auth/refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// SessionResponse is the public shape of a login session. The refresh token
// string itself is never included.
type SessionResponse struct {
	ID         string    `json:"id"`
	UserAgent  string    `json:"user_agent"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	Current    bool      `json:"current"`
}

// TableName specifies the table name for RefreshToken
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsValid checks if the refresh token is still valid
func (rt *RefreshToken) IsValid() bool {
	return rt.RevokedAt == nil && time.Now().Before(rt.ExpiresAt)
}

// VerificationToken represents an email verification token
type VerificationToken struct {
	ID        string     `gorm:"primaryKey" json:"id"`
	UserID    string     `gorm:"index;not null" json:"user_id"`
	TenantID  string     `gorm:"index;not null" json:"tenant_id"`
	Token     string     `gorm:"uniqueIndex;not null" json:"token"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// TableName specifies the table name for VerificationToken
func (VerificationToken) TableName() string {
	return "verification_tokens"
}

// BeforeCreate generates a UUID if not already set
func (vt *VerificationToken) BeforeCreate(tx *gorm.DB) error {
	if vt.ID == "" {
		vt.ID = uuid.New().String()
	}
	return nil
}

// IsValid checks if the verification token is still valid
func (vt *VerificationToken) IsValid() bool {
	return vt.UsedAt == nil && time.Now().Before(vt.ExpiresAt)
}

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        string     `gorm:"primaryKey" json:"id"`
	UserID    string     `gorm:"index;not null" json:"user_id"`
	TenantID  string     `gorm:"index;not null" json:"tenant_id"`
	Token     string     `gorm:"uniqueIndex;not null" json:"token"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// TableName specifies the table name for PasswordResetToken
func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}

// BeforeCreate generates a UUID if not already set
func (prt *PasswordResetToken) BeforeCreate(tx *gorm.DB) error {
	if prt.ID == "" {
		prt.ID = uuid.New().String()
	}
	return nil
}

// IsValid checks if the password reset token is still valid
func (prt *PasswordResetToken) IsValid() bool {
	return prt.UsedAt == nil && time.Now().Before(prt.ExpiresAt)
}
