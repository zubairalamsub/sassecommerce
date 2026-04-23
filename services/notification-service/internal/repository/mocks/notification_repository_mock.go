package mocks

import (
	"context"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockNotificationRepository struct {
	mock.Mock
}

func (m *MockNotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotificationRepository) GetByID(ctx context.Context, id string) (*models.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Notification), args.Error(1)
}

func (m *MockNotificationRepository) GetByUserID(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.Notification, int64, error) {
	args := m.Called(ctx, tenantID, userID, page, pageSize)
	return args.Get(0).([]models.Notification), args.Get(1).(int64), args.Error(2)
}

func (m *MockNotificationRepository) Update(ctx context.Context, notification *models.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotificationRepository) MarkAsRead(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockNotificationRepository) GetPreference(ctx context.Context, tenantID, userID string) (*models.UserPreference, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserPreference), args.Error(1)
}

func (m *MockNotificationRepository) UpsertPreference(ctx context.Context, pref *models.UserPreference) error {
	args := m.Called(ctx, pref)
	return args.Error(0)
}
