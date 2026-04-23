package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// UserService defines the interface for user operations
type UserService interface {
	GetUserByID(ctx context.Context, id string) (*models.UserResponse, error)
	GetUserByEmail(ctx context.Context, tenantID, email string) (*models.UserResponse, error)
	ListUsers(ctx context.Context, tenantID string, offset, limit int) ([]models.UserResponse, int64, error)
	UpdateUser(ctx context.Context, userID string, req *models.UpdateUserRequest) (*models.UserResponse, error)
	DeleteUser(ctx context.Context, userID string) error
	UpdateUserRole(ctx context.Context, userID string, role models.UserRole) error
	UpdateUserStatus(ctx context.Context, userID string, status models.UserStatus) error
}

type userService struct {
	userRepo      repository.UserRepository
	kafkaProducer KafkaPublisher
	logger        *logrus.Logger
}

// NewUserService creates a new user service
func NewUserService(
	userRepo repository.UserRepository,
	kafkaProducer KafkaPublisher,
	logger *logrus.Logger,
) UserService {
	return &userService{
		userRepo:      userRepo,
		kafkaProducer: kafkaProducer,
		logger:        logger,
	}
}

// GetUserByID retrieves a user by ID
func (s *userService) GetUserByID(ctx context.Context, id string) (*models.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", id).Error("Failed to get user")
		return nil, errors.New("user not found")
	}

	return user.ToResponse(), nil
}

// GetUserByEmail retrieves a user by email
func (s *userService) GetUserByEmail(ctx context.Context, tenantID, email string) (*models.UserResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, tenantID, email)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"email":     email,
		}).Error("Failed to get user by email")
		return nil, errors.New("user not found")
	}

	return user.ToResponse(), nil
}

// ListUsers retrieves users with pagination
func (s *userService) ListUsers(ctx context.Context, tenantID string, offset, limit int) ([]models.UserResponse, int64, error) {
	users, total, err := s.userRepo.List(ctx, tenantID, offset, limit)
	if err != nil {
		s.logger.WithError(err).WithField("tenant_id", tenantID).Error("Failed to list users")
		return nil, 0, errors.New("failed to retrieve users")
	}

	responses := make([]models.UserResponse, len(users))
	for i, user := range users {
		responses[i] = *user.ToResponse()
	}

	return responses, total, nil
}

// UpdateUser updates a user's profile
func (s *userService) UpdateUser(ctx context.Context, userID string, req *models.UpdateUserRequest) (*models.UserResponse, error) {
	// Get existing user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user for update")
		return nil, errors.New("user not found")
	}

	// Update fields if provided
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	if req.Avatar != nil {
		user.Avatar = *req.Avatar
	}

	// Save changes
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to update user")
		return nil, errors.New("failed to update user")
	}

	s.logger.WithField("user_id", userID).Info("User updated successfully")

	// Publish UserUpdated event
	s.publishEvent(ctx, "UserUpdated", map[string]interface{}{
		"tenant_id": user.TenantID,
		"user_id":   user.ID,
		"email":     user.Email,
		"name":      user.FirstName + " " + user.LastName,
	})

	return user.ToResponse(), nil
}

// DeleteUser deletes a user (soft delete)
func (s *userService) DeleteUser(ctx context.Context, userID string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user for delete")
		return errors.New("user not found")
	}

	if err := s.userRepo.Delete(ctx, userID); err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to delete user")
		return errors.New("failed to delete user")
	}

	s.logger.WithField("user_id", userID).Info("User deleted successfully")

	// Publish UserDeleted event
	s.publishEvent(ctx, "UserDeleted", map[string]interface{}{
		"tenant_id": user.TenantID,
		"user_id":   userID,
	})

	return nil
}

// UpdateUserRole updates a user's role (admin function)
func (s *userService) UpdateUserRole(ctx context.Context, userID string, role models.UserRole) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user for role update")
		return errors.New("user not found")
	}

	user.Role = role

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to update user role")
		return errors.New("failed to update user role")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"role":    role,
	}).Info("User role updated successfully")

	return nil
}

// UpdateUserStatus updates a user's status (admin function)
func (s *userService) UpdateUserStatus(ctx context.Context, userID string, status models.UserStatus) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user for status update")
		return errors.New("user not found")
	}

	user.Status = status

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to update user status")
		return errors.New("failed to update user status")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"status":  status,
	}).Info("User status updated successfully")

	return nil
}

// publishEvent publishes an event to Kafka (non-blocking, logs warning on failure)
func (s *userService) publishEvent(ctx context.Context, eventType string, payload map[string]interface{}) {
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
