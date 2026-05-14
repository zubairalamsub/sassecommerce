package mocks

import (
	"context"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockRefreshTokenRepository is a mock implementation of RefreshTokenRepository.
type MockRefreshTokenRepository struct {
	mock.Mock
}

func (m *MockRefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	return m.Called(ctx, token).Error(0)
}

func (m *MockRefreshTokenRepository) GetByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepository) GetByID(ctx context.Context, id string) (*models.RefreshToken, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepository) ListByUser(ctx context.Context, userID string) ([]models.RefreshToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepository) Revoke(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockRefreshTokenRepository) RevokeByToken(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

func (m *MockRefreshTokenRepository) RevokeAllByUser(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockRefreshTokenRepository) UpdateLastUsed(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockRefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}
