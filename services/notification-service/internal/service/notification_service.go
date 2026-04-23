package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type NotificationService interface {
	SendNotification(ctx context.Context, req *models.SendNotificationRequest) (*models.NotificationResponse, error)
	GetNotification(ctx context.Context, id string) (*models.NotificationResponse, error)
	GetUserNotifications(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.NotificationResponse, int64, error)
	MarkAsRead(ctx context.Context, id string) error
	GetPreference(ctx context.Context, tenantID, userID string) (*models.UserPreferenceResponse, error)
	UpdatePreference(ctx context.Context, tenantID, userID string, req *models.UpdatePreferenceRequest) (*models.UserPreferenceResponse, error)
}

type notificationService struct {
	repo      repository.NotificationRepository
	providers map[models.Channel]NotificationProvider
	logger    *logrus.Logger
}

func NewNotificationService(repo repository.NotificationRepository, providers map[models.Channel]NotificationProvider, logger *logrus.Logger) NotificationService {
	return &notificationService{
		repo:      repo,
		providers: providers,
		logger:    logger,
	}
}

func (s *notificationService) SendNotification(ctx context.Context, req *models.SendNotificationRequest) (*models.NotificationResponse, error) {
	channel := models.Channel(req.Channel)

	// Check if user has opted out
	pref, _ := s.repo.GetPreference(ctx, req.TenantID, req.UserID)
	if pref != nil {
		if !s.isChannelEnabled(pref, channel) {
			return nil, fmt.Errorf("user has disabled %s notifications", req.Channel)
		}
		if s.isOptedOut(pref, models.NotificationType(req.Type)) {
			return nil, fmt.Errorf("user has opted out of %s notifications", req.Type)
		}
	}

	notification := &models.Notification{
		ID:            uuid.New().String(),
		TenantID:      req.TenantID,
		UserID:        req.UserID,
		Channel:       channel,
		Type:          models.NotificationType(req.Type),
		Status:        models.StatusPending,
		Subject:       req.Subject,
		Body:          req.Body,
		Recipient:     req.Recipient,
		ReferenceID:   req.ReferenceID,
		ReferenceType: req.ReferenceType,
		Metadata:      req.Metadata,
	}

	// Get provider for channel
	provider, ok := s.providers[channel]
	if !ok {
		notification.Status = models.StatusFailed
		notification.FailureReason = fmt.Sprintf("no provider configured for channel: %s", channel)
		if err := s.repo.Create(ctx, notification); err != nil {
			s.logger.WithError(err).Error("Failed to save failed notification")
		}
		return nil, fmt.Errorf("no provider configured for channel: %s", channel)
	}

	// Send via provider
	result, err := provider.Send(notification)
	if err != nil {
		notification.Status = models.StatusFailed
		notification.FailureReason = err.Error()
		if saveErr := s.repo.Create(ctx, notification); saveErr != nil {
			s.logger.WithError(saveErr).Error("Failed to save failed notification")
		}
		return nil, fmt.Errorf("failed to send notification: %w", err)
	}

	now := time.Now().UTC()
	if result.Success {
		notification.Status = models.StatusSent
		notification.SentAt = &now
		notification.ProviderName = result.ProviderName
		notification.ProviderMessageID = result.MessageID
	} else {
		notification.Status = models.StatusFailed
		notification.FailureReason = result.Error
		notification.ProviderName = result.ProviderName
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		s.logger.WithError(err).Error("Failed to save notification")
		return nil, fmt.Errorf("failed to save notification: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID,
		"channel":         channel,
		"type":            req.Type,
		"status":          notification.Status,
	}).Info("Notification processed")

	return toNotificationResponse(notification), nil
}

func (s *notificationService) GetNotification(ctx context.Context, id string) (*models.NotificationResponse, error) {
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toNotificationResponse(notification), nil
}

func (s *notificationService) GetUserNotifications(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.NotificationResponse, int64, error) {
	notifications, total, err := s.repo.GetByUserID(ctx, tenantID, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.NotificationResponse, len(notifications))
	for i, n := range notifications {
		responses[i] = *toNotificationResponse(&n)
	}

	return responses, total, nil
}

