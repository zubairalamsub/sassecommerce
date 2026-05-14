package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"strings"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// TwoFactorService handles TOTP enrollment, verification, and recovery.
type TwoFactorService interface {
	BeginSetup(ctx context.Context, userID, tenantID, accountName string) (*models.TwoFactorSetupResponse, error)
	ConfirmSetup(ctx context.Context, userID string, code string) (*models.TwoFactorConfirmResponse, error)
	Disable(ctx context.Context, userID string) error
	IsEnabled(ctx context.Context, userID string) (bool, error)
	Verify(ctx context.Context, userID, code string) (bool, error)
}

type twoFactorService struct {
	repo          repository.TwoFactorRepository
	logger        *logrus.Logger
	issuer        string
	now           func() time.Time
	codeRand      func([]byte) (int, error) // injectable for tests
	kafkaProducer KafkaPublisher            // optional; nil disables event publishing
}

// TwoFactorServiceOption configures TwoFactorService.
type TwoFactorServiceOption func(*twoFactorService)

func WithTwoFactorIssuer(issuer string) TwoFactorServiceOption {
	return func(s *twoFactorService) { s.issuer = issuer }
}

// WithTwoFactorKafkaPublisher attaches an event publisher so the service can
// emit TwoFactorEnabled / TwoFactorDisabled events onto user-events. When
// omitted (typical in unit tests), publishing is silently skipped.
func WithTwoFactorKafkaPublisher(publisher KafkaPublisher) TwoFactorServiceOption {
	return func(s *twoFactorService) { s.kafkaProducer = publisher }
}

