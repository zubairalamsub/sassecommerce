package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ecommerce/product-service/internal/repository"
	sharedstorage "github.com/ecommerce/shared/go/pkg/storage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// CategoryImageService handles category image uploads via presigned URLs.
// Mirrors ImageService (the product image service) but stores a single image
// directly on the category document instead of appending to an array.
type CategoryImageService interface {
	// PresignUpload returns a presigned PUT URL the browser can use to upload
	// the category's hero image directly to object storage. The returned
	// imageURL is the public URL the client should call ConfirmUpload with
	// after the PUT succeeds.
	PresignUpload(ctx context.Context, tenantID, categoryID, contentType, filename string) (*PresignUploadResult, error)

	// ConfirmUpload records a successfully uploaded image on the category.
	// It verifies the object actually exists in storage before persisting.
	ConfirmUpload(ctx context.Context, tenantID, categoryID, imageURL string) error

	// RemoveImage clears the image field on the category and best-effort
	// deletes the underlying object from storage.
	RemoveImage(ctx context.Context, tenantID, categoryID string) error
}

type categoryImageService struct {
	repo    repository.CategoryRepository
	storage sharedstorage.Client
	logger  *logrus.Logger
}

// NewCategoryImageService constructs a CategoryImageService.
func NewCategoryImageService(repo repository.CategoryRepository, storage sharedstorage.Client, logger *logrus.Logger) CategoryImageService {
	return &categoryImageService{repo: repo, storage: storage, logger: logger}
}

func (s *categoryImageService) PresignUpload(ctx context.Context, tenantID, categoryID, contentType, filename string) (*PresignUploadResult, error) {
	if tenantID == "" || categoryID == "" {
		return nil, errors.New("tenantID and categoryID are required")
	}

	ext, err := sharedstorage.DefaultImagePolicy.PickExtension(contentType, filename)
	if err != nil {
		return nil, err
	}

	// Confirm the category exists and belongs to this tenant before issuing
	// a presigned URL — mirrors the product image flow's cross-tenant guard.
	category, err := s.repo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, errors.New("category not found")
	}
	if category.TenantID != tenantID {
		return nil, errors.New("category not found")
	}

	objectKey := fmt.Sprintf("categories/%s/%s%s", categoryID, uuid.New().String(), ext)

	pre, err := s.storage.PresignPut(ctx, tenantID, objectKey, contentType, 15*time.Minute)
	if err != nil {
		s.logger.WithError(err).Error("Failed to presign category image upload")
		return nil, errors.New("failed to issue upload URL")
	}

	publicURL := s.storage.PublicURL(tenantID, objectKey)

	return &PresignUploadResult{
		UploadURL:    pre.URL,
		UploadMethod: pre.Method,
		Headers:      pre.Headers,
		ExpiresAt:    pre.ExpiresAt,
		ObjectKey:    objectKey,
		ImageURL:     publicURL,
	}, nil
}

func (s *categoryImageService) ConfirmUpload(ctx context.Context, tenantID, categoryID, imageURL string) error {
	if tenantID == "" || categoryID == "" || imageURL == "" {
		return errors.New("tenantID, categoryID, and imageURL are required")
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

	category, err := s.repo.GetByID(ctx, categoryID)
	if err != nil {
		return errors.New("category not found")
	}
	if category.TenantID != tenantID {
		return errors.New("category not found")
	}

	// If the category already had an image, best-effort delete the previous
	// object so we don't pile up orphans on every replace.
	previousURL := category.Image

	if err := s.repo.UpdateImage(ctx, categoryID, imageURL); err != nil {
		s.logger.WithError(err).Error("Failed to record image on category")
		return errors.New("failed to record image")
	}

	if previousURL != "" && previousURL != imageURL {
		if prevKey, ok := sharedstorage.KeyFromPublicURL(tenantID, previousURL); ok {
			if err := s.storage.Delete(ctx, tenantID, prevKey); err != nil {
				s.logger.WithError(err).WithField("key", prevKey).
					Warn("Failed to delete previous category image; new image is set but old one is orphaned")
			}
		}
	}

	return nil
}

func (s *categoryImageService) RemoveImage(ctx context.Context, tenantID, categoryID string) error {
	if tenantID == "" || categoryID == "" {
		return errors.New("tenantID and categoryID are required")
	}

	category, err := s.repo.GetByID(ctx, categoryID)
	if err != nil {
		return errors.New("category not found")
	}
	if category.TenantID != tenantID {
		return errors.New("category not found")
	}

	previousURL := category.Image
	if err := s.repo.UpdateImage(ctx, categoryID, ""); err != nil {
		return err
	}

	// Best-effort delete from storage. Don't fail the request on cleanup
	// errors — the image is already detached from the category.
	if previousURL != "" {
		if objectKey, ok := sharedstorage.KeyFromPublicURL(tenantID, previousURL); ok {
			if err := s.storage.Delete(ctx, tenantID, objectKey); err != nil {
				s.logger.WithError(err).WithField("key", objectKey).
					Warn("Failed to delete category image from storage; image is detached but orphaned")
			}
		}
	}
	return nil
}
