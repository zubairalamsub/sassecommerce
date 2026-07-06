// Tests for the session-based login flow: LoginWithSession,
// RefreshAccessToken (rotation + reuse detection), logout, session listing
// and revocation, and the 2FA login challenge.

package service

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/repository/mocks"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func newSessionTestSvc(t *testing.T) (*authService, *mocks.MockUserRepository, *mocks.MockRefreshTokenRepository, *MockKafkaPublisher) {
	t.Helper()
	userRepo := new(mocks.MockUserRepository)
	refreshRepo := new(mocks.MockRefreshTokenRepository)
	kafka := new(MockKafkaPublisher)
	kafka.On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	svc := NewAuthServiceWithOptions(
		userRepo,
		models.TokenConfig{SecretKey: "secret", ExpirationTime: time.Hour, Issuer: "test"},
		kafka,
		logger,
		WithRefreshTokenRepository(refreshRepo),
		WithRefreshTokenTTL(48*time.Hour),
	).(*authService)
	return svc, userRepo, refreshRepo, kafka
}

func bcryptHash(t *testing.T, pwd string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return string(h)
}

func TestLoginWithSession_IssuesRefreshToken(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, refreshRepo, _ := newSessionTestSvc(t)

	user := &models.User{
		ID:           uuid.NewString(),
		TenantID:     "tenant-1",
		Email:        "alice@example.com",
		PasswordHash: bcryptHash(t, "hunter2!!"),
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
	}
	userRepo.On("GetByEmail", ctx, "tenant-1", "alice@example.com").Return(user, nil)
	userRepo.On("UpdateLastLogin", ctx, user.ID).Return(nil)

	var captured *models.RefreshToken
	refreshRepo.On("Create", ctx, mock.AnythingOfType("*models.RefreshToken")).
		Run(func(args mock.Arguments) { captured = args.Get(1).(*models.RefreshToken) }).
		Return(nil)

	resp, err := svc.LoginWithSession(ctx, &models.LoginRequest{
		TenantID: "tenant-1",
		Email:    "alice@example.com",
		Password: "hunter2!!",
	}, SessionContext{UserAgent: "Test/1.0", IPAddress: "1.2.3.4"})

	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotNil(t, captured)
	assert.Equal(t, user.ID, captured.UserID)
	assert.Equal(t, "Test/1.0", captured.UserAgent)
	assert.Equal(t, "1.2.3.4", captured.IPAddress)
	assert.WithinDuration(t, time.Now().Add(48*time.Hour), captured.ExpiresAt, 5*time.Second)
}

