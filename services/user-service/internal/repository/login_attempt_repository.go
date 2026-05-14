package repository

import (
	"context"
	"strings"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"gorm.io/gorm"
)

// LoginAttemptRepository defines persistence operations for login attempts.
//
// The repository is keyed by lowercase email so that case variations of the
// same address are not used to circumvent the per-email lockout.
type LoginAttemptRepository interface {
	Create(ctx context.Context, attempt *models.LoginAttempt) error
	CountRecentFailedByEmail(ctx context.Context, tenantID, email string, since time.Time) (int, error)
	CountRecentFailedByIP(ctx context.Context, ip string, since time.Time) (int, error)
	// CountRecentByEmailAndReason counts attempts (regardless of success)
	// for an email since the given time, optionally filtered by reason.
	// Used by forgot-password rate limiting.
	CountRecentByEmailAndReason(ctx context.Context, tenantID, email, reason string, since time.Time) (int, error)
	MarkSuccess(ctx context.Context, email string) error
	DeleteOlderThan(ctx context.Context, before time.Time) error
}

type loginAttemptRepository struct {
	db *gorm.DB
}

// NewLoginAttemptRepository constructs a LoginAttemptRepository backed by GORM.
func NewLoginAttemptRepository(db *gorm.DB) LoginAttemptRepository {
	return &loginAttemptRepository{db: db}
}

// Create persists a login attempt. Email is normalized to lowercase to keep
// counters consistent regardless of how the user typed their address.
func (r *loginAttemptRepository) Create(ctx context.Context, attempt *models.LoginAttempt) error {
	if attempt == nil {
		return nil
	}
	attempt.Email = strings.ToLower(strings.TrimSpace(attempt.Email))
	return r.db.WithContext(ctx).Create(attempt).Error
}

// CountRecentFailedByEmail returns the number of failed attempts for an email
// (within a tenant) since the given timestamp.
func (r *loginAttemptRepository) CountRecentFailedByEmail(ctx context.Context, tenantID, email string, since time.Time) (int, error) {
	var count int64
	normalized := strings.ToLower(strings.TrimSpace(email))
	q := r.db.WithContext(ctx).
		Model(&models.LoginAttempt{}).
		Where("email = ? AND successful = ? AND created_at >= ?", normalized, false, since)
	if tenantID != "" {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// CountRecentFailedByIP returns the number of failed attempts originating
// from the given IP since the given timestamp.
func (r *loginAttemptRepository) CountRecentFailedByIP(ctx context.Context, ip string, since time.Time) (int, error) {
	if ip == "" {
		return 0, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.LoginAttempt{}).
		Where("ip_address = ? AND successful = ? AND created_at >= ?", ip, false, since).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// CountRecentByEmailAndReason counts attempts for an email since the given
// timestamp, optionally narrowed by reason. Passing an empty reason counts
// all attempts. Used for forgot-password throttling, which lives in the
// same table for schema simplicity.
func (r *loginAttemptRepository) CountRecentByEmailAndReason(ctx context.Context, tenantID, email, reason string, since time.Time) (int, error) {
	var count int64
	normalized := strings.ToLower(strings.TrimSpace(email))
	q := r.db.WithContext(ctx).
		Model(&models.LoginAttempt{}).
		Where("email = ? AND created_at >= ?", normalized, since)
	if tenantID != "" {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if reason != "" {
		q = q.Where("reason = ?", reason)
	}
	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// MarkSuccess clears the failed-attempt counter for an email by marking all
// outstanding failed attempts as superseded. We delete the rows here rather
// than carrying a flag — keeping the audit trail of successful logins via
// future Create(successful=true) entries is sufficient.
func (r *loginAttemptRepository) MarkSuccess(ctx context.Context, email string) error {
	normalized := strings.ToLower(strings.TrimSpace(email))
	return r.db.WithContext(ctx).
		Where("email = ? AND successful = ?", normalized, false).
		Delete(&models.LoginAttempt{}).Error
}

// DeleteOlderThan removes attempt rows older than the cutoff. Intended to be
// called by a periodic cleanup job to keep the table small.
func (r *loginAttemptRepository) DeleteOlderThan(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&models.LoginAttempt{}).Error
}
