package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// LockoutConfig captures the brute-force protection thresholds applied to
// the login flow. Zero values disable the corresponding check.
type LockoutConfig struct {
	MaxFailedPerEmail        int
	MaxFailedPerIP           int
	Window                   time.Duration
	ForgotPasswordMaxPerHour int
}

// DefaultLockoutConfig returns the documented defaults: 5 failures per email
// and 20 per IP in a 15-minute rolling window, plus 3 forgot-password
// requests per hour.
func DefaultLockoutConfig() LockoutConfig {
	return LockoutConfig{
		MaxFailedPerEmail:        5,
		MaxFailedPerIP:           20,
		Window:                   15 * time.Minute,
		ForgotPasswordMaxPerHour: 3,
	}
}

// LockoutConfigFromEnv overlays env-var overrides on top of the defaults.
// Recognised vars:
//   LOGIN_MAX_FAILED_PER_EMAIL      (default 5)
//   LOGIN_MAX_FAILED_PER_IP         (default 20)
//   LOGIN_LOCKOUT_WINDOW_MINUTES    (default 15)
//   FORGOT_PASSWORD_MAX_PER_HOUR    (default 3)
func LockoutConfigFromEnv() LockoutConfig {
	cfg := DefaultLockoutConfig()
	if v := os.Getenv("LOGIN_MAX_FAILED_PER_EMAIL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxFailedPerEmail = n
		}
	}
	if v := os.Getenv("LOGIN_MAX_FAILED_PER_IP"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxFailedPerIP = n
		}
	}
	if v := os.Getenv("LOGIN_LOCKOUT_WINDOW_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Window = time.Duration(n) * time.Minute
		}
	}
	if v := os.Getenv("FORGOT_PASSWORD_MAX_PER_HOUR"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ForgotPasswordMaxPerHour = n
		}
	}
	return cfg
}

// KafkaPublisher defines the interface for publishing messages to Kafka
type KafkaPublisher interface {
	Publish(ctx context.Context, topic, key string, value []byte) error
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	Register(ctx context.Context, req *models.RegisterRequest) (*models.UserResponse, error)
	Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error)
	LoginWithSession(ctx context.Context, req *models.LoginRequest, sctx SessionContext) (*models.LoginResponse, error)
	VerifyTwoFactorChallenge(ctx context.Context, req *models.TwoFactorVerifyRequest, sctx SessionContext) (*models.LoginResponse, error)
	RefreshAccessToken(ctx context.Context, req *models.RefreshTokenRequest, sctx SessionContext) (*models.LoginResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID string) error
	ListSessions(ctx context.Context, userID, currentRefreshToken string) ([]models.SessionResponse, error)
	RevokeSession(ctx context.Context, userID, sessionID string) error
	VerifyToken(ctx context.Context, tokenString string) (*models.TokenClaims, error)
	VerifyUserPassword(ctx context.Context, userID, password string) error
	ChangePassword(ctx context.Context, userID string, req *models.ChangePasswordRequest) error
	RequestEmailVerification(ctx context.Context, userID, tenantID, email string) (string, error)
	ResendEmailVerification(ctx context.Context, req *models.ResendVerificationRequest) error
	VerifyEmail(ctx context.Context, req *models.VerifyEmailRequest) error
	RequestPasswordReset(ctx context.Context, req *models.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req *models.ResetPasswordRequest) error
}

type authService struct {
	userRepo         repository.UserRepository
	tokenRepo        repository.TokenRepository
	loginAttemptRepo repository.LoginAttemptRepository
	refreshTokenRepo repository.RefreshTokenRepository
	refreshTokenTTL  time.Duration
	twoFactor        TwoFactorService
	lockout          LockoutConfig
	tokenConfig      models.TokenConfig
	kafkaProducer    KafkaPublisher
	logger           *logrus.Logger
}

// AuthServiceOption configures optional dependencies on authService.
type AuthServiceOption func(*authService)

// WithLoginAttemptRepository attaches a brute-force tracking repository.
// When omitted, lockout checks are silently skipped (gracefully degrades).
func WithLoginAttemptRepository(repo repository.LoginAttemptRepository) AuthServiceOption {
	return func(s *authService) { s.loginAttemptRepo = repo }
}

// WithLockoutConfig overrides the lockout thresholds.
func WithLockoutConfig(cfg LockoutConfig) AuthServiceOption {
	return func(s *authService) { s.lockout = cfg }
}

// WithTokenRepository attaches the verification/password-reset token repo.
func WithTokenRepository(repo repository.TokenRepository) AuthServiceOption {
	return func(s *authService) { s.tokenRepo = repo }
}

