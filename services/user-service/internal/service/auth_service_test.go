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
	"github.com/stretchr/testify/suite"
)

type AuthServiceTestSuite struct {
	suite.Suite
	mockRepo      *mocks.MockUserRepository
	mockTokenRepo *mocks.MockTokenRepository
	mockKafka     *MockKafkaPublisher
	service       AuthService
	tokenConfig   models.TokenConfig
	logger        *logrus.Logger
}

func (suite *AuthServiceTestSuite) SetupTest() {
	suite.mockRepo = new(mocks.MockUserRepository)
	suite.mockTokenRepo = new(mocks.MockTokenRepository)
	suite.mockKafka = new(MockKafkaPublisher)
	suite.logger = logrus.New()
	suite.logger.SetOutput(io.Discard) // Disable logging during tests

	suite.tokenConfig = models.TokenConfig{
		SecretKey:      "test-secret-key",
		ExpirationTime: 24 * time.Hour,
		Issuer:         "test-service",
	}

	// Allow any Kafka publish calls (non-blocking in auth service)
	suite.mockKafka.On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	suite.service = NewAuthService(suite.mockRepo, suite.tokenConfig, suite.mockKafka, suite.logger, suite.mockTokenRepo)
}

func (suite *AuthServiceTestSuite) TestRegister_Success() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	req := &models.RegisterRequest{
		TenantID:  tenantID,
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	suite.mockRepo.On("EmailExists", ctx, tenantID, req.Email).Return(false, nil)
	suite.mockRepo.On("UsernameExists", ctx, tenantID, req.Username).Return(false, nil)
	suite.mockRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).Return(nil)
	// Registration triggers email verification
	suite.mockTokenRepo.On("InvalidateVerificationTokens", ctx, mock.AnythingOfType("string")).Return(nil).Maybe()
	suite.mockTokenRepo.On("CreateVerificationToken", ctx, mock.AnythingOfType("*models.VerificationToken")).Return(nil).Maybe()

	result, err := suite.service.Register(ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), req.Email, result.Email)
	assert.Equal(suite.T(), req.Username, result.Username)
	assert.Equal(suite.T(), req.FirstName, result.FirstName)
	assert.Equal(suite.T(), req.LastName, result.LastName)
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestRegister_EmailExists() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	req := &models.RegisterRequest{
		TenantID:  tenantID,
		Email:     "existing@example.com",
		Username:  "testuser",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	suite.mockRepo.On("EmailExists", ctx, tenantID, req.Email).Return(true, nil)

	result, err := suite.service.Register(ctx, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), "email already exists", err.Error())
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestRegister_UsernameExists() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	req := &models.RegisterRequest{
		TenantID:  tenantID,
		Email:     "test@example.com",
		Username:  "existinguser",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	suite.mockRepo.On("EmailExists", ctx, tenantID, req.Email).Return(false, nil)
	suite.mockRepo.On("UsernameExists", ctx, tenantID, req.Username).Return(true, nil)

	result, err := suite.service.Register(ctx, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), "username already exists", err.Error())
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestLogin_Success() {
	ctx := context.Background()
	tenantID := uuid.New().String()
	userID := uuid.New().String()

	// Create password hash
	passwordHash, _ := hashPassword("password123")

	user := &models.User{
		ID:           userID,
		TenantID:     tenantID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: passwordHash,
		FirstName:    "Test",
		LastName:     "User",
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
	}

	req := &models.LoginRequest{
		TenantID: tenantID,
		Email:    "test@example.com",
		Password: "password123",
	}

	suite.mockRepo.On("GetByEmail", ctx, tenantID, req.Email).Return(user, nil)
	suite.mockRepo.On("UpdateLastLogin", ctx, userID).Return(nil)

	result, err := suite.service.Login(ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.NotEmpty(suite.T(), result.Token)
	assert.Equal(suite.T(), user.Email, result.User.Email)
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestLogin_InvalidEmail() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	req := &models.LoginRequest{
		TenantID: tenantID,
		Email:    "nonexistent@example.com",
		Password: "password123",
	}

	suite.mockRepo.On("GetByEmail", ctx, tenantID, req.Email).Return(nil, errors.New("user not found"))

	result, err := suite.service.Login(ctx, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), "invalid email or password", err.Error())
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestLogin_InvalidPassword() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	passwordHash, _ := hashPassword("correctpassword")

	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
	}

	req := &models.LoginRequest{
		TenantID: tenantID,
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	suite.mockRepo.On("GetByEmail", ctx, tenantID, req.Email).Return(user, nil)

	result, err := suite.service.Login(ctx, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), "invalid email or password", err.Error())
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestLogin_InactiveUser() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	passwordHash, _ := hashPassword("password123")

	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Status:       models.UserStatusInactive,
		Role:         models.UserRoleCustomer,
	}

	req := &models.LoginRequest{
		TenantID: tenantID,
		Email:    "test@example.com",
		Password: "password123",
	}

	suite.mockRepo.On("GetByEmail", ctx, tenantID, req.Email).Return(user, nil)

	result, err := suite.service.Login(ctx, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), "user account is not active", err.Error())
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestVerifyToken_Success() {
	ctx := context.Background()
	tenantID := uuid.New().String()
	userID := uuid.New().String()

	user := &models.User{
		ID:       userID,
		TenantID: tenantID,
		Email:    "test@example.com",
		Role:     models.UserRoleCustomer,
	}

	token, _, err := suite.service.(*authService).generateToken(user)
	assert.NoError(suite.T(), err)

	claims, err := suite.service.VerifyToken(ctx, token)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), claims)
	assert.Equal(suite.T(), userID, claims.UserID)
	assert.Equal(suite.T(), tenantID, claims.TenantID)
	assert.Equal(suite.T(), user.Email, claims.Email)
	assert.Equal(suite.T(), user.Role, claims.Role)
}

