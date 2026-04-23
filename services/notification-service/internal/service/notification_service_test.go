package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ecommerce/notification-service/internal/models"
	repoMocks "github.com/ecommerce/notification-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockProvider implements NotificationProvider for testing
type mockProvider struct {
	mock.Mock
	channel models.Channel
}

func (m *mockProvider) Send(notification *models.Notification) (*ProviderResult, error) {
	args := m.Called(notification)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ProviderResult), args.Error(1)
}

func (m *mockProvider) Channel() models.Channel {
	return m.channel
}

func newTestService() (*notificationService, *repoMocks.MockNotificationRepository, *mockProvider) {
	mockRepo := new(repoMocks.MockNotificationRepository)
	mockProv := &mockProvider{channel: models.ChannelEmail}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	providers := map[models.Channel]NotificationProvider{
		models.ChannelEmail: mockProv,
	}

	svc := &notificationService{
		repo:      mockRepo,
		providers: providers,
		logger:    logger,
	}

	return svc, mockRepo, mockProv
}

func createTestSendRequest() *models.SendNotificationRequest {
	return &models.SendNotificationRequest{
		TenantID:      "tenant-1",
		UserID:        "user-1",
		Channel:       "email",
		Type:          "order_confirmation",
		Subject:       "Order Confirmed",
		Body:          "Your order has been confirmed.",
		Recipient:     "user@example.com",
		ReferenceID:   "order-1",
		ReferenceType: "order",
	}
}

// === SendNotification Tests ===

func TestSendNotification_Success(t *testing.T) {
	svc, mockRepo, mockProv := newTestService()
	ctx := context.Background()
	req := createTestSendRequest()

	mockRepo.On("GetPreference", ctx, "tenant-1", "user-1").Return(nil, nil)
	mockProv.On("Send", mock.AnythingOfType("*models.Notification")).Return(&ProviderResult{
		ProviderName: "simulated-sendgrid",
		MessageID:    "email_123456789",
		Success:      true,
	}, nil)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(nil)

	result, err := svc.SendNotification(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusSent, result.Status)
	assert.Equal(t, "order_confirmation", string(result.Type))
	assert.Equal(t, "user@example.com", result.Recipient)
	assert.NotNil(t, result.SentAt)
}

