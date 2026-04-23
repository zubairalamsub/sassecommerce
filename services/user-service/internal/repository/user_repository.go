package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"gorm.io/gorm"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, tenantID, email string) (*models.User, error)
	GetByUsername(ctx context.Context, tenantID, username string) (*models.User, error)
	List(ctx context.Context, tenantID string, offset, limit int) ([]models.User, int64, error)
	Update(ctx context.Context, user *models.User) error
	UpdateLastLogin(ctx context.Context, userID string) error
	Delete(ctx context.Context, id string) error
	EmailExists(ctx context.Context, tenantID, email string) (bool, error)
	UsernameExists(ctx context.Context, tenantID, username string) (bool, error)
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create creates a new user
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetByID retrieves a user by ID
func (r *userRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	return &user, nil
}

// GetByEmail retrieves a user by email within a tenant
func (r *userRepository) GetByEmail(ctx context.Context, tenantID, email string) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND email = ? AND deleted_at IS NULL", tenantID, email).
		First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	return &user, nil
}

// GetByUsername retrieves a user by username within a tenant
func (r *userRepository) GetByUsername(ctx context.Context, tenantID, username string) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND username = ? AND deleted_at IS NULL", tenantID, username).
		First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	return &user, nil
}

// List retrieves users with pagination
func (r *userRepository) List(ctx context.Context, tenantID string, offset, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// Get total count
	countResult := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Count(&total)

	if countResult.Error != nil {
		return nil, 0, countResult.Error
	}

	// Get paginated results
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&users)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	return users, total, nil
}

// Update updates a user
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()
	result := r.db.WithContext(ctx).
		Model(user).
		Where("id = ? AND deleted_at IS NULL", user.ID).
		Updates(user)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *userRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Update("last_login_at", now)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Delete soft deletes a user
func (r *userRepository) Delete(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"status":     models.UserStatusDeleted,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// EmailExists checks if an email already exists for a tenant
func (r *userRepository) EmailExists(ctx context.Context, tenantID, email string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("tenant_id = ? AND email = ? AND deleted_at IS NULL", tenantID, email).
		Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}

// UsernameExists checks if a username already exists for a tenant
func (r *userRepository) UsernameExists(ctx context.Context, tenantID, username string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("tenant_id = ? AND username = ? AND deleted_at IS NULL", tenantID, username).
		Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}
