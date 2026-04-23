package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	TenantID     string    `gorm:"index;not null" json:"tenant_id"`
	Email        string    `gorm:"uniqueIndex:idx_tenant_email;not null" json:"email"`
	Username     string    `gorm:"uniqueIndex:idx_tenant_username" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Phone        string    `json:"phone,omitempty"`
	Avatar       string    `json:"avatar,omitempty"`
	Status       UserStatus `gorm:"default:'active'" json:"status"`
	Role         UserRole  `gorm:"default:'customer'" json:"role"`
	EmailVerified bool     `gorm:"default:false" json:"email_verified"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// UserStatus represents the status of a user
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDeleted   UserStatus = "deleted"
)

// UserRole represents the role of a user
type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleModerator UserRole = "moderator"
	UserRoleCustomer UserRole = "customer"
	UserRoleGuest    UserRole = "guest"
)

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}

// BeforeCreate generates a UUID if not already set
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	TenantID  string `json:"tenant_id" binding:"required,uuid"`
	Email     string `json:"email" binding:"required,email"`
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Phone     string `json:"phone,omitempty"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	TenantID string `json:"tenant_id" binding:"required,uuid"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UpdateUserRequest represents a user update request
type UpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Avatar    *string `json:"avatar,omitempty"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// UserResponse represents a user response
type UserResponse struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	Email         string     `json:"email"`
	Username      string     `json:"username"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	Phone         string     `json:"phone,omitempty"`
	Avatar        string     `json:"avatar,omitempty"`
	Status        UserStatus `json:"status"`
	Role          UserRole   `json:"role"`
	EmailVerified bool       `json:"email_verified"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User  *UserResponse `json:"user"`
	Token string        `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ToResponse converts User to UserResponse
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:            u.ID,
		TenantID:      u.TenantID,
		Email:         u.Email,
		Username:      u.Username,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Phone:         u.Phone,
		Avatar:        u.Avatar,
		Status:        u.Status,
		Role:          u.Role,
		EmailVerified: u.EmailVerified,
		LastLoginAt:   u.LastLoginAt,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}
