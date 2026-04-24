package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// KafkaPublisher defines the interface for publishing messages to Kafka
type KafkaPublisher interface {
	Publish(ctx context.Context, topic, key string, value []byte) error
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	Register(ctx context.Context, req *models.RegisterRequest) (*models.UserResponse, error)
	Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error)
	VerifyToken(ctx context.Context, tokenString string) (*models.TokenClaims, error)
	ChangePassword(ctx context.Context, userID string, req *models.ChangePasswordRequest) error
	RequestEmailVerification(ctx context.Context, userID, tenantID, email string) (string, error)
	ResendEmailVerification(ctx context.Context, req *models.ResendVerificationRequest) error
	VerifyEmail(ctx context.Context, req *models.VerifyEmailRequest) error
	RequestPasswordReset(ctx context.Context, req *models.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req *models.ResetPasswordRequest) error
}

type authService struct {
	userRepo      repository.UserRepository
	tokenRepo     repository.TokenRepository
	tokenConfig   models.TokenConfig
	kafkaProducer KafkaPublisher
	logger        *logrus.Logger
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo repository.UserRepository,
	tokenConfig models.TokenConfig,
	kafkaProducer KafkaPublisher,
	logger *logrus.Logger,
	tokenRepo ...repository.TokenRepository,
) AuthService {
	s := &authService{
		userRepo:      userRepo,
		tokenConfig:   tokenConfig,
		kafkaProducer: kafkaProducer,
		logger:        logger,
	}
	if len(tokenRepo) > 0 {
		s.tokenRepo = tokenRepo[0]
	}
	return s
}

// Register registers a new user
func (s *authService) Register(ctx context.Context, req *models.RegisterRequest) (*models.UserResponse, error) {
	// Check if email already exists
	emailExists, err := s.userRepo.EmailExists(ctx, req.TenantID, req.Email)
	if err != nil {
		s.logger.WithError(err).Error("Failed to check email existence")
		return nil, errors.New("failed to check email availability")
	}
	if emailExists {
		return nil, errors.New("email already exists")
	}

	// Check if username already exists
	usernameExists, err := s.userRepo.UsernameExists(ctx, req.TenantID, req.Username)
	if err != nil {
		s.logger.WithError(err).Error("Failed to check username existence")
		return nil, errors.New("failed to check username availability")
	}
	if usernameExists {
		return nil, errors.New("username already exists")
	}

	// Hash password
	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		s.logger.WithError(err).Error("Failed to hash password")
		return nil, errors.New("failed to process password")
	}

	// Create user
	user := &models.User{
		TenantID:     req.TenantID,
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: passwordHash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Phone:        req.Phone,
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
		EmailVerified: false,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.WithError(err).Error("Failed to create user")
		return nil, errors.New("failed to create user")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":   user.ID,
		"tenant_id": user.TenantID,
		"email":     user.Email,
	}).Info("User registered successfully")

	// Publish UserRegistered event
	s.publishEvent(ctx, "UserRegistered", map[string]interface{}{
		"tenant_id": user.TenantID,
		"user_id":   user.ID,
		"email":     user.Email,
		"name":      user.FirstName + " " + user.LastName,
		"role":      user.Role,
	})

	// Send verification email if token repo is available
	if s.tokenRepo != nil {
		if _, err := s.RequestEmailVerification(ctx, user.ID, user.TenantID, user.Email); err != nil {
			s.logger.WithError(err).Warn("Failed to send verification email on registration")
		}
	}

	return user.ToResponse(), nil
}