func (s *notificationService) MarkAsRead(ctx context.Context, id string) error {
	// Verify notification exists
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.MarkAsRead(ctx, id)
}

func (s *notificationService) GetPreference(ctx context.Context, tenantID, userID string) (*models.UserPreferenceResponse, error) {
	pref, err := s.repo.GetPreference(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	// Return defaults if no preference exists
	if pref == nil {
		return &models.UserPreferenceResponse{
			TenantID:     tenantID,
			UserID:       userID,
			EmailEnabled: true,
			SMSEnabled:   true,
			PushEnabled:  true,
			OptedOut:     []models.NotificationType{},
		}, nil
	}

	return toPreferenceResponse(pref), nil
}

func (s *notificationService) UpdatePreference(ctx context.Context, tenantID, userID string, req *models.UpdatePreferenceRequest) (*models.UserPreferenceResponse, error) {
	pref, _ := s.repo.GetPreference(ctx, tenantID, userID)

	if pref == nil {
		pref = &models.UserPreference{
			ID:           uuid.New().String(),
			TenantID:     tenantID,
			UserID:       userID,
			EmailEnabled: true,
			SMSEnabled:   true,
			PushEnabled:  true,
		}
	}

	if req.EmailEnabled != nil {
		pref.EmailEnabled = *req.EmailEnabled
	}
	if req.SMSEnabled != nil {
		pref.SMSEnabled = *req.SMSEnabled
	}
	if req.PushEnabled != nil {
		pref.PushEnabled = *req.PushEnabled
	}
	if req.OptedOut != nil {
		pref.OptedOut = req.OptedOut
	}
	if req.Email != "" {
		pref.Email = req.Email
	}
	if req.PhoneNumber != "" {
		pref.PhoneNumber = req.PhoneNumber
	}
	if req.DeviceToken != "" {
		pref.DeviceToken = req.DeviceToken
	}

	if err := s.repo.UpsertPreference(ctx, pref); err != nil {
		return nil, fmt.Errorf("failed to update preferences: %w", err)
	}

	return toPreferenceResponse(pref), nil
}

func (s *notificationService) isChannelEnabled(pref *models.UserPreference, channel models.Channel) bool {
	switch channel {
	case models.ChannelEmail:
		return pref.EmailEnabled
	case models.ChannelSMS:
		return pref.SMSEnabled
	case models.ChannelPush:
		return pref.PushEnabled
	}
	return true
}

func (s *notificationService) isOptedOut(pref *models.UserPreference, notifType models.NotificationType) bool {
	for _, opt := range pref.OptedOut {
		if opt == notifType {
			return true
		}
	}
	return false
}

func toNotificationResponse(n *models.Notification) *models.NotificationResponse {
	return &models.NotificationResponse{
		ID:            n.ID,
		TenantID:      n.TenantID,
		UserID:        n.UserID,
		Channel:       n.Channel,
		Type:          n.Type,
		Status:        n.Status,
		Subject:       n.Subject,
		Body:          n.Body,
		Recipient:     n.Recipient,
		ReferenceID:   n.ReferenceID,
		ReferenceType: n.ReferenceType,
		FailureReason: n.FailureReason,
		ReadAt:        n.ReadAt,
		SentAt:        n.SentAt,
		CreatedAt:     n.CreatedAt,
	}
}

func toPreferenceResponse(p *models.UserPreference) *models.UserPreferenceResponse {
	optedOut := p.OptedOut
	if optedOut == nil {
		optedOut = []models.NotificationType{}
	}
	return &models.UserPreferenceResponse{
		ID:           p.ID,
		TenantID:     p.TenantID,
		UserID:       p.UserID,
		EmailEnabled: p.EmailEnabled,
		SMSEnabled:   p.SMSEnabled,
		PushEnabled:  p.PushEnabled,
		OptedOut:     optedOut,
		Email:        p.Email,
		PhoneNumber:  p.PhoneNumber,
		DeviceToken:  p.DeviceToken,
	}
}