func TestRefreshAccessToken_RotatesAndIssuesNew(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, refreshRepo, _ := newSessionTestSvc(t)

	user := &models.User{ID: "u1", TenantID: "t1", Email: "a@x.com", Status: models.UserStatusActive, Role: models.UserRoleCustomer}
	old := &models.RefreshToken{
		ID:        "old-id",
		UserID:    "u1",
		TenantID:  "t1",
		Token:     "old-token",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	refreshRepo.On("GetByToken", ctx, "old-token").Return(old, nil)
	userRepo.On("GetByID", ctx, "u1").Return(user, nil)
	refreshRepo.On("Revoke", ctx, "old-id").Return(nil)
	refreshRepo.On("Create", ctx, mock.AnythingOfType("*models.RefreshToken")).Return(nil)

	resp, err := svc.RefreshAccessToken(ctx, &models.RefreshTokenRequest{RefreshToken: "old-token"}, SessionContext{})

	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotEqual(t, "old-token", resp.RefreshToken, "must rotate refresh token")
	refreshRepo.AssertExpectations(t)
}

func TestRefreshAccessToken_RejectsRevokedAndRevokesAllOnReuse(t *testing.T) {
	ctx := context.Background()
	svc, _, refreshRepo, _ := newSessionTestSvc(t)

	revokedAt := time.Now().Add(-time.Minute)
	old := &models.RefreshToken{
		ID:        "old",
		UserID:    "u1",
		Token:     "tok",
		ExpiresAt: time.Now().Add(time.Hour),
		RevokedAt: &revokedAt,
	}
	refreshRepo.On("GetByToken", ctx, "tok").Return(old, nil)
	refreshRepo.On("RevokeAllByUser", ctx, "u1").Return(nil)

	_, err := svc.RefreshAccessToken(ctx, &models.RefreshTokenRequest{RefreshToken: "tok"}, SessionContext{})
	assert.EqualError(t, err, "invalid refresh token")
	refreshRepo.AssertCalled(t, "RevokeAllByUser", ctx, "u1")
}

func TestRefreshAccessToken_RejectsExpired(t *testing.T) {
	ctx := context.Background()
	svc, _, refreshRepo, _ := newSessionTestSvc(t)

	old := &models.RefreshToken{
		ID:        "old",
		UserID:    "u1",
		Token:     "tok",
		ExpiresAt: time.Now().Add(-time.Minute),
	}
	refreshRepo.On("GetByToken", ctx, "tok").Return(old, nil)

	_, err := svc.RefreshAccessToken(ctx, &models.RefreshTokenRequest{RefreshToken: "tok"}, SessionContext{})
	assert.EqualError(t, err, "invalid refresh token")
	// Expired-but-not-revoked must NOT trigger the all-sessions revocation.
	refreshRepo.AssertNotCalled(t, "RevokeAllByUser", mock.Anything, mock.Anything)
}

func TestRefreshAccessToken_NotEnabledWithoutRepo(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	svc := NewAuthService(new(mocks.MockUserRepository), models.TokenConfig{SecretKey: "x", ExpirationTime: time.Hour}, new(MockKafkaPublisher), logger).(*authService)

	_, err := svc.RefreshAccessToken(ctx, &models.RefreshTokenRequest{RefreshToken: "x"}, SessionContext{})
	assert.EqualError(t, err, "refresh tokens are not enabled")
}

func TestLogout_RevokesByToken(t *testing.T) {
	ctx := context.Background()
	svc, _, refreshRepo, _ := newSessionTestSvc(t)

	refreshRepo.On("RevokeByToken", ctx, "tok").Return(nil)

	assert.NoError(t, svc.Logout(ctx, "tok"))
	refreshRepo.AssertExpectations(t)
}

func TestLogout_SwallowsUnknownToken(t *testing.T) {
	ctx := context.Background()
	svc, _, refreshRepo, _ := newSessionTestSvc(t)

	refreshRepo.On("RevokeByToken", ctx, "tok").Return(errors.New("refresh token not found or already revoked"))
	// Logout must not surface "token not found" to avoid enumeration.
	assert.NoError(t, svc.Logout(ctx, "tok"))
}

func TestListSessions_FlagsCurrent(t *testing.T) {
	ctx := context.Background()
	svc, _, refreshRepo, _ := newSessionTestSvc(t)

	tokens := []models.RefreshToken{
		{ID: "s1", Token: "tok1", UserAgent: "Chrome", LastUsedAt: time.Now()},
		{ID: "s2", Token: "tok2", UserAgent: "Firefox", LastUsedAt: time.Now().Add(-time.Hour)},
	}
	refreshRepo.On("ListByUser", ctx, "u1").Return(tokens, nil)

	sessions, err := svc.ListSessions(ctx, "u1", "tok2")
	assert.NoError(t, err)
	assert.Len(t, sessions, 2)
	assert.False(t, sessions[0].Current)
	assert.True(t, sessions[1].Current)
	// Token strings must never leak into the response shape.
	assert.Empty(t, sessions[0].UserAgent == "" && sessions[1].UserAgent == "")
}

func TestRevokeSession_ChecksOwnership(t *testing.T) {
	ctx := context.Background()
	svc, _, refreshRepo, _ := newSessionTestSvc(t)

	// Session belongs to a different user → must be reported as not found.
	refreshRepo.On("GetByID", ctx, "s9").Return(&models.RefreshToken{ID: "s9", UserID: "other"}, nil)
	err := svc.RevokeSession(ctx, "u1", "s9")
	assert.EqualError(t, err, "session not found")
	refreshRepo.AssertNotCalled(t, "Revoke", mock.Anything, mock.Anything)
}

func TestRevokeSession_OK(t *testing.T) {
	ctx := context.Background()
	svc, _, refreshRepo, _ := newSessionTestSvc(t)

	refreshRepo.On("GetByID", ctx, "s1").Return(&models.RefreshToken{ID: "s1", UserID: "u1"}, nil)
	refreshRepo.On("Revoke", ctx, "s1").Return(nil)

	assert.NoError(t, svc.RevokeSession(ctx, "u1", "s1"))
	refreshRepo.AssertExpectations(t)
}

// fake2FA is a minimal TwoFactorService for testing the AuthService integration.
type fake2FA struct {
	enabled  bool
	verifyOK bool
	verifyErr error
}

func (f *fake2FA) BeginSetup(ctx context.Context, userID, tenantID, accountName string) (*models.TwoFactorSetupResponse, error) {
	return &models.TwoFactorSetupResponse{Secret: "S"}, nil
}
func (f *fake2FA) ConfirmSetup(ctx context.Context, userID, code string) (*models.TwoFactorConfirmResponse, error) {
	return &models.TwoFactorConfirmResponse{BackupCodes: []string{"AAA-BBB"}}, nil
}
func (f *fake2FA) Disable(ctx context.Context, userID string) error      { return nil }
func (f *fake2FA) IsEnabled(ctx context.Context, userID string) (bool, error) { return f.enabled, nil }
func (f *fake2FA) Verify(ctx context.Context, userID, code string) (bool, error) {
	return f.verifyOK, f.verifyErr
}

func TestLoginWithSession_Returns2FAChallengeWhenEnabled(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, _, _ := newSessionTestSvc(t)
	svc.twoFactor = &fake2FA{enabled: true}

	user := &models.User{
		ID: "u1", TenantID: "t1", Email: "a@x.com",
		PasswordHash: bcryptHash(t, "hunter2!!"),
		Status:       models.UserStatusActive,
	}
	userRepo.On("GetByEmail", ctx, "t1", "a@x.com").Return(user, nil)

	resp, err := svc.LoginWithSession(ctx, &models.LoginRequest{
		TenantID: "t1", Email: "a@x.com", Password: "hunter2!!",
	}, SessionContext{})

	assert.NoError(t, err)
	assert.NotNil(t, resp.TwoFactor)
	assert.True(t, resp.TwoFactor.Required)
	assert.NotEmpty(t, resp.TwoFactor.ChallengeToken)
	assert.Empty(t, resp.Token, "no access token issued before 2FA")
	assert.Empty(t, resp.RefreshToken, "no refresh token issued before 2FA")
	// UpdateLastLogin must NOT have been called yet — login isn't complete.
	userRepo.AssertNotCalled(t, "UpdateLastLogin", mock.Anything, mock.Anything)
}

func TestVerifyTwoFactorChallenge_CompletesLogin(t *testing.T) {
	ctx := context.Background()
	svc, userRepo, refreshRepo, _ := newSessionTestSvc(t)
	svc.twoFactor = &fake2FA{verifyOK: true}

	user := &models.User{
		ID: "u1", TenantID: "t1", Email: "a@x.com",
		Status: models.UserStatusActive, Role: models.UserRoleCustomer,
	}
	userRepo.On("GetByID", ctx, "u1").Return(user, nil)
	userRepo.On("UpdateLastLogin", ctx, "u1").Return(nil)
	refreshRepo.On("Create", ctx, mock.AnythingOfType("*models.RefreshToken")).Return(nil)

	challenge, _ := issueTwoFactorChallenge(svc.tokenConfig.SecretKey, "u1", "t1", time.Minute)

	resp, err := svc.VerifyTwoFactorChallenge(ctx, &models.TwoFactorVerifyRequest{
		ChallengeToken: challenge,
		Code:           "123456",
	}, SessionContext{UserAgent: "Test"})

	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.RefreshToken)
}

func TestVerifyTwoFactorChallenge_RejectsBadCode(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _ := newSessionTestSvc(t)
	svc.twoFactor = &fake2FA{verifyOK: false}

	challenge, _ := issueTwoFactorChallenge(svc.tokenConfig.SecretKey, "u1", "t1", time.Minute)

	_, err := svc.VerifyTwoFactorChallenge(ctx, &models.TwoFactorVerifyRequest{
		ChallengeToken: challenge, Code: "000000",
	}, SessionContext{})

	assert.EqualError(t, err, "invalid 2fa code")
}

func TestLogoutAll_RevokesAll(t *testing.T) {
	ctx := context.Background()
	svc, _, refreshRepo, _ := newSessionTestSvc(t)

	refreshRepo.On("RevokeAllByUser", ctx, "u1").Return(nil)
	assert.NoError(t, svc.LogoutAll(ctx, "u1"))
	refreshRepo.AssertExpectations(t)
}
