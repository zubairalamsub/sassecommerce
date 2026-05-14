package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ecommerce/user-service/internal/repository"
	sharedstorage "github.com/ecommerce/shared/go/pkg/storage"
	"github.com/sirupsen/logrus"
)

// AvatarService handles user avatar uploads via OCI presigned URLs.
type AvatarService interface {
	PresignUpload(ctx context.Context, tenantID, userID, contentType, filename string) (*AvatarPresignResult, error)
	ConfirmUpload(ctx context.Context, tenantID, userID, imageURL string) error
	Remove(ctx context.Context, tenantID, userID string) error
}

// AvatarPresignResult is the response of PresignUpload.
type AvatarPresignResult struct {
	UploadURL    string            `json:"upload_url"`
	UploadMethod string            `json:"upload_method"`
	Headers      map[string]string `json:"headers,omitempty"`
	ExpiresAt    time.Time         `json:"expires_at"`
	ImageURL     string            `json:"image_url"`
}

type avatarService struct {
	repo    repository.UserRepository
	storage sharedstorage.Client
	logger  *logrus.Logger
}

// NewAvatarService constructs an AvatarService. Returns nil if storage is nil
// (callers should branch on this to keep the routes optional).
func NewAvatarService(repo repository.UserRepository, storage sharedstorage.Client, logger *logrus.Logger) AvatarService {
	return &avatarService{repo: repo, storage: storage, logger: logger}
}

func (s *avatarService) PresignUpload(ctx context.Context, tenantID, userID, contentType, filename string) (*AvatarPresignResult, error) {
	if tenantID == "" || userID == "" {
		return nil, errors.New("tenantID and userID are required")
	}
	ext, err := sharedstorage.DefaultImagePolicy.PickExtension(contentType, filename)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if user.TenantID != tenantID {
		return nil, errors.New("user not found")
	}

	// Use a UUID rather than userID alone so the URL changes on each upload —
	// avoids stale CDN caches showing the previous avatar.
	objectKey := fmt.Sprintf("users/%s/avatar-%d%s", userID, time.Now().UnixNano(), ext)

	pre, err := s.storage.PresignPut(ctx, tenantID, objectKey, contentType, 15*time.Minute)
	if err != nil {
		s.logger.WithError(err).Error("Failed to presign avatar upload")
		return nil, errors.New("failed to issue upload URL")
	}

	return &AvatarPresignResult{
		UploadURL:    pre.URL,
		UploadMethod: pre.Method,
		Headers:      pre.Headers,
		ExpiresAt:    pre.ExpiresAt,
		ImageURL:     s.storage.PublicURL(tenantID, objectKey),
	}, nil
}

func (s *avatarService) ConfirmUpload(ctx context.Context, tenantID, userID, imageURL string) error {
	if tenantID == "" || userID == "" || imageURL == "" {
		return errors.New("tenantID, userID, and imageURL are required")
	}

	objectKey, ok := sharedstorage.KeyFromPublicURL(tenantID, imageURL)
	if !ok {
		return errors.New("imageURL does not belong to this tenant's bucket")
	}

	exists, err := s.storage.Exists(ctx, tenantID, objectKey)
	if err != nil {
		return fmt.Errorf("storage check failed: %w", err)
	}
	if !exists {
		return errors.New("uploaded object not found in storage; please retry the upload")
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}
	if user.TenantID != tenantID {
		return errors.New("user not found")
	}

	// Delete the old avatar from storage if there was one (best-effort).
	if user.Avatar != "" && user.Avatar != imageURL {
		if oldKey, ok := sharedstorage.KeyFromPublicURL(tenantID, user.Avatar); ok {
			if err := s.storage.Delete(ctx, tenantID, oldKey); err != nil {
				s.logger.WithError(err).WithField("key", oldKey).
					Warn("Failed to delete previous avatar object")
			}
		}
	}

	if err := s.repo.UpdateAvatar(ctx, userID, imageURL); err != nil {
		return errors.New("failed to update avatar")
	}
	return nil
}

func (s *avatarService) Remove(ctx context.Context, tenantID, userID string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}
	if user.TenantID != tenantID {
		return errors.New("user not found")
	}

	currentURL := user.Avatar
	if err := s.repo.UpdateAvatar(ctx, userID, ""); err != nil {
		return errors.New("failed to clear avatar")
	}

	if currentURL != "" {
		if oldKey, ok := sharedstorage.KeyFromPublicURL(tenantID, currentURL); ok {
			if err := s.storage.Delete(ctx, tenantID, oldKey); err != nil {
				s.logger.WithError(err).Warn("Failed to delete avatar object; field cleared but orphaned")
			}
		}
	}
	return nil
}
