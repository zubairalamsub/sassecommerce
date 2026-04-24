package mocks

import (
	"context"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockTokenRepository is a mock implementation of TokenRepository
type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) CreateVerificationToken(ctx context.Context, token *models.VerificationToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockTokenRepository) GetVerificationTokenByToken(ctx context.Context, token string) (*models.VerificationToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VerificationToken), args.Error(1)
}

func (m *MockTokenRepository) InvalidateVerificationTokens(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockTokenRepository) MarkVerificationTokenUsed(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTokenRepository) CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockTokenRepository) GetPasswordResetTokenByToken(ctx context.Context, token string) (*models.PasswordResetToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PasswordResetToken), args.Error(1)
}

func (m *MockTokenRepository) InvalidatePasswordResetTokens(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockTokenRepository) MarkPasswordResetTokenUsed(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