func TestSendNotification_ProviderFailure(t *testing.T) {
	svc, mockRepo, mockProv := newTestService()
	ctx := context.Background()
	req := createTestSendRequest()

	mockRepo.On("GetPreference", ctx, "tenant-1", "user-1").Return(nil, nil)
	mockProv.On("Send", mock.AnythingOfType("*models.Notification")).
		Return(nil, errors.New("provider error"))
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(nil)

	result, err := svc.SendNotification(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to send notification")
}

func TestSendNotification_ProviderReturnsFailed(t *testing.T) {
	svc, mockRepo, mockProv := newTestService()
	ctx := context.Background()
	req := createTestSendRequest()

	mockRepo.On("GetPreference", ctx, "tenant-1", "user-1").Return(nil, nil)
	mockProv.On("Send", mock.AnythingOfType("*models.Notification")).Return(&ProviderResult{
		ProviderName: "simulated-sendgrid",
		Success:      false,
		Error:        "recipient email is required",
	}, nil)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(nil)

	result, err := svc.SendNotification(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusFailed, result.Status)
	assert.Equal(t, "recipient email is required", result.FailureReason)
}

func TestSendNotification_NoProviderForChannel(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()
	req := createTestSendRequest()
	req.Channel = "sms" // no SMS provider in test setup

	mockRepo.On("GetPreference", ctx, "tenant-1", "user-1").Return(nil, nil)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(nil)

	result, err := svc.SendNotification(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no provider configured for channel: sms")
}

func TestSendNotification_ChannelDisabled(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()
	req := createTestSendRequest()

	pref := &models.UserPreference{
		TenantID:     "tenant-1",
		UserID:       "user-1",
		EmailEnabled: false,
		SMSEnabled:   true,
		PushEnabled:  true,
	}
	mockRepo.On("GetPreference", ctx, "tenant-1", "user-1").Return(pref, nil)

	result, err := svc.SendNotification(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user has disabled email notifications")
}

func TestSendNotification_TypeOptedOut(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()
	req := createTestSendRequest()
	req.Type = "promotion"

	pref := &models.UserPreference{
		TenantID:     "tenant-1",
		UserID:       "user-1",
		EmailEnabled: true,
		SMSEnabled:   true,
		PushEnabled:  true,
		OptedOut:     []models.NotificationType{models.TypePromotion},
	}
	mockRepo.On("GetPreference", ctx, "tenant-1", "user-1").Return(pref, nil)

	result, err := svc.SendNotification(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user has opted out of promotion notifications")
}

func TestSendNotification_RepoCreateFails(t *testing.T) {
	svc, mockRepo, mockProv := newTestService()
	ctx := context.Background()
	req := createTestSendRequest()

	mockRepo.On("GetPreference", ctx, "tenant-1", "user-1").Return(nil, nil)
	mockProv.On("Send", mock.AnythingOfType("*models.Notification")).Return(&ProviderResult{
		ProviderName: "simulated-sendgrid",
		MessageID:    "email_123",
		Success:      true,
	}, nil)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Notification")).Return(errors.New("db error"))

	result, err := svc.SendNotification(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to save notification")
}

// === GetNotification Tests ===

func TestGetNotification_Success(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	notification := &models.Notification{
		ID:        "notif-1",
		TenantID:  "tenant-1",
		UserID:    "user-1",
		Channel:   models.ChannelEmail,
		Type:      models.TypeOrderConfirmation,
		Status:    models.StatusSent,
		Subject:   "Order Confirmed",
		Body:      "Your order has been confirmed.",
		Recipient: "user@example.com",
		CreatedAt: time.Now().UTC(),
	}

	mockRepo.On("GetByID", ctx, "notif-1").Return(notification, nil)

	result, err := svc.GetNotification(ctx, "notif-1")

	assert.NoError(t, err)
	assert.Equal(t, "notif-1", result.ID)
	assert.Equal(t, models.StatusSent, result.Status)
}

func TestGetNotification_NotFound(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "nonexistent").Return(nil, errors.New("notification not found"))

	result, err := svc.GetNotification(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === GetUserNotifications Tests ===

func TestGetUserNotifications_Success(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	notifications := []models.Notification{
		{
			ID:        "notif-1",
			TenantID:  "tenant-1",
			UserID:    "user-1",
			Channel:   models.ChannelEmail,
			Type:      models.TypeOrderConfirmation,
			Status:    models.StatusSent,
			Subject:   "Order Confirmed",
			Recipient: "user@example.com",
			CreatedAt: time.Now().UTC(),
		},
	}

	mockRepo.On("GetByUserID", ctx, "tenant-1", "user-1", 1, 20).Return(notifications, int64(1), nil)

	results, total, err := svc.GetUserNotifications(ctx, "tenant-1", "user-1", 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, "notif-1", results[0].ID)
}

func TestGetUserNotifications_Empty(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByUserID", ctx, "tenant-1", "user-1", 1, 20).Return([]models.Notification{}, int64(0), nil)

	results, total, err := svc.GetUserNotifications(ctx, "tenant-1", "user-1", 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, results, 0)
}

// === MarkAsRead Tests ===

func TestMarkAsRead_Success(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	notification := &models.Notification{
		ID:     "notif-1",
		Status: models.StatusSent,
	}
	mockRepo.On("GetByID", ctx, "notif-1").Return(notification, nil)
	mockRepo.On("MarkAsRead", ctx, "notif-1").Return(nil)

	err := svc.MarkAsRead(ctx, "notif-1")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMarkAsRead_NotFound(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	mockRepo.On("GetByID", ctx, "nonexistent").Return(nil, errors.New("notification not found"))

	err := svc.MarkAsRead(ctx, "nonexistent")

	assert.Error(t, err)
}

// === GetPreference Tests ===

func TestGetPreference_Exists(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	pref := &models.UserPreference{
		ID:           "pref-1",
		TenantID:     "tenant-1",
		UserID:       "user-1",
		EmailEnabled: true,
		SMSEnabled:   false,
		PushEnabled:  true,
		Email:        "user@example.com",
	}
	mockRepo.On("GetPreference", ctx, "tenant-1", "user-1").Return(pref, nil)

	result, err := svc.GetPreference(ctx, "tenant-1", "user-1")

	assert.NoError(t, err)
	assert.True(t, result.EmailEnabled)
	assert.False(t, result.SMSEnabled)
	assert.True(t, result.PushEnabled)
	assert.Equal(t, "user@example.com", result.Email)
}

func TestGetPreference_DefaultsWhenNone(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	mockRepo.On("GetPreference", ctx, "tenant-1", "user-1").Return(nil, nil)

	result, err := svc.GetPreference(ctx, "tenant-1", "user-1")

	assert.NoError(t, err)
	assert.True(t, result.EmailEnabled)
	assert.True(t, result.SMSEnabled)
	assert.True(t, result.PushEnabled)
	assert.Empty(t, result.OptedOut)
}

// === UpdatePreference Tests ===

func TestUpdatePreference_NewPreference(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	mockRepo.On("GetPreference", ctx, "tenant-1", "user-1").Return(nil, nil)
	mockRepo.On("UpsertPreference", ctx, mock.AnythingOfType("*models.UserPreference")).Return(nil)

	emailEnabled := false
	req := &models.UpdatePreferenceRequest{
		EmailEnabled: &emailEnabled,
		Email:        "new@example.com",
	}

	result, err := svc.UpdatePreference(ctx, "tenant-1", "user-1", req)

	assert.NoError(t, err)
	assert.False(t, result.EmailEnabled)
	assert.True(t, result.SMSEnabled) // default
	assert.True(t, result.PushEnabled) // default
	assert.Equal(t, "new@example.com", result.Email)
}

func TestUpdatePreference_UpdateExisting(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	existing := &models.UserPreference{
		ID:           "pref-1",
		TenantID:     "tenant-1",
		UserID:       "user-1",
		EmailEnabled: true,
		SMSEnabled:   true,
		PushEnabled:  true,
		Email:        "old@example.com",
	}
	mockRepo.On("GetPreference", ctx, "tenant-1", "user-1").Return(existing, nil)
	mockRepo.On("UpsertPreference", ctx, mock.AnythingOfType("*models.UserPreference")).Return(nil)

	smsEnabled := false
	req := &models.UpdatePreferenceRequest{
		SMSEnabled:  &smsEnabled,
		PhoneNumber: "+1234567890",
		OptedOut:    []models.NotificationType{models.TypePromotion},
	}

	result, err := svc.UpdatePreference(ctx, "tenant-1", "user-1", req)

	assert.NoError(t, err)
	assert.True(t, result.EmailEnabled)
	assert.False(t, result.SMSEnabled)
	assert.Equal(t, "+1234567890", result.PhoneNumber)
	assert.Equal(t, "old@example.com", result.Email) // not changed
	assert.Contains(t, result.OptedOut, models.TypePromotion)
}

func TestUpdatePreference_UpsertFails(t *testing.T) {
	svc, mockRepo, _ := newTestService()
	ctx := context.Background()

	mockRepo.On("GetPreference", ctx, "tenant-1", "user-1").Return(nil, nil)
	mockRepo.On("UpsertPreference", ctx, mock.AnythingOfType("*models.UserPreference")).Return(errors.New("db error"))

	req := &models.UpdatePreferenceRequest{Email: "test@example.com"}

	result, err := svc.UpdatePreference(ctx, "tenant-1", "user-1", req)

	assert.Error(t, err)
	assert.Nil(t, result)
}