func (suite *AuthServiceTestSuite) TestVerifyToken_Invalid() {
	ctx := context.Background()

	claims, err := suite.service.VerifyToken(ctx, "invalid-token")

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), claims)
}

func (suite *AuthServiceTestSuite) TestChangePassword_Success() {
	ctx := context.Background()
	userID := uuid.New().String()

	oldPasswordHash, _ := hashPassword("oldpassword123")

	user := &models.User{
		ID:           userID,
		PasswordHash: oldPasswordHash,
	}

	req := &models.ChangePasswordRequest{
		OldPassword: "oldpassword123",
		NewPassword: "newpassword123",
	}

	suite.mockRepo.On("GetByID", ctx, userID).Return(user, nil)
	suite.mockRepo.On("Update", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	err := suite.service.ChangePassword(ctx, userID, req)

	assert.NoError(suite.T(), err)
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestChangePassword_IncorrectOldPassword() {
	ctx := context.Background()
	userID := uuid.New().String()

	oldPasswordHash, _ := hashPassword("oldpassword123")

	user := &models.User{
		ID:           userID,
		PasswordHash: oldPasswordHash,
	}

	req := &models.ChangePasswordRequest{
		OldPassword: "wrongpassword",
		NewPassword: "newpassword123",
	}

	suite.mockRepo.On("GetByID", ctx, userID).Return(user, nil)

	err := suite.service.ChangePassword(ctx, userID, req)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "incorrect old password", err.Error())
	suite.mockRepo.AssertExpectations(suite.T())
}

// --- Email Verification Tests ---

func (suite *AuthServiceTestSuite) TestRequestEmailVerification_Success() {
	ctx := context.Background()
	userID := uuid.New().String()
	tenantID := uuid.New().String()
	email := "test@example.com"

	suite.mockTokenRepo.On("InvalidateVerificationTokens", ctx, userID).Return(nil)
	suite.mockTokenRepo.On("CreateVerificationToken", ctx, mock.AnythingOfType("*models.VerificationToken")).Return(nil)

	token, err := suite.service.RequestEmailVerification(ctx, userID, tenantID, email)

	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), token)
	assert.Len(suite.T(), token, 64) // 32 bytes = 64 hex chars
	suite.mockTokenRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestVerifyEmail_Success() {
	ctx := context.Background()
	tokenID := uuid.New().String()
	userID := uuid.New().String()
	tenantID := uuid.New().String()
	tokenStr := "valid-token-string"

	vt := &models.VerificationToken{
		ID:        tokenID,
		UserID:    userID,
		TenantID:  tenantID,
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	suite.mockTokenRepo.On("GetVerificationTokenByToken", ctx, tokenStr).Return(vt, nil)
	suite.mockRepo.On("SetEmailVerified", ctx, userID).Return(nil)
	suite.mockTokenRepo.On("MarkVerificationTokenUsed", ctx, tokenID).Return(nil)

	err := suite.service.VerifyEmail(ctx, &models.VerifyEmailRequest{Token: tokenStr})

	assert.NoError(suite.T(), err)
	suite.mockTokenRepo.AssertExpectations(suite.T())
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestVerifyEmail_InvalidToken() {
	ctx := context.Background()
	tokenStr := "invalid-token"

	suite.mockTokenRepo.On("GetVerificationTokenByToken", ctx, tokenStr).Return(nil, errors.New("verification token not found"))

	err := suite.service.VerifyEmail(ctx, &models.VerifyEmailRequest{Token: tokenStr})

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "invalid or expired verification token", err.Error())
}

func (suite *AuthServiceTestSuite) TestVerifyEmail_ExpiredToken() {
	ctx := context.Background()
	tokenStr := "expired-token"

	vt := &models.VerificationToken{
		ID:        uuid.New().String(),
		UserID:    uuid.New().String(),
		TenantID:  uuid.New().String(),
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
	}

	suite.mockTokenRepo.On("GetVerificationTokenByToken", ctx, tokenStr).Return(vt, nil)

	err := suite.service.VerifyEmail(ctx, &models.VerifyEmailRequest{Token: tokenStr})

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "invalid or expired verification token", err.Error())
}