func NewTwoFactorService(repo repository.TwoFactorRepository, logger *logrus.Logger, opts ...TwoFactorServiceOption) TwoFactorService {
	s := &twoFactorService{
		repo:     repo,
		logger:   logger,
		issuer:   "Saajan",
		now:      time.Now,
		codeRand: rand.Read,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// BeginSetup generates a new pending TOTP secret and returns the provisioning
// URI plus a base64-encoded QR code image that the frontend can render directly.
func (s *twoFactorService) BeginSetup(ctx context.Context, userID, tenantID, accountName string) (*models.TwoFactorSetupResponse, error) {
	existing, err := s.repo.GetByUserID(ctx, userID)
	if err == nil && existing.Enabled {
		return nil, errors.New("2fa is already enabled; disable it first to re-enroll")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: accountName,
		Period:      30,
		SecretSize:  20,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	row := &models.TwoFactorSecret{
		UserID:   userID,
		TenantID: tenantID,
		Pending:  key.Secret(),
	}
	// Preserve confirmed secret if 2FA exists but wasn't enabled (mid-setup re-attempt)
	if existing != nil {
		row.Secret = existing.Secret
		row.Enabled = existing.Enabled
		row.ConfirmedAt = existing.ConfirmedAt
	}
	if err := s.repo.Upsert(ctx, row); err != nil {
		return nil, fmt.Errorf("failed to store pending secret: %w", err)
	}

	img, err := key.Image(220, 220)
	if err != nil {
		return nil, fmt.Errorf("failed to render QR: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode QR PNG: %w", err)
	}
	qrDataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	return &models.TwoFactorSetupResponse{
		Secret:          key.Secret(),
		ProvisioningURI: key.URL(),
		QRCodeImage:     qrDataURL,
	}, nil
}

// ConfirmSetup validates a code against the pending secret and enables 2FA.
// On success it generates and returns a new set of single-use backup codes.
func (s *twoFactorService) ConfirmSetup(ctx context.Context, userID, code string) (*models.TwoFactorConfirmResponse, error) {
	row, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, errors.New("no pending 2fa setup")
	}
	if row.Pending == "" {
		return nil, errors.New("no pending 2fa setup")
	}
	if !totp.Validate(code, row.Pending) {
		return nil, errors.New("invalid code")
	}

	// Promote pending → secret, enable, clear pending.
	confirmed := s.now()
	row.Secret = row.Pending
	row.Pending = ""
	row.Enabled = true
	row.ConfirmedAt = &confirmed
	if err := s.repo.Upsert(ctx, row); err != nil {
		return nil, fmt.Errorf("failed to enable 2fa: %w", err)
	}

	// Replace any existing backup codes.
	if err := s.repo.DeleteBackupCodes(ctx, userID); err != nil {
		s.logger.WithError(err).Warn("Failed to clear old backup codes")
	}
	plain, hashed, err := s.generateBackupCodes(userID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}
	if err := s.repo.CreateBackupCodes(ctx, hashed); err != nil {
		return nil, fmt.Errorf("failed to save backup codes: %w", err)
	}

	// Publish TwoFactorEnabled so the audit log captures the enrollment.
	s.publish2FAEvent(ctx, "TwoFactorEnabled", map[string]interface{}{
		"tenant_id": row.TenantID,
		"user_id":   userID,
		"method":    "totp",
	})

	return &models.TwoFactorConfirmResponse{BackupCodes: plain}, nil
}

func (s *twoFactorService) Disable(ctx context.Context, userID string) error {
	// Capture tenant before deletion so the audit event can attribute it.
	var tenantID string
	if existing, err := s.repo.GetByUserID(ctx, userID); err == nil && existing != nil {
		tenantID = existing.TenantID
	}
	if err := s.repo.Delete(ctx, userID); err != nil {
		return err
	}
	s.publish2FAEvent(ctx, "TwoFactorDisabled", map[string]interface{}{
		"tenant_id": tenantID,
		"user_id":   userID,
	})
	return nil
}

func (s *twoFactorService) IsEnabled(ctx context.Context, userID string) (bool, error) {
	row, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		// "not configured" is not an error from the caller's perspective.
		if strings.Contains(err.Error(), "not configured") {
			return false, nil
		}
		return false, err
	}
	return row.Enabled && row.Secret != "", nil
}

// Verify checks a TOTP code or backup code. Returns true on success.
func (s *twoFactorService) Verify(ctx context.Context, userID, code string) (bool, error) {
	row, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	if !row.Enabled || row.Secret == "" {
		return false, errors.New("2fa not enabled")
	}

	clean := strings.ReplaceAll(strings.TrimSpace(code), " ", "")

	// First try TOTP.
	if totp.Validate(clean, row.Secret) {
		_ = s.repo.UpdateLastUsed(ctx, userID)
		return true, nil
	}

	// Fall back to backup codes (case-insensitive, dashes stripped).
	normalizedBackup := strings.ToUpper(strings.ReplaceAll(clean, "-", ""))
	codes, err := s.repo.ListUnusedBackupCodes(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, bc := range codes {
		if bcrypt.CompareHashAndPassword([]byte(bc.CodeHash), []byte(normalizedBackup)) == nil {
			if err := s.repo.MarkBackupCodeUsed(ctx, bc.ID); err != nil {
				s.logger.WithError(err).Warn("Failed to mark backup code as used")
			}
			_ = s.repo.UpdateLastUsed(ctx, userID)
			return true, nil
		}
	}

	return false, nil
}

// generateBackupCodes returns (plaintextDisplay, hashedRows). Plaintext is shown
// to the user once; the database only holds bcrypt hashes.
func (s *twoFactorService) generateBackupCodes(userID string, n int) ([]string, []models.TwoFactorBackupCode, error) {
	plain := make([]string, 0, n)
	hashed := make([]models.TwoFactorBackupCode, 0, n)
	for i := 0; i < n; i++ {
		raw := make([]byte, 6) // 6 random bytes → ~10 chars in base32
		if _, err := s.codeRand(raw); err != nil {
			return nil, nil, err
		}
		code := strings.TrimRight(base32.StdEncoding.EncodeToString(raw), "=")
		// Display as 5-5 grouping for legibility, e.g. "ABCDE-FGHIJ".
		display := code
		if len(code) >= 10 {
			display = code[:5] + "-" + code[5:10]
		}
		plain = append(plain, display)

		// Store the un-grouped uppercase form so user input (with or without dash) matches.
		h, err := bcrypt.GenerateFromPassword([]byte(strings.ToUpper(code)), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, err
		}
		hashed = append(hashed, models.TwoFactorBackupCode{
			UserID:   userID,
			CodeHash: string(h),
		})
	}
	return plain, hashed, nil
}

// === Challenge token helpers used by the login flow ===

// twoFactorChallengeClaims is a short-lived JWT proving the user passed
// password auth and is awaiting a TOTP code.
type twoFactorChallengeClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Purpose  string `json:"purpose"`
	jwt.RegisteredClaims
}

const twoFactorChallengePurpose = "2fa-challenge"

func issueTwoFactorChallenge(secret string, userID, tenantID string, ttl time.Duration) (string, error) {
	claims := &twoFactorChallengeClaims{
		UserID:   userID,
		TenantID: tenantID,
		Purpose:  twoFactorChallengePurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(secret))
}

func parseTwoFactorChallenge(secret, tokenString string) (*twoFactorChallengeClaims, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &twoFactorChallengeClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, errors.New("invalid challenge token")
	}
	claims, ok := parsed.Claims.(*twoFactorChallengeClaims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid challenge token")
	}
	if claims.Purpose != twoFactorChallengePurpose {
		return nil, errors.New("invalid challenge token")
	}
	return claims, nil
}

// publish2FAEvent publishes a 2FA security event to user-events. When no
// publisher is configured (e.g. unit tests), it returns silently. Failures
// are logged but never propagated — auditing must not block 2FA setup.
func (s *twoFactorService) publish2FAEvent(ctx context.Context, eventType string, payload map[string]interface{}) {
	if s.kafkaProducer == nil {
		return
	}
	event := map[string]interface{}{
		"event_id":   uuid.New().String(),
		"event_type": eventType,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"version":    "1.0.0",
		"payload":    payload,
	}
	data, err := json.Marshal(event)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to marshal 2FA event")
		return
	}
	if err := s.kafkaProducer.Publish(ctx, "user-events", event["event_id"].(string), data); err != nil {
		s.logger.WithError(err).Warn("Failed to publish 2FA event")
	}
}
