package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"gorm.io/gorm"
)

// TokenRepository defines the interface for token data operations
type TokenRepository interface {
	CreateVerificationToken(ctx context.Context, token *models.VerificationToken) error
	GetVerificationTokenByToken(ctx context.Context, token string) (*models.VerificationToken, error)
	InvalidateVerificationTokens(ctx context.Context, userID string) error
	MarkVerificationTokenUsed(ctx context.Context, id string) error

	CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error
	GetPasswordResetTokenByToken(ctx context.Context, token string) (*models.PasswordResetToken, error)
	InvalidatePasswordResetTokens(ctx context.Context, userID string) error
	MarkPasswordResetTokenUsed(ctx context.Context, id string) error
}

type tokenRepository struct {
	db *gorm.DB
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *gorm.DB) TokenRepository {
	return &tokenRepository{db: db}
}

// CreateVerificationToken creates a new email verification token
func (r *tokenRepository) CreateVerificationToken(ctx context.Context, token *models.VerificationToken) error {
	result := r.db.WithContext(ctx).Create(token)
	return result.Error
}

// GetVerificationTokenByToken retrieves a verification token by its token string
func (r *tokenRepository) GetVerificationTokenByToken(ctx context.Context, token string) (*models.VerificationToken, error) {
	var vt models.VerificationToken
	result := r.db.WithContext(ctx).
		Where("token = ? AND used_at IS NULL", token).
		First(&vt)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("verification token not found")
		}
		return nil, result.Error
	}

	return &vt, nil
}

// InvalidateVerificationTokens marks all unused verification tokens for a user as used
func (r *tokenRepository) InvalidateVerificationTokens(ctx context.Context, userID string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.VerificationToken{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Update("used_at", now)
	return result.Error
}

// MarkVerificationTokenUsed marks a specific verification token as used
func (r *tokenRepository) MarkVerificationTokenUsed(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.VerificationToken{}).
		Where("id = ?", id).
		Update("used_at", now)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("verification token not found")
	}
	return nil
}

// CreatePasswordResetToken creates a new password reset token
func (r *tokenRepository) CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error {
	result := r.db.WithContext(ctx).Create(token)
	return result.Error
}

// GetPasswordResetTokenByToken retrieves a password reset token by its token string
func (r *tokenRepository) GetPasswordResetTokenByToken(ctx context.Context, token string) (*models.PasswordResetToken, error) {
	var prt models.PasswordResetToken
	result := r.db.WithContext(ctx).
		Where("token = ? AND used_at IS NULL", token).
		First(&prt)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("password reset token not found")
		}
		return nil, result.Error
	}

	return &prt, nil
}

// InvalidatePasswordResetTokens marks all unused reset tokens for a user as used
func (r *tokenRepository) InvalidatePasswordResetTokens(ctx context.Context, userID string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Update("used_at", now)
	return result.Error
}

// MarkPasswordResetTokenUsed marks a specific password reset token as used
func (r *tokenRepository) MarkPasswordResetTokenUsed(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.PasswordResetToken{}).
		Where("id = ?", id).
		Update("used_at", now)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("password reset token not found")
	}
	return nil
}