// WithRefreshTokenRepository enables refresh-token sessions: login issues a
// refresh token, /refresh rotates it, and password changes revoke them all.
func WithRefreshTokenRepository(repo repository.RefreshTokenRepository) AuthServiceOption {
	return func(s *authService) { s.refreshTokenRepo = repo }
}

// WithRefreshTokenTTL overrides how long refresh tokens live (default 30 days).
func WithRefreshTokenTTL(ttl time.Duration) AuthServiceOption {
	return func(s *authService) { s.refreshTokenTTL = ttl }
}

// WithTwoFactorService enables the 2FA gate in the login flow: users with 2FA
// enrolled receive a challenge instead of tokens until they present a code.
func WithTwoFactorService(tf TwoFactorService) AuthServiceOption {
	return func(s *authService) { s.twoFactor = tf }
}

// NewAuthService creates a new authentication service.
//
// The variadic tokenRepo parameter is kept for backward compatibility with
// existing call sites. For new options (lockout, login-attempt repo, etc.)
// prefer NewAuthServiceWithOptions.
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
		lockout:       DefaultLockoutConfig(),
	}
	if len(tokenRepo) > 0 {
		s.tokenRepo = tokenRepo[0]
	}
	return s
}

// NewAuthServiceWithOptions constructs an authService and applies functional
// options. Use this when you need to inject the login-attempt repository,
// override the lockout policy, or attach other optional collaborators.
func NewAuthServiceWithOptions(
	userRepo repository.UserRepository,
	tokenConfig models.TokenConfig,
	kafkaProducer KafkaPublisher,
	logger *logrus.Logger,
	opts ...AuthServiceOption,
) AuthService {
	s := &authService{
		userRepo:      userRepo,
		tokenConfig:   tokenConfig,
		kafkaProducer: kafkaProducer,
		logger:        logger,
		lockout:       DefaultLockoutConfig(),
	}
	for _, opt := range opts {
		opt(s)
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

// Login authenticates a user and returns a JWT token.
//
// Brute-force protection: before any password check we count recent failed
// attempts both for the email and for the source IP. Exceeding either
// threshold returns a lockout error and records the attempt so the lockout
// window keeps sliding for as long as abuse continues.
//
// The lockout error never reveals whether the email belongs to a real
// account — it is the same shape for an unknown email and a real one.
func (s *authService) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	user, err := s.authenticate(ctx, req)
	if err != nil {
		return nil, err
	}

	// 5. Generate JWT token
	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate token")
		return nil, errors.New("failed to generate authentication token")
	}

	// 6-7. Update last-login, clear failure counters, audit, publish.
	s.finishLogin(ctx, user, req.Email, req.IPAddress, req.UserAgent)

	return &models.LoginResponse{
		User:      user.ToResponse(),
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// authenticate runs the credential checks shared by Login and
// LoginWithSession: lockout windows, user lookup, active check, and password
// verification. Failures are recorded and published; the returned errors are
// deliberately generic to prevent account enumeration.
func (s *authService) authenticate(ctx context.Context, req *models.LoginRequest) (*models.User, error) {
	emailKey := strings.ToLower(strings.TrimSpace(req.Email))

	// 1. Pre-check lockout windows (per-email and per-IP).
	if msg, locked := s.checkLockout(ctx, req.TenantID, emailKey, req.IPAddress); locked {
		s.publishLoginFailed(ctx, req.TenantID, "", req.Email, "rate_limited", req.IPAddress, req.UserAgent)
		return nil, errors.New(msg)
	}

	// 2. Get user by email. Don't reveal not-found yet — record the attempt
	//    against the typed email so enumeration is impossible.
	user, err := s.userRepo.GetByEmail(ctx, req.TenantID, req.Email)
	if err != nil {
		s.logger.WithError(err).WithField("email", req.Email).Warn("Login attempt with non-existent email")
		s.recordAttempt(ctx, req.TenantID, "", emailKey, req.IPAddress, req.UserAgent, false, models.LoginAttemptReasonUnknownEmail)
		s.publishLoginFailed(ctx, req.TenantID, "", req.Email, "unknown_email", req.IPAddress, req.UserAgent)
		return nil, errors.New("invalid email or password")
	}

	// 3. Check if user is active
	if user.Status != models.UserStatusActive {
		s.logger.WithField("user_id", user.ID).Warn("Login attempt for inactive user")
		s.recordAttempt(ctx, user.TenantID, user.ID, emailKey, req.IPAddress, req.UserAgent, false, models.LoginAttemptReasonUserInactive)
		s.publishLoginFailed(ctx, user.TenantID, user.ID, req.Email, "user_inactive", req.IPAddress, req.UserAgent)
		return nil, errors.New("user account is not active")
	}

	// 4. Verify password
	if !verifyPassword(user.PasswordHash, req.Password) {
		s.logger.WithField("user_id", user.ID).Warn("Login attempt with incorrect password")
		s.recordAttempt(ctx, user.TenantID, user.ID, emailKey, req.IPAddress, req.UserAgent, false, models.LoginAttemptReasonInvalidPassword)
		s.publishLoginFailed(ctx, user.TenantID, user.ID, req.Email, "invalid_password", req.IPAddress, req.UserAgent)
		return nil, errors.New("invalid email or password")
	}

	return user, nil
}

// finishLogin performs the post-authentication bookkeeping for a completed
// login: last-login timestamp, failure-counter reset, audit row, and the
// LoginSucceeded event. Best-effort: none of these block the login.
func (s *authService) finishLogin(ctx context.Context, user *models.User, email, ip, userAgent string) {
	emailKey := strings.ToLower(strings.TrimSpace(email))

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.WithError(err).Warn("Failed to update last login timestamp")
	}

	if s.loginAttemptRepo != nil {
		if err := s.loginAttemptRepo.MarkSuccess(ctx, emailKey); err != nil {
			s.logger.WithError(err).Warn("Failed to clear failed login attempts")
		}
	}
	s.recordAttempt(ctx, user.TenantID, user.ID, emailKey, ip, userAgent, true, models.LoginAttemptReasonSuccess)

	s.logger.WithFields(logrus.Fields{
		"user_id":   user.ID,
		"tenant_id": user.TenantID,
		"email":     user.Email,
	}).Info("User logged in successfully")

	// Publish LoginSucceeded for the centralised audit log. Includes the
	// originating IP and user agent so security review can correlate sessions.
	s.publishEvent(ctx, "LoginSucceeded", map[string]interface{}{
		"tenant_id":  user.TenantID,
		"user_id":    user.ID,
		"email":      user.Email,
		"role":       user.Role,
		"ip_address": ip,
		"user_agent": userAgent,
	})
}

// publishLoginFailed emits a LoginFailed event for the centralised audit log.
// The email is included (never the password) so operators can spot
// enumeration / brute-force patterns. user_id may be empty when the email is
// unknown. Publishing is best-effort: failures are logged but never returned.
func (s *authService) publishLoginFailed(ctx context.Context, tenantID, userID, email, reason, ip, userAgent string) {
	payload := map[string]interface{}{
		"user_id":    userID,
		"email":      email,
		"reason":     reason,
		"ip_address": ip,
		"user_agent": userAgent,
	}
	// tenant_id is needed by the audit consumer to bucket the event. If we
	// don't have one (unknown-email path with no tenant context) the consumer
	// will skip the row rather than crash.
	if tenantID != "" {
		payload["tenant_id"] = tenantID
	}
	s.publishEvent(ctx, "LoginFailed", payload)
}

// checkLockout consults the LoginAttempt repository for recent failures and
// returns (message, true) if either the per-email or per-IP threshold has
// been hit. The locked-out attempt itself is also recorded so the window
// keeps extending while the abuse continues.
//
// When the repository isn't configured (graceful-degradation mode) this
// always returns ("", false) — the older behavior with no lockout.
func (s *authService) checkLockout(ctx context.Context, tenantID, email, ip string) (string, bool) {
	if s.loginAttemptRepo == nil {
		return "", false
	}
	window := s.lockout.Window
	if window <= 0 {
		window = 15 * time.Minute
	}
	since := time.Now().Add(-window)

	// Per-email check.
	if s.lockout.MaxFailedPerEmail > 0 {
		count, err := s.loginAttemptRepo.CountRecentFailedByEmail(ctx, tenantID, email, since)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to count failed login attempts by email; allowing login")
		} else if count >= s.lockout.MaxFailedPerEmail {
			s.recordAttempt(ctx, tenantID, "", email, ip, "", false, models.LoginAttemptReasonAccountLocked)
			mins := int(window.Minutes())
			if mins < 1 {
				mins = 1
			}
			return fmt.Sprintf("Too many login attempts. Try again in %d minutes", mins), true
		}
	}

	// Per-IP check.
	if s.lockout.MaxFailedPerIP > 0 && ip != "" {
		count, err := s.loginAttemptRepo.CountRecentFailedByIP(ctx, ip, since)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to count failed login attempts by IP; allowing login")
		} else if count >= s.lockout.MaxFailedPerIP {
			s.recordAttempt(ctx, tenantID, "", email, ip, "", false, models.LoginAttemptReasonRateLimited)
			return "Too many login attempts from your network", true
		}
	}

	return "", false
}

