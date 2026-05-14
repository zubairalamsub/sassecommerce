package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	sharedstorage "github.com/ecommerce/shared/go/pkg/storage"
	"github.com/sirupsen/logrus"
)

// BrandingKind identifies which slot the upload is for. Each kind has its own
// path so admins can swap a logo without affecting the favicon.
type BrandingKind string

const (
	KindLogo    BrandingKind = "logo"
	KindFavicon BrandingKind = "favicon"
	KindBanner  BrandingKind = "banner"
	KindPromoBackground BrandingKind = "promo-background"
)

func (k BrandingKind) IsValid() bool {
	switch k {
	case KindLogo, KindFavicon, KindBanner, KindPromoBackground:
		return true
	}
	return false
}

// BrandingService presigns upload URLs for tenant branding assets and cleans
// them up. It does NOT update the tenant's config — that's the caller's job
// via the existing PATCH /tenants/:id/config endpoint.
type BrandingService interface {
	PresignUpload(ctx context.Context, tenantID string, kind BrandingKind, contentType, filename string) (*BrandingPresignResult, error)
	RemoveAsset(ctx context.Context, tenantID, imageURL string) error
}

// BrandingPresignResult is the response of PresignUpload.
type BrandingPresignResult struct {
	UploadURL    string            `json:"upload_url"`
	UploadMethod string            `json:"upload_method"`
	Headers      map[string]string `json:"headers,omitempty"`
	ExpiresAt    time.Time         `json:"expires_at"`
	ImageURL     string            `json:"image_url"`
}

type brandingService struct {
	storage sharedstorage.Client
	logger  *logrus.Logger
}

// NewBrandingService constructs a BrandingService.
func NewBrandingService(storage sharedstorage.Client, logger *logrus.Logger) BrandingService {
	return &brandingService{storage: storage, logger: logger}
}

func (s *brandingService) PresignUpload(ctx context.Context, tenantID string, kind BrandingKind, contentType, filename string) (*BrandingPresignResult, error) {
	if tenantID == "" {
		return nil, errors.New("tenantID is required")
	}
	if !kind.IsValid() {
		return nil, fmt.Errorf("unknown branding kind: %s", kind)
	}
	ext, err := sharedstorage.BrandingImagePolicy.PickExtension(contentType, filename)
	if err != nil {
		return nil, err
	}

	// Banner uploads are versioned by timestamp because tenants may have many
	// banner slides. Logo/favicon use a stable name with a timestamp suffix to
	// bust CDN caches when re-uploaded.
	objectKey := fmt.Sprintf("branding/%s-%d%s", kind, time.Now().UnixNano(), ext)

	pre, err := s.storage.PresignPut(ctx, tenantID, objectKey, contentType, 15*time.Minute)
	if err != nil {
		s.logger.WithError(err).Error("Failed to presign branding upload")
		return nil, errors.New("failed to issue upload URL")
	}

	return &BrandingPresignResult{
		UploadURL:    pre.URL,
		UploadMethod: pre.Method,
		Headers:      pre.Headers,
		ExpiresAt:    pre.ExpiresAt,
		ImageURL:     s.storage.PublicURL(tenantID, objectKey),
	}, nil
}

func (s *brandingService) RemoveAsset(ctx context.Context, tenantID, imageURL string) error {
	if tenantID == "" || imageURL == "" {
		return errors.New("tenantID and imageURL are required")
	}
	objectKey, ok := sharedstorage.KeyFromPublicURL(tenantID, imageURL)
	if !ok {
		return errors.New("imageURL does not belong to this tenant's bucket")
	}
	return s.storage.Delete(ctx, tenantID, objectKey)
}
