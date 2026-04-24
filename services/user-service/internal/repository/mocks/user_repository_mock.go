package mocks

import (
	"context"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, tenantID, email string) (*models.User, error) {
	args := m.Called(ctx, tenantID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, tenantID, username string) (*models.User, error) {
	args := m.Called(ctx, tenantID, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) List(ctx context.Context, tenantID string, offset, limit int) ([]models.User, int64, error) {
	args := m.Called(ctx, tenantID, offset, limit)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) EmailExists(ctx context.Context, tenantID, email string) (bool, error) {
	args := m.Called(ctx, tenantID, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) UsernameExists(ctx context.Context, tenantID, username string) (bool, error) {
	args := m.Called(ctx, tenantID, username)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	args := m.Called(ctx, userID, passwordHash)
	return args.Error(0)
}

func (m *MockUserRepository) SetEmailVerified(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}