// recordAttempt persists a LoginAttempt. Errors are logged but never
// surfaced — audit logging must not block authentication.
func (s *authService) recordAttempt(ctx context.Context, tenantID, _ string, email, ip, ua string, successful bool, reason string) {
	if s.loginAttemptRepo == nil {
		return
	}
	attempt := &models.LoginAttempt{
		TenantID:   tenantID,
		Email:      email,
		IPAddress:  ip,
		UserAgent:  ua,
		Successful: successful,
		Reason:     reason,
	}
	if err := s.loginAttemptRepo.Create(ctx, attempt); err != nil {
		s.logger.WithError(err).Warn("Failed to record login attempt")
	}
}

// forgotPasswordReasonPrefix marks LoginAttempt rows used purely for
// forgot-password rate-limiting. The rows are kept separate from login
// failures by being recorded as Successful=true so they do not contribute
// to the per-email lockout counter; the reason value lets us filter them
// when counting forgot-password traffic.
const (
	forgotPasswordReason        = "forgot_password"
	forgotPasswordRateLimitedReason = "forgot_password_rate_limited"
)

// countForgotPasswordRequests returns how many forgot-password requests we
// have seen for an email within the given time window.
func (s *authService) countForgotPasswordRequests(ctx context.Context, tenantID, email string, since time.Time) (int, error) {
	if s.loginAttemptRepo == nil {
		return 0, nil
	}
	return s.loginAttemptRepo.CountRecentByEmailAndReason(ctx, tenantID, email, forgotPasswordReason, since)
}

