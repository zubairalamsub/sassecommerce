package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	sharedstorage "github.com/ecommerce/shared/go/pkg/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type fakeStorage struct {
	deletes []string
	puts    []string
}

func (f *fakeStorage) Bucket() string                                                        { return "test" }
func (f *fakeStorage) Put(ctx context.Context, t, k string, b io.Reader, c string) error      { return nil }
func (f *fakeStorage) Get(ctx context.Context, t, k string) (io.ReadCloser, error) {
	return nil, errors.New("nope")
}
func (f *fakeStorage) Delete(ctx context.Context, t, k string) error {
	f.deletes = append(f.deletes, t+"/"+k)
	return nil
}
func (f *fakeStorage) Exists(ctx context.Context, t, k string) (bool, error) { return false, nil }
func (f *fakeStorage) PresignPut(ctx context.Context, t, k, c string, e time.Duration) (sharedstorage.PresignedURL, error) {
	f.puts = append(f.puts, t+"/"+k)
	return sharedstorage.PresignedURL{URL: "https://upload/" + k, Method: "PUT", Headers: map[string]string{"Content-Type": c}, ExpiresAt: time.Now().Add(e)}, nil
}
func (f *fakeStorage) PresignGet(ctx context.Context, t, k string, e time.Duration) (sharedstorage.PresignedURL, error) {
	return sharedstorage.PresignedURL{}, nil
}
func (f *fakeStorage) PublicURL(t, k string) string {
	return "https://cdn.example.com/tenants/" + t + "/" + k
}

func newSvc() (*brandingService, *fakeStorage) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	storage := &fakeStorage{}
	return NewBrandingService(storage, logger).(*brandingService), storage
}

func TestPresignBranding_BuildsKindScopedKey(t *testing.T) {
	svc, _ := newSvc()
	res, err := svc.PresignUpload(context.Background(), "t1", KindLogo, "image/png", "logo.PNG")
	assert.NoError(t, err)
	assert.Contains(t, res.ImageURL, "/tenants/t1/branding/logo-")
	assert.True(t, strings.HasSuffix(res.ImageURL, ".png") || strings.HasSuffix(res.ImageURL, ".PNG"))
	assert.Equal(t, "image/png", res.Headers["Content-Type"])
}

func TestPresignBranding_AcceptsSVGForLogo(t *testing.T) {
	svc, _ := newSvc()
	res, err := svc.PresignUpload(context.Background(), "t1", KindLogo, "image/svg+xml", "logo.svg")
	assert.NoError(t, err)
	assert.True(t, strings.HasSuffix(res.ImageURL, ".svg"))
}

func TestPresignBranding_RejectsUnknownKind(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.PresignUpload(context.Background(), "t1", BrandingKind("hero-video"), "image/png", "")
	assert.ErrorContains(t, err, "unknown branding kind")
}

func TestPresignBranding_RejectsBadMime(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.PresignUpload(context.Background(), "t1", KindLogo, "application/zip", "logo.zip")
	assert.ErrorContains(t, err, "unsupported content type")
}

func TestRemoveBrandingAsset_DeletesObject(t *testing.T) {
	svc, storage := newSvc()
	url := "https://cdn.example.com/tenants/t1/branding/logo-123.png"
	assert.NoError(t, svc.RemoveAsset(context.Background(), "t1", url))
	assert.Equal(t, []string{"t1/branding/logo-123.png"}, storage.deletes)
}

func TestRemoveBrandingAsset_RejectsForeignURL(t *testing.T) {
	svc, _ := newSvc()
	err := svc.RemoveAsset(context.Background(), "t1", "https://cdn.example.com/tenants/other/branding/logo.png")
	assert.ErrorContains(t, err, "does not belong")
}

var _ BrandingService = (*brandingService)(nil)
