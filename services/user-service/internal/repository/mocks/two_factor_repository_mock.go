package mocks

import (
	"context"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockTwoFactorRepository is a mock implementation of TwoFactorRepository.
type MockTwoFactorRepository struct {
	mock.Mock
}

func (m *MockTwoFactorRepository) GetByUserID(ctx context.Context, userID string) (*models.TwoFactorSecret, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TwoFactorSecret), args.Error(1)
}

func (m *MockTwoFactorRepository) Upsert(ctx context.Context, secret *models.TwoFactorSecret) error {
	return m.Called(ctx, secret).Error(0)
}

func (m *MockTwoFactorRepository) Delete(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockTwoFactorRepository) UpdateLastUsed(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockTwoFactorRepository) CreateBackupCodes(ctx context.Context, codes []models.TwoFactorBackupCode) error {
	return m.Called(ctx, codes).Error(0)
}

func (m *MockTwoFactorRepository) DeleteBackupCodes(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockTwoFactorRepository) ListUnusedBackupCodes(ctx context.Context, userID string) ([]models.TwoFactorBackupCode, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TwoFactorBackupCode), args.Error(1)
}

func (m *MockTwoFactorRepository) MarkBackupCodeUsed(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockTwoFactorRepository) MigrateLegacyPlaintextSecrets(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}