// recordForgotPasswordRequest writes a marker row so future rate-limit
// checks can see the request. Recorded as Successful=true to keep these
// rows out of the login failure counter.
func (s *authService) recordForgotPasswordRequest(ctx context.Context, tenantID, email, ip, ua, reason string) {
	if s.loginAttemptRepo == nil {
		return
	}
	r := forgotPasswordReason
	if reason == "rate_limited" {
		r = forgotPasswordRateLimitedReason
	}
	attempt := &models.LoginAttempt{
		TenantID:   tenantID,
		Email:      email,
		IPAddress:  ip,
		UserAgent:  ua,
		Successful: true,
		Reason:     r,
	}
	if err := s.loginAttemptRepo.Create(ctx, attempt); err != nil {
		s.logger.WithError(err).Warn("Failed to record forgot-password request")
	}
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

	// Revoke every live session: a stolen refresh token must not survive a
	// password change.
	s.revokeAllSessions(ctx, userID)

	// Publish PasswordChanged so the audit log records the change. The
	// "changed_by" is the user themselves (self-service flow); admin-initiated
	// password changes would go through a different code path with a separate
	// event payload.
	s.publishEvent(ctx, "PasswordChanged", map[string]interface{}{
		"tenant_id":  user.TenantID,
		"user_id":    user.ID,
		"changed_by": "self",
	})

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

// RequestPasswordReset generates a password reset token and publishes an event.
//
// Forgot-password requests are rate-limited per email (default 3/hour) to
// protect the email-sending pipeline from abuse. The limit is applied
// before the user lookup so it also serves to dampen email enumeration
// probes. A throttled request is silently dropped — the caller always sees
// the same "if the email exists..." response shape from the HTTP handler.
func (s *authService) RequestPasswordReset(ctx context.Context, req *models.ForgotPasswordRequest) error {
	emailKey := strings.ToLower(strings.TrimSpace(req.Email))

	// Rate-limit per email. We piggyback on the LoginAttempt table by using
	// a synthetic reason value, which keeps the schema lean.
	if s.loginAttemptRepo != nil && s.lockout.ForgotPasswordMaxPerHour > 0 {
		since := time.Now().Add(-time.Hour)
		count, err := s.countForgotPasswordRequests(ctx, req.TenantID, emailKey, since)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to count recent password-reset requests; allowing")
		} else if count >= s.lockout.ForgotPasswordMaxPerHour {
			s.logger.WithField("email", emailKey).Warn("Forgot-password rate limit exceeded; dropping request")
			// Record the throttled attempt so the window keeps sliding,
			// then return nil — caller behavior is identical to the
			// unknown-email path.
			s.recordForgotPasswordRequest(ctx, req.TenantID, emailKey, req.IPAddress, req.UserAgent, "rate_limited")
			return nil
		}
		s.recordForgotPasswordRequest(ctx, req.TenantID, emailKey, req.IPAddress, req.UserAgent, "forgot_password")
	}

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

	// Revoke every live session: whoever reset the password (typically after
	// a compromise) must be the only one left logged in.
	s.revokeAllSessions(ctx, prt.UserID)

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