// Login authenticates a user and returns a JWT token
func (s *authService) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.TenantID, req.Email)
	if err != nil {
		s.logger.WithError(err).WithField("email", req.Email).Warn("Login attempt with non-existent email")
		return nil, errors.New("invalid email or password")
	}

	// Check if user is active
	if user.Status != models.UserStatusActive {
		s.logger.WithField("user_id", user.ID).Warn("Login attempt for inactive user")
		return nil, errors.New("user account is not active")
	}

	// Verify password
	if !verifyPassword(user.PasswordHash, req.Password) {
		s.logger.WithField("user_id", user.ID).Warn("Login attempt with incorrect password")
		return nil, errors.New("invalid email or password")
	}

	// Generate JWT token
	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate token")
		return nil, errors.New("failed to generate authentication token")
	}

	// Update last login timestamp
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.WithError(err).Warn("Failed to update last login timestamp")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":   user.ID,
		"tenant_id": user.TenantID,
		"email":     user.Email,
	}).Info("User logged in successfully")

	return &models.LoginResponse{
		User:      user.ToResponse(),
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// VerifyToken verifies a JWT token and returns the claims
func (s *authService) VerifyToken(ctx context.Context, tokenString string) (*models.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.tokenConfig.SecretKey), nil
	})

	if err != nil {
		s.logger.WithError(err).Warn("Failed to parse token")
		return nil, errors.New("invalid token")
	}

	if claims, ok := token.Claims.(*models.TokenClaims); ok && token.Valid {
		// Check if token is expired
		if time.Now().After(claims.ExpiresAt.Time) {
			return nil, errors.New("token has expired")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// ChangePassword changes a user's password
func (s *authService) ChangePassword(ctx context.Context, userID string, req *models.ChangePasswordRequest) error {
	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get user")
		return errors.New("user not found")
	}

	// Verify old password
	if !verifyPassword(user.PasswordHash, req.OldPassword) {
		s.logger.WithField("user_id", userID).Warn("Password change attempt with incorrect old password")
		return errors.New("incorrect old password")
	}

	// Hash new password
	newPasswordHash, err := hashPassword(req.NewPassword)
	if err != nil {
		s.logger.WithError(err).Error("Failed to hash new password")
		return errors.New("failed to process new password")
	}

	// Update password
	user.PasswordHash = newPasswordHash
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.WithError(err).Error("Failed to update password")
		return errors.New("failed to update password")
	}

	s.logger.WithField("user_id", userID).Info("Password changed successfully")

	return nil
}

// RequestEmailVerification generates a verification token and publishes an event
func (s *authService) RequestEmailVerification(ctx context.Context, userID, tenantID, email string) (string, error) {
	// Invalidate any existing verification tokens for this user
	if err := s.tokenRepo.InvalidateVerificationTokens(ctx, userID); err != nil {
		s.logger.WithError(err).Warn("Failed to invalidate existing verification tokens")
	}

	// Generate a secure random token
	tokenStr, err := generateSecureToken()
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate verification token")
		return "", errors.New("failed to generate verification token")
	}

	// Create verification token (valid for 24 hours)
	vt := &models.VerificationToken{
		UserID:    userID,
		TenantID:  tenantID,
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.tokenRepo.CreateVerificationToken(ctx, vt); err != nil {
		s.logger.WithError(err).Error("Failed to save verification token")
		return "", errors.New("failed to create verification token")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"tenant_id": tenantID,
	}).Info("Email verification token created")

	// Publish event so notification service can send the email
	s.publishEvent(ctx, "EmailVerificationRequested", map[string]interface{}{
		"tenant_id": tenantID,
		"user_id":   userID,
		"email":     email,
		"token":     tokenStr,
	})

	return tokenStr, nil
}

// VerifyEmail verifies a user's email using the provided token
func (s *authService) VerifyEmail(ctx context.Context, req *models.VerifyEmailRequest) error {
	// Look up the token
	vt, err := s.tokenRepo.GetVerificationTokenByToken(ctx, req.Token)
	if err != nil {
		s.logger.WithError(err).Warn("Invalid verification token attempted")
		return errors.New("invalid or expired verification token")
	}

	// Check expiration
	if !vt.IsValid() {
		return errors.New("invalid or expired verification token")
	}

	// Mark the user's email as verified
	if err := s.userRepo.SetEmailVerified(ctx, vt.UserID); err != nil {
		s.logger.WithError(err).Error("Failed to set email as verified")
		return errors.New("failed to verify email")
	}

	// Mark the token as used
	if err := s.tokenRepo.MarkVerificationTokenUsed(ctx, vt.ID); err != nil {
		s.logger.WithError(err).Warn("Failed to mark verification token as used")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":   vt.UserID,
		"tenant_id": vt.TenantID,
	}).Info("Email verified successfully")

	// Publish event
	s.publishEvent(ctx, "EmailVerified", map[string]interface{}{
		"tenant_id": vt.TenantID,
		"user_id":   vt.UserID,
	})

	return nil
}

// ResendEmailVerification looks up a user by email and sends a new verification token
func (s *authService) ResendEmailVerification(ctx context.Context, req *models.ResendVerificationRequest) error {
	// Look up user — return nil even if not found to prevent email enumeration
	user, err := s.userRepo.GetByEmail(ctx, req.TenantID, req.Email)
	if err != nil {
		s.logger.WithField("email", req.Email).Debug("Resend verification requested for non-existent email")
		return nil
	}

	if user.EmailVerified {
		s.logger.WithField("user_id", user.ID).Debug("Resend verification requested for already verified email")
		return nil
	}

	if _, err := s.RequestEmailVerification(ctx, user.ID, user.TenantID, user.Email); err != nil {
		s.logger.WithError(err).Error("Failed to resend verification email")
		return errors.New("failed to resend verification email")
	}

	return nil
}

