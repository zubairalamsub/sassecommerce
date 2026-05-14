package service

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/repository/mocks"
	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func newTwoFactorTestSvc(t *testing.T) (*twoFactorService, *mocks.MockTwoFactorRepository) {
	t.Helper()
	repo := new(mocks.MockTwoFactorRepository)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	svc := NewTwoFactorService(repo, logger).(*twoFactorService)
	return svc, repo
}

func TestBeginSetup_StoresPendingSecret(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTwoFactorTestSvc(t)

	repo.On("GetByUserID", ctx, "u1").Return(nil, assertErr("not configured")).Once()

	var captured *models.TwoFactorSecret
	repo.On("Upsert", ctx, mock.AnythingOfType("*models.TwoFactorSecret")).
		Run(func(args mock.Arguments) { captured = args.Get(1).(*models.TwoFactorSecret) }).
		Return(nil)

	resp, err := svc.BeginSetup(ctx, "u1", "t1", "alice@example.com")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Secret)
	assert.True(t, strings.HasPrefix(resp.ProvisioningURI, "otpauth://"))
	assert.True(t, strings.HasPrefix(resp.QRCodeImage, "data:image/png;base64,"))
	assert.NotNil(t, captured)
	assert.Equal(t, captured.Pending, resp.Secret)
	assert.False(t, captured.Enabled)
}

func TestBeginSetup_BlocksWhenAlreadyEnabled(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTwoFactorTestSvc(t)

	repo.On("GetByUserID", ctx, "u1").Return(&models.TwoFactorSecret{Enabled: true}, nil)

	_, err := svc.BeginSetup(ctx, "u1", "t1", "alice@example.com")
	assert.ErrorContains(t, err, "already enabled")
}

func TestConfirmSetup_PromotesPendingAndReturnsBackupCodes(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTwoFactorTestSvc(t)

	pendingSecret := "JBSWY3DPEHPK3PXP" // standard test base32 secret
	repo.On("GetByUserID", ctx, "u1").Return(&models.TwoFactorSecret{
		UserID: "u1", Pending: pendingSecret,
	}, nil)
	repo.On("Upsert", ctx, mock.MatchedBy(func(s *models.TwoFactorSecret) bool {
		return s.Enabled && s.Secret == pendingSecret && s.Pending == "" && s.ConfirmedAt != nil
	})).Return(nil)
	repo.On("DeleteBackupCodes", ctx, "u1").Return(nil)
	repo.On("CreateBackupCodes", ctx, mock.AnythingOfType("[]models.TwoFactorBackupCode")).Return(nil)

	code, err := totp.GenerateCode(pendingSecret, time.Now())
	assert.NoError(t, err)

	resp, err := svc.ConfirmSetup(ctx, "u1", code)
	assert.NoError(t, err)
	assert.Len(t, resp.BackupCodes, 10)
	for _, c := range resp.BackupCodes {
		assert.Contains(t, c, "-")
	}
}

func TestConfirmSetup_RejectsInvalidCode(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTwoFactorTestSvc(t)
	repo.On("GetByUserID", ctx, "u1").Return(&models.TwoFactorSecret{Pending: "JBSWY3DPEHPK3PXP"}, nil)

	_, err := svc.ConfirmSetup(ctx, "u1", "000000")
	assert.EqualError(t, err, "invalid code")
}

func TestVerify_TOTP_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTwoFactorTestSvc(t)

	secret := "JBSWY3DPEHPK3PXP"
	repo.On("GetByUserID", ctx, "u1").Return(&models.TwoFactorSecret{
		UserID: "u1", Secret: secret, Enabled: true,
	}, nil)
	repo.On("UpdateLastUsed", ctx, "u1").Return(nil)

	code, _ := totp.GenerateCode(secret, time.Now())

	ok, err := svc.Verify(ctx, "u1", code)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestVerify_BackupCode_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTwoFactorTestSvc(t)

	repo.On("GetByUserID", ctx, "u1").Return(&models.TwoFactorSecret{
		UserID: "u1", Secret: "JBSWY3DPEHPK3PXP", Enabled: true,
	}, nil)

	hash, _ := bcrypt.GenerateFromPassword([]byte("ABCDEFGHIJ"), bcrypt.MinCost)
	repo.On("ListUnusedBackupCodes", ctx, "u1").Return([]models.TwoFactorBackupCode{
		{ID: "bc1", UserID: "u1", CodeHash: string(hash)},
	}, nil)
	repo.On("MarkBackupCodeUsed", ctx, "bc1").Return(nil)
	repo.On("UpdateLastUsed", ctx, "u1").Return(nil)

	// User typed it with the dash; uppercased input must still match.
	ok, err := svc.Verify(ctx, "u1", "abcde-fghij")
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestVerify_RejectsWrongCode(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTwoFactorTestSvc(t)

	repo.On("GetByUserID", ctx, "u1").Return(&models.TwoFactorSecret{
		UserID: "u1", Secret: "JBSWY3DPEHPK3PXP", Enabled: true,
	}, nil)
	repo.On("ListUnusedBackupCodes", ctx, "u1").Return([]models.TwoFactorBackupCode{}, nil)

	ok, err := svc.Verify(ctx, "u1", "999999")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestVerify_RejectsWhenDisabled(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTwoFactorTestSvc(t)
	repo.On("GetByUserID", ctx, "u1").Return(&models.TwoFactorSecret{Enabled: false}, nil)

	_, err := svc.Verify(ctx, "u1", "123456")
	assert.EqualError(t, err, "2fa not enabled")
}

func TestIsEnabled_NotConfiguredReturnsFalseNoError(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTwoFactorTestSvc(t)
	repo.On("GetByUserID", ctx, "u1").Return(nil, assertErr("2fa not configured"))

	enabled, err := svc.IsEnabled(ctx, "u1")
	assert.NoError(t, err)
	assert.False(t, enabled)
}

func TestChallengeToken_Roundtrip(t *testing.T) {
	tok, err := issueTwoFactorChallenge("test-secret", "u1", "t1", time.Minute)
	assert.NoError(t, err)
	claims, err := parseTwoFactorChallenge("test-secret", tok)
	assert.NoError(t, err)
	assert.Equal(t, "u1", claims.UserID)
	assert.Equal(t, "t1", claims.TenantID)
}

func TestChallengeToken_RejectsWrongSecret(t *testing.T) {
	tok, _ := issueTwoFactorChallenge("test-secret", "u1", "t1", time.Minute)
	_, err := parseTwoFactorChallenge("other-secret", tok)
	assert.Error(t, err)
}

func TestChallengeToken_RejectsExpired(t *testing.T) {
	tok, _ := issueTwoFactorChallenge("test-secret", "u1", "t1", -time.Minute)
	_, err := parseTwoFactorChallenge("test-secret", tok)
	assert.Error(t, err)
}

// assertErr is a tiny helper — we want to control the error message so the
// IsEnabled "not configured" branch is exercised.
type assertableErr struct{ msg string }

func (e assertableErr) Error() string { return e.msg }
func assertErr(msg string) error      { return assertableErr{msg} }
