package repository

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/ecommerce/user-service/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTwoFactorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=busy_timeout(5000)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.TwoFactorSecret{}, &models.TwoFactorBackupCode{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestMigrateLegacyPlaintextSecrets(t *testing.T) {
	ctx := context.Background()
	db := newTwoFactorTestDB(t)
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes

	// Rows written before the key existed: base64-encoded plaintext.
	legacyRepo, err := NewTwoFactorRepository(db, nil)
	if err != nil {
		t.Fatalf("legacy repo: %v", err)
	}
	if err := legacyRepo.Upsert(ctx, &models.TwoFactorSecret{
		UserID: "legacy-user", Secret: "LEGACYTOTPSECRET", Pending: "PENDINGSECRET",
	}); err != nil {
		t.Fatalf("legacy upsert: %v", err)
	}

	// A row written with the key must be left alone.
	keyedRepo, err := NewTwoFactorRepository(db, key)
	if err != nil {
		t.Fatalf("keyed repo: %v", err)
	}
	if err := keyedRepo.Upsert(ctx, &models.TwoFactorSecret{
		UserID: "modern-user", Secret: "MODERNTOTPSECRET",
	}); err != nil {
		t.Fatalf("modern upsert: %v", err)
	}

	migrated, err := keyedRepo.MigrateLegacyPlaintextSecrets(ctx)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if migrated != 1 {
		t.Errorf("migrated = %d, want 1", migrated)
	}

	// Legacy secrets decrypt to their original values through the keyed repo.
	got, err := keyedRepo.GetByUserID(ctx, "legacy-user")
	if err != nil {
		t.Fatalf("get legacy: %v", err)
	}
	if got.Secret != "LEGACYTOTPSECRET" || got.Pending != "PENDINGSECRET" {
		t.Errorf("legacy secrets = (%q, %q), want originals", got.Secret, got.Pending)
	}

	// At rest, the stored value is no longer base64(plaintext).
	var raw models.TwoFactorSecret
	if err := db.Where("user_id = ?", "legacy-user").First(&raw).Error; err != nil {
		t.Fatalf("raw read: %v", err)
	}
	if raw.Secret == base64.StdEncoding.EncodeToString([]byte("LEGACYTOTPSECRET")) {
		t.Error("stored secret is still base64 plaintext after migration")
	}

	// Modern row still decrypts and a second run migrates nothing.
	if got, err := keyedRepo.GetByUserID(ctx, "modern-user"); err != nil || got.Secret != "MODERNTOTPSECRET" {
		t.Errorf("modern secret after migration = (%v, %v), want MODERNTOTPSECRET", got, err)
	}
	if again, err := keyedRepo.MigrateLegacyPlaintextSecrets(ctx); err != nil || again != 0 {
		t.Errorf("second run = (%d, %v), want (0, nil) — migration must be idempotent", again, err)
	}
}

func TestMigrateLegacyPlaintextSecrets_NoKeyIsNoop(t *testing.T) {
	db := newTwoFactorTestDB(t)
	repo, err := NewTwoFactorRepository(db, nil)
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	if n, err := repo.MigrateLegacyPlaintextSecrets(context.Background()); err != nil || n != 0 {
		t.Errorf("no-key migration = (%d, %v), want (0, nil)", n, err)
	}
}