func (suite *AuthServiceTestSuite) TestResendEmailVerification_Success() {
	ctx := context.Background()
	tenantID := uuid.New().String()
	userID := uuid.New().String()

	user := &models.User{
		ID:            userID,
		TenantID:      tenantID,
		Email:         "test@example.com",
		EmailVerified: false,
		Status:        models.UserStatusActive,
	}

	suite.mockRepo.On("GetByEmail", ctx, tenantID, user.Email).Return(user, nil)
	suite.mockTokenRepo.On("InvalidateVerificationTokens", ctx, userID).Return(nil)
	suite.mockTokenRepo.On("CreateVerificationToken", ctx, mock.AnythingOfType("*models.VerificationToken")).Return(nil)

	req := &models.ResendVerificationRequest{
		TenantID: tenantID,
		Email:    user.Email,
	}

	err := suite.service.ResendEmailVerification(ctx, req)

	assert.NoError(suite.T(), err)
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestResendEmailVerification_AlreadyVerified() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	user := &models.User{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		Email:         "verified@example.com",
		EmailVerified: true,
		Status:        models.UserStatusActive,
	}

	suite.mockRepo.On("GetByEmail", ctx, tenantID, user.Email).Return(user, nil)

	req := &models.ResendVerificationRequest{
		TenantID: tenantID,
		Email:    user.Email,
	}

	err := suite.service.ResendEmailVerification(ctx, req)

	assert.NoError(suite.T(), err) // No error, just silently succeeds
}

// --- Password Reset Tests ---

func (suite *AuthServiceTestSuite) TestRequestPasswordReset_Success() {
	ctx := context.Background()
	tenantID := uuid.New().String()
	userID := uuid.New().String()

	user := &models.User{
		ID:        userID,
		TenantID:  tenantID,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Status:    models.UserStatusActive,
	}

	suite.mockRepo.On("GetByEmail", ctx, tenantID, user.Email).Return(user, nil)
	suite.mockTokenRepo.On("InvalidatePasswordResetTokens", ctx, userID).Return(nil)
	suite.mockTokenRepo.On("CreatePasswordResetToken", ctx, mock.AnythingOfType("*models.PasswordResetToken")).Return(nil)

	req := &models.ForgotPasswordRequest{
		TenantID: tenantID,
		Email:    user.Email,
	}

	err := suite.service.RequestPasswordReset(ctx, req)

	assert.NoError(suite.T(), err)
	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockTokenRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestRequestPasswordReset_NonExistentEmail() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	suite.mockRepo.On("GetByEmail", ctx, tenantID, "nobody@example.com").Return(nil, errors.New("user not found"))

	req := &models.ForgotPasswordRequest{
		TenantID: tenantID,
		Email:    "nobody@example.com",
	}

	err := suite.service.RequestPasswordReset(ctx, req)

	// Should NOT return error (prevents email enumeration)
	assert.NoError(suite.T(), err)
}

func (suite *AuthServiceTestSuite) TestResetPassword_Success() {
	ctx := context.Background()
	tokenID := uuid.New().String()
	userID := uuid.New().String()
	tenantID := uuid.New().String()
	tokenStr := "valid-reset-token"

	prt := &models.PasswordResetToken{
		ID:        tokenID,
		UserID:    userID,
		TenantID:  tenantID,
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	suite.mockTokenRepo.On("GetPasswordResetTokenByToken", ctx, tokenStr).Return(prt, nil)
	suite.mockRepo.On("UpdatePassword", ctx, userID, mock.AnythingOfType("string")).Return(nil)
	suite.mockTokenRepo.On("MarkPasswordResetTokenUsed", ctx, tokenID).Return(nil)
	suite.mockTokenRepo.On("InvalidatePasswordResetTokens", ctx, userID).Return(nil)

	req := &models.ResetPasswordRequest{
		Token:       tokenStr,
		NewPassword: "newpassword123",
	}

	err := suite.service.ResetPassword(ctx, req)

	assert.NoError(suite.T(), err)
	suite.mockTokenRepo.AssertExpectations(suite.T())
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestResetPassword_InvalidToken() {
	ctx := context.Background()
	tokenStr := "invalid-reset-token"

	suite.mockTokenRepo.On("GetPasswordResetTokenByToken", ctx, tokenStr).Return(nil, errors.New("password reset token not found"))

	req := &models.ResetPasswordRequest{
		Token:       tokenStr,
		NewPassword: "newpassword123",
	}

	err := suite.service.ResetPassword(ctx, req)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "invalid or expired reset token", err.Error())
}

func (suite *AuthServiceTestSuite) TestResetPassword_ExpiredToken() {
	ctx := context.Background()
	tokenStr := "expired-reset-token"

	prt := &models.PasswordResetToken{
		ID:        uuid.New().String(),
		UserID:    uuid.New().String(),
		TenantID:  uuid.New().String(),
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
	}

	suite.mockTokenRepo.On("GetPasswordResetTokenByToken", ctx, tokenStr).Return(prt, nil)

	req := &models.ResetPasswordRequest{
		Token:       tokenStr,
		NewPassword: "newpassword123",
	}

	err := suite.service.ResetPassword(ctx, req)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "invalid or expired reset token", err.Error())
}

func TestAuthServiceTestSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceTestSuite))
}
