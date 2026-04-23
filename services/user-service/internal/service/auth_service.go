package service

import (
	"context"
	"errors"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	Register(ctx context.Context, req *models.RegisterRequest) (*models.UserResponse, error)
	Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error)
	VerifyToken(ctx context.Context, tokenString string) (*models.TokenClaims, error)
	ChangePassword(ctx context.Context, userID string, req *models.ChangePasswordRequest) error
}

type authService struct {
	userRepo    repository.UserRepository
	tokenConfig models.TokenConfig
	logger      *logrus.Logger
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo repository.UserRepository,
	tokenConfig models.TokenConfig,
	logger *logrus.Logger,
) AuthService {
	return &authService{
		userRepo:    userRepo,
		tokenConfig: tokenConfig,
		logger:      logger,
	}
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
