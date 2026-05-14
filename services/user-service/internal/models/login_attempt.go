package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoginAttempt records every authentication attempt — successful or not.
// Tracked by email (not user ID) so that attempts against non-existent
// accounts are still counted, which prevents account enumeration via
// observation of lockout behavior.
type LoginAttempt struct {
	ID         string    `gorm:"primaryKey" json:"id"`
	TenantID   string    `gorm:"index" json:"tenant_id"`
	Email      string    `gorm:"index:idx_login_attempt_email" json:"email"`
	IPAddress  string    `gorm:"index:idx_login_attempt_ip" json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	Successful bool      `gorm:"index" json:"successful"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

// Reason values for LoginAttempt.Reason.
const (
	LoginAttemptReasonSuccess         = "success"
	LoginAttemptReasonInvalidPassword = "invalid_password"
	LoginAttemptReasonUnknownEmail    = "unknown_email"
	LoginAttemptReasonUserInactive    = "user_inactive"
	LoginAttemptReasonRateLimited     = "rate_limited"
	LoginAttemptReasonAccountLocked   = "account_locked"
)

// TableName specifies the table name for LoginAttempt.
func (LoginAttempt) TableName() string {
	return "login_attempts"
}

// BeforeCreate generates a UUID and timestamp if not already set.
func (la *LoginAttempt) BeforeCreate(tx *gorm.DB) error {
	if la.ID == "" {
		la.ID = uuid.New().String()
	}
	if la.CreatedAt.IsZero() {
		la.CreatedAt = time.Now().UTC()
	}
	return nil
}
