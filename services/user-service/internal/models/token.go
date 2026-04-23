package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
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

// RefreshToken represents a refresh token
type RefreshToken struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"index;not null" json:"user_id"`
	Token     string    `gorm:"uniqueIndex;not null" json:"token"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// TableName specifies the table name for RefreshToken
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsValid checks if the refresh token is still valid
func (rt *RefreshToken) IsValid() bool {
	return rt.RevokedAt == nil && time.Now().Before(rt.ExpiresAt)
}
