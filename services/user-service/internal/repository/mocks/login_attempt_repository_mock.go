package mocks

import (
	"context"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockLoginAttemptRepository is a mock implementation of LoginAttemptRepository.
type MockLoginAttemptRepository struct {
	mock.Mock
}

func (m *MockLoginAttemptRepository) Create(ctx context.Context, attempt *models.LoginAttempt) error {
	args := m.Called(ctx, attempt)
	return args.Error(0)
}

func (m *MockLoginAttemptRepository) CountRecentFailedByEmail(ctx context.Context, tenantID, email string, since time.Time) (int, error) {
	args := m.Called(ctx, tenantID, email, since)
	return args.Int(0), args.Error(1)
}

func (m *MockLoginAttemptRepository) CountRecentFailedByIP(ctx context.Context, ip string, since time.Time) (int, error) {
	args := m.Called(ctx, ip, since)
	return args.Int(0), args.Error(1)
}

func (m *MockLoginAttemptRepository) CountRecentByEmailAndReason(ctx context.Context, tenantID, email, reason string, since time.Time) (int, error) {
	args := m.Called(ctx, tenantID, email, reason, since)
	return args.Int(0), args.Error(1)
}

func (m *MockLoginAttemptRepository) MarkSuccess(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

func (m *MockLoginAttemptRepository) DeleteOlderThan(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}
