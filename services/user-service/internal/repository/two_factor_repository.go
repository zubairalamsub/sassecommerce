package repository

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"gorm.io/gorm"
)

// TwoFactorRepository persists 2FA secrets and backup codes. Secrets are
// encrypted at rest using AES-GCM with a key supplied at construction time.
type TwoFactorRepository interface {
	GetByUserID(ctx context.Context, userID string) (*models.TwoFactorSecret, error)
	Upsert(ctx context.Context, secret *models.TwoFactorSecret) error
	Delete(ctx context.Context, userID string) error
	UpdateLastUsed(ctx context.Context, userID string) error

	CreateBackupCodes(ctx context.Context, codes []models.TwoFactorBackupCode) error
	DeleteBackupCodes(ctx context.Context, userID string) error
	ListUnusedBackupCodes(ctx context.Context, userID string) ([]models.TwoFactorBackupCode, error)
	MarkBackupCodeUsed(ctx context.Context, id string) error
}

type twoFactorRepository struct {
	db    *gorm.DB
	gcm   cipher.AEAD
	noEnc bool
}

// NewTwoFactorRepository constructs a repo. If encryptionKey is empty, secrets
// are stored as base64-encoded plaintext (acceptable for local dev only — log
// a warning at startup).
func NewTwoFactorRepository(db *gorm.DB, encryptionKey []byte) (TwoFactorRepository, error) {
	r := &twoFactorRepository{db: db}
	if len(encryptionKey) == 0 {
		r.noEnc = true
		return r, nil
	}
	if l := len(encryptionKey); l != 16 && l != 24 && l != 32 {
		return nil, errors.New("encryption key must be 16, 24, or 32 bytes")
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	r.gcm = gcm
	return r, nil
}

func (r *twoFactorRepository) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if r.noEnc {
		return base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
	}
	nonce := make([]byte, r.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := r.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (r *twoFactorRepository) decrypt(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return "", err
	}
	if r.noEnc {
		return string(raw), nil
	}
	ns := r.gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := raw[:ns], raw[ns:]
	plaintext, err := r.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (r *twoFactorRepository) GetByUserID(ctx context.Context, userID string) (*models.TwoFactorSecret, error) {
	var t models.TwoFactorSecret
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("2fa not configured")
		}
		return nil, err
	}
	if t.Secret != "" {
		dec, derr := r.decrypt(t.Secret)
		if derr != nil {
			return nil, derr
		}
		t.Secret = dec
	}
	if t.Pending != "" {
		dec, derr := r.decrypt(t.Pending)
		if derr != nil {
			return nil, derr
		}
		t.Pending = dec
	}
	return &t, nil
}

func (r *twoFactorRepository) Upsert(ctx context.Context, secret *models.TwoFactorSecret) error {
	encS, err := r.encrypt(secret.Secret)
	if err != nil {
		return err
	}
	encP, err := r.encrypt(secret.Pending)
	if err != nil {
		return err
	}

	// Make a shallow copy so we don't leak ciphertext back to callers.
	row := *secret
	row.Secret = encS
	row.Pending = encP

	var existing models.TwoFactorSecret
	err = r.db.WithContext(ctx).Where("user_id = ?", secret.UserID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(&row).Error
	}
	if err != nil {
		return err
	}
	row.ID = existing.ID
	row.CreatedAt = existing.CreatedAt
	return r.db.WithContext(ctx).Save(&row).Error
}

func (r *twoFactorRepository) Delete(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.TwoFactorSecret{}).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.TwoFactorBackupCode{}).Error
}

func (r *twoFactorRepository) UpdateLastUsed(ctx context.Context, userID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.TwoFactorSecret{}).
		Where("user_id = ?", userID).
		Update("last_used_at", now).Error
}

func (r *twoFactorRepository) CreateBackupCodes(ctx context.Context, codes []models.TwoFactorBackupCode) error {
	if len(codes) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&codes).Error
}

func (r *twoFactorRepository) DeleteBackupCodes(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.TwoFactorBackupCode{}).Error
}

func (r *twoFactorRepository) ListUnusedBackupCodes(ctx context.Context, userID string) ([]models.TwoFactorBackupCode, error) {
	var codes []models.TwoFactorBackupCode
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND used_at IS NULL", userID).
		Find(&codes).Error
	return codes, err
}

func (r *twoFactorRepository) MarkBackupCodeUsed(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.TwoFactorBackupCode{}).
		Where("id = ?", id).
		Update("used_at", now).Error
}