// RequestPasswordReset generates a password reset token and publishes an event
func (s *authService) RequestPasswordReset(ctx context.Context, req *models.ForgotPasswordRequest) error {
	// Look up user — return success even if not found to prevent email enumeration
	user, err := s.userRepo.GetByEmail(ctx, req.TenantID, req.Email)
	if err != nil {
		s.logger.WithField("email", req.Email).Debug("Password reset requested for non-existent email")
		return nil
	}

	if user.Status != models.UserStatusActive {
		s.logger.WithField("user_id", user.ID).Debug("Password reset requested for inactive user")
		return nil
	}

	// Invalidate any existing reset tokens
	if err := s.tokenRepo.InvalidatePasswordResetTokens(ctx, user.ID); err != nil {
		s.logger.WithError(err).Warn("Failed to invalidate existing reset tokens")
	}

	// Generate a secure random token
	tokenStr, err := generateSecureToken()
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate reset token")
		return errors.New("failed to process password reset request")
	}

	// Create reset token (valid for 1 hour)
	prt := &models.PasswordResetToken{
		UserID:    user.ID,
		TenantID:  req.TenantID,
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.tokenRepo.CreatePasswordResetToken(ctx, prt); err != nil {
		s.logger.WithError(err).Error("Failed to save password reset token")
		return errors.New("failed to process password reset request")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":   user.ID,
		"tenant_id": req.TenantID,
	}).Info("Password reset token created")

	// Publish event so notification service can send the email
	s.publishEvent(ctx, "PasswordResetRequested", map[string]interface{}{
		"tenant_id": req.TenantID,
		"user_id":   user.ID,
		"email":     user.Email,
		"name":      user.FirstName + " " + user.LastName,
		"token":     tokenStr,
	})

	return nil
}

// ResetPassword resets a user's password using the provided token
func (s *authService) ResetPassword(ctx context.Context, req *models.ResetPasswordRequest) error {
	// Look up the token
	prt, err := s.tokenRepo.GetPasswordResetTokenByToken(ctx, req.Token)
	if err != nil {
		s.logger.WithError(err).Warn("Invalid password reset token attempted")
		return errors.New("invalid or expired reset token")
	}

	// Check expiration
	if !prt.IsValid() {
		return errors.New("invalid or expired reset token")
	}

	// Hash the new password
	newPasswordHash, err := hashPassword(req.NewPassword)
	if err != nil {
		s.logger.WithError(err).Error("Failed to hash new password")
		return errors.New("failed to process new password")
	}

	// Update the password
	if err := s.userRepo.UpdatePassword(ctx, prt.UserID, newPasswordHash); err != nil {
		s.logger.WithError(err).Error("Failed to update password")
		return errors.New("failed to reset password")
	}

	// Mark the token as used
	if err := s.tokenRepo.MarkPasswordResetTokenUsed(ctx, prt.ID); err != nil {
		s.logger.WithError(err).Warn("Failed to mark reset token as used")
	}

	// Invalidate all other reset tokens for this user
	if err := s.tokenRepo.InvalidatePasswordResetTokens(ctx, prt.UserID); err != nil {
		s.logger.WithError(err).Warn("Failed to invalidate remaining reset tokens")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":   prt.UserID,
		"tenant_id": prt.TenantID,
	}).Info("Password reset successfully")

	// Publish event
	s.publishEvent(ctx, "PasswordReset", map[string]interface{}{
		"tenant_id": prt.TenantID,
		"user_id":   prt.UserID,
	})

	return nil
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateToken generates a JWT token for a user
func (s *authService) generateToken(user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.tokenConfig.ExpirationTime)

	claims := &models.TokenClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.tokenConfig.Issuer,
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.tokenConfig.SecretKey))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// hashPassword hashes a password using bcrypt
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// verifyPassword verifies a password against a hash
func verifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// publishEvent publishes an event to Kafka (non-blocking, logs warning on failure)
func (s *authService) publishEvent(ctx context.Context, eventType string, payload map[string]interface{}) {
	event := map[string]interface{}{
		"event_id":   uuid.New().String(),
		"event_type": eventType,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"version":    "1.0.0",
		"payload":    payload,
	}

	data, err := json.Marshal(event)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to marshal user event")
		return
	}

	if err := s.kafkaProducer.Publish(ctx, "user-events", event["event_id"].(string), data); err != nil {
		s.logger.WithError(err).Warn("Failed to publish user event")
	}
}
