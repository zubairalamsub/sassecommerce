package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"gorm.io/gorm"
)

// RefreshTokenRepository defines the interface for refresh token / session storage.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	GetByToken(ctx context.Context, token string) (*models.RefreshToken, error)
	GetByID(ctx context.Context, id string) (*models.RefreshToken, error)
	ListByUser(ctx context.Context, userID string) ([]models.RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeByToken(ctx context.Context, token string) error
	RevokeAllByUser(ctx context.Context, userID string) error
	UpdateLastUsed(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type refreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository constructs a refresh token repository.
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *refreshTokenRepository) GetByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	var rt models.RefreshToken
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&rt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("refresh token not found")
		}
		return nil, err
	}
	return &rt, nil
}

func (r *refreshTokenRepository) GetByID(ctx context.Context, id string) (*models.RefreshToken, error) {
	var rt models.RefreshToken
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("refresh token not found")
		}
		return nil, err
	}
	return &rt, nil
}

func (r *refreshTokenRepository) ListByUser(ctx context.Context, userID string) ([]models.RefreshToken, error) {
	var tokens []models.RefreshToken
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("last_used_at DESC").
		Find(&tokens).Error
	return tokens, err
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id string) error {
	now := time.Now()
	res := r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("refresh token not found or already revoked")
	}
	return nil
}

func (r *refreshTokenRepository) RevokeByToken(ctx context.Context, token string) error {
	now := time.Now()
	res := r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("token = ? AND revoked_at IS NULL", token).
		Update("revoked_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("refresh token not found or already revoked")
	}
	return nil
}

func (r *refreshTokenRepository) RevokeAllByUser(ctx context.Context, userID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

func (r *refreshTokenRepository) UpdateLastUsed(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}

func (r *refreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("expires_at < ? OR revoked_at < ?", time.Now(), time.Now().Add(-30*24*time.Hour)).
		Delete(&models.RefreshToken{})
	return res.RowsAffected, res.Error
}
