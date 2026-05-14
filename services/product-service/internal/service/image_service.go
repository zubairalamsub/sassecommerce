package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/ecommerce/product-service/internal/repository"
	sharedstorage "github.com/ecommerce/shared/go/pkg/storage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ImageService handles product image uploads via presigned URLs.
type ImageService interface {
	// PresignUpload returns a presigned PUT URL the browser can use to upload
	// an image directly to object storage. The returned imageURL is the public
	// URL the client should call ConfirmUpload with after the PUT succeeds.
	PresignUpload(ctx context.Context, tenantID, productID, contentType, filename string) (*PresignUploadResult, error)

	// ConfirmUpload records a successfully uploaded image on the product.
	// It verifies the object actually exists in storage before persisting.
	ConfirmUpload(ctx context.Context, tenantID, productID, imageURL string) error

	// RemoveImage removes an image from the product and deletes the object.
	RemoveImage(ctx context.Context, tenantID, productID, imageURL string) error
}

// PresignUploadResult is the response of PresignUpload.
type PresignUploadResult struct {
	UploadURL    string            `json:"upload_url"`
	UploadMethod string            `json:"upload_method"`
	Headers      map[string]string `json:"headers,omitempty"`
	ExpiresAt    time.Time         `json:"expires_at"`
	// ObjectKey is the storage key (relative to the tenant prefix). Persisted
	// later via ConfirmUpload.
	ObjectKey string `json:"object_key"`
	// ImageURL is the public URL the storefront should display once uploaded.
	// The client should pass this back to ConfirmUpload verbatim.
	ImageURL string `json:"image_url"`
}

type imageService struct {
	repo    repository.ProductRepository
	storage sharedstorage.Client
	logger  *logrus.Logger
}

// NewImageService constructs an ImageService.
func NewImageService(repo repository.ProductRepository, storage sharedstorage.Client, logger *logrus.Logger) ImageService {
	return &imageService{repo: repo, storage: storage, logger: logger}
}

// allowedImageTypes restricts uploads to common web image MIME types so
// presigned URLs can't be abused for arbitrary file hosting.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
	"image/avif": ".avif",
}

func (s *imageService) PresignUpload(ctx context.Context, tenantID, productID, contentType, filename string) (*PresignUploadResult, error) {
	if tenantID == "" || productID == "" {
		return nil, errors.New("tenantID and productID are required")
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	ext, ok := allowedImageTypes[contentType]
	if !ok {
		return nil, fmt.Errorf("unsupported content type: %s (allowed: jpeg, png, webp, gif, avif)", contentType)
	}

	// Confirm the product exists and belongs to this tenant before issuing
	// a presigned URL — otherwise a tenant could mint URLs against another
	// tenant's product IDs (the storage prefix would still isolate them, but
	// failing fast surfaces the bug to the caller).
	product, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		return nil, errors.New("product not found")
	}
	if product.TenantID != tenantID {
		return nil, errors.New("product not found")
	}

	// Use an extension preferred by the MIME type rather than the user-supplied
	// filename — defends against ".jpg" filenames hiding executables.
	safeExt := ext
	if filename != "" {
		userExt := strings.ToLower(path.Ext(filename))
		if userExt == ".jpeg" || userExt == ".jpg" || userExt == ".png" ||
			userExt == ".webp" || userExt == ".gif" || userExt == ".avif" {
			safeExt = userExt
		}
	}

	objectKey := fmt.Sprintf("products/%s/%s%s", productID, uuid.New().String(), safeExt)

	pre, err := s.storage.PresignPut(ctx, tenantID, objectKey, contentType, 15*time.Minute)
	if err != nil {
		s.logger.WithError(err).Error("Failed to presign upload")
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

func (s *imageService) ConfirmUpload(ctx context.Context, tenantID, productID, imageURL string) error {
	if tenantID == "" || productID == "" || imageURL == "" {
		return errors.New("tenantID, productID, and imageURL are required")
	}

	// Recover the object key from the public URL so we can verify it exists.
	objectKey, ok := s.storageKeyFromPublicURL(tenantID, imageURL)
	if !ok {
		return errors.New("imageURL does not belong to this tenant's product bucket")
	}

	exists, err := s.storage.Exists(ctx, tenantID, objectKey)
	if err != nil {
		return fmt.Errorf("storage check failed: %w", err)
	}
	if !exists {
		return errors.New("uploaded object not found in storage; please retry the upload")
	}

	product, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		return errors.New("product not found")
	}
	if product.TenantID != tenantID {
		return errors.New("product not found")
	}

	if err := s.repo.AddImage(ctx, productID, imageURL); err != nil {
		s.logger.WithError(err).Error("Failed to record image on product")
		return errors.New("failed to record image")
	}
	return nil
}

func (s *imageService) RemoveImage(ctx context.Context, tenantID, productID, imageURL string) error {
	if tenantID == "" || productID == "" || imageURL == "" {
		return errors.New("tenantID, productID, and imageURL are required")
	}

	product, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		return errors.New("product not found")
	}
	if product.TenantID != tenantID {
		return errors.New("product not found")
	}

	if err := s.repo.RemoveImage(ctx, productID, imageURL); err != nil {
		return err
	}

	// Best-effort delete from storage. Don't fail the request on cleanup
	// errors — the image is already detached from the product.
	if objectKey, ok := s.storageKeyFromPublicURL(tenantID, imageURL); ok {
		if err := s.storage.Delete(ctx, tenantID, objectKey); err != nil {
			s.logger.WithError(err).WithField("key", objectKey).
				Warn("Failed to delete object from storage; image is detached but orphaned")
		}
	}
	return nil
}

// storageKeyFromPublicURL recovers the tenant-relative key from a public URL
// that the storage client originally issued. Returns false if the URL doesn't
// reference this tenant's prefix — defends against confused-deputy where one
// tenant submits another tenant's URL.
func (s *imageService) storageKeyFromPublicURL(tenantID, publicURL string) (string, bool) {
	prefix := fmt.Sprintf("tenants/%s/", tenantID)
	idx := strings.Index(publicURL, prefix)
	if idx < 0 {
		return "", false
	}
	return publicURL[idx+len(prefix):], true
}
