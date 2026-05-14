package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	sharedstorage "github.com/ecommerce/shared/go/pkg/storage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AttachmentService presigns upload URLs for review attachment images.
// Uploads are scoped to the customer who's writing the review; the URLs are
// then submitted as part of the review body to the existing CreateReview API.
type AttachmentService interface {
	PresignUpload(ctx context.Context, tenantID, userID, contentType, filename string) (*AttachmentPresignResult, error)
	Remove(ctx context.Context, tenantID, imageURL string) error
}

// AttachmentPresignResult is the response of PresignUpload.
type AttachmentPresignResult struct {
	UploadURL    string            `json:"upload_url"`
	UploadMethod string            `json:"upload_method"`
	Headers      map[string]string `json:"headers,omitempty"`
	ExpiresAt    time.Time         `json:"expires_at"`
	ImageURL     string            `json:"image_url"`
}

type attachmentService struct {
	storage sharedstorage.Client
	logger  *logrus.Logger
}

// NewAttachmentService constructs an AttachmentService.
func NewAttachmentService(storage sharedstorage.Client, logger *logrus.Logger) AttachmentService {
	return &attachmentService{storage: storage, logger: logger}
}

func (s *attachmentService) PresignUpload(ctx context.Context, tenantID, userID, contentType, filename string) (*AttachmentPresignResult, error) {
	if tenantID == "" || userID == "" {
		return nil, errors.New("tenantID and userID are required")
	}
	ext, err := sharedstorage.DefaultImagePolicy.PickExtension(contentType, filename)
	if err != nil {
		return nil, err
	}

	// Bucket review attachments by user so a user can browse/clean up their
	// own uploads, and so admins can spot-check by user when moderating.
	objectKey := fmt.Sprintf("reviews/users/%s/%s%s", userID, uuid.New().String(), ext)

	pre, err := s.storage.PresignPut(ctx, tenantID, objectKey, contentType, 15*time.Minute)
	if err != nil {
		s.logger.WithError(err).Error("Failed to presign review attachment upload")
		return nil, errors.New("failed to issue upload URL")
	}
	return &AttachmentPresignResult{
		UploadURL:    pre.URL,
		UploadMethod: pre.Method,
		Headers:      pre.Headers,
		ExpiresAt:    pre.ExpiresAt,
		ImageURL:     s.storage.PublicURL(tenantID, objectKey),
	}, nil
}

func (s *attachmentService) Remove(ctx context.Context, tenantID, imageURL string) error {
	if tenantID == "" || imageURL == "" {
		return errors.New("tenantID and imageURL are required")
	}
	objectKey, ok := sharedstorage.KeyFromPublicURL(tenantID, imageURL)
	if !ok {
		return errors.New("imageURL does not belong to this tenant's bucket")
	}
	return s.storage.Delete(ctx, tenantID, objectKey)
}
