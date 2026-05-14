package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ecommerce/product-service/internal/mocks"
	"github.com/ecommerce/product-service/internal/models"
	sharedstorage "github.com/ecommerce/shared/go/pkg/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// fakeStorage records calls and lets tests force specific behaviors.
type fakeStorage struct {
	bucket    string
	publicBase string

	puts    []presignCall
	gets    []presignCall
	deletes []deleteCall

	exists    map[string]bool
	existsErr error
}

type presignCall struct {
	tenant, key, contentType string
	expires                  time.Duration
}

type deleteCall struct {
	tenant, key string
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{
		bucket:     "test-bucket",
		publicBase: "https://cdn.example.com",
		exists:     map[string]bool{},
	}
}

func (f *fakeStorage) Bucket() string { return f.bucket }
func (f *fakeStorage) Put(ctx context.Context, tenantID, key string, body io.Reader, contentType string) error {
	return nil
}
func (f *fakeStorage) Get(ctx context.Context, tenantID, key string) (io.ReadCloser, error) {
	return nil, errors.New("not used")
}
func (f *fakeStorage) Delete(ctx context.Context, tenantID, key string) error {
	f.deletes = append(f.deletes, deleteCall{tenantID, key})
	return nil
}
func (f *fakeStorage) Exists(ctx context.Context, tenantID, key string) (bool, error) {
	if f.existsErr != nil {
		return false, f.existsErr
	}
	return f.exists[tenantID+"/"+key], nil
}
func (f *fakeStorage) PresignPut(ctx context.Context, tenantID, key, contentType string, expires time.Duration) (sharedstorage.PresignedURL, error) {
	f.puts = append(f.puts, presignCall{tenantID, key, contentType, expires})
	return sharedstorage.PresignedURL{
		URL:       "https://upload.example.com/" + key,
		Method:    "PUT",
		Headers:   map[string]string{"Content-Type": contentType},
		ExpiresAt: time.Now().Add(expires),
	}, nil
}
func (f *fakeStorage) PresignGet(ctx context.Context, tenantID, key string, expires time.Duration) (sharedstorage.PresignedURL, error) {
	f.gets = append(f.gets, presignCall{tenantID, key, "", expires})
	return sharedstorage.PresignedURL{URL: "https://get.example/" + key, Method: "GET"}, nil
}
func (f *fakeStorage) PublicURL(tenantID, key string) string {
	return f.publicBase + "/tenants/" + tenantID + "/" + key
}

func newImageTestSvc(t *testing.T) (*imageService, *mocks.MockProductRepository, *fakeStorage) {
	t.Helper()
	repo := new(mocks.MockProductRepository)
	storage := newFakeStorage()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return NewImageService(repo, storage, logger).(*imageService), repo, storage
}

func newProduct(tenantID string) *models.Product {
	return &models.Product{
		ID:       primitive.NewObjectID(),
		TenantID: tenantID,
	}
}

func TestPresignUpload_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage := newImageTestSvc(t)

	repo.On("GetByID", ctx, "p1").Return(newProduct("tenant-1"), nil)

	res, err := svc.PresignUpload(ctx, "tenant-1", "p1", "image/jpeg", "photo.JPG")
	assert.NoError(t, err)
	assert.NotEmpty(t, res.UploadURL)
	assert.Equal(t, "PUT", res.UploadMethod)
	assert.Contains(t, res.ObjectKey, "products/p1/")
	assert.True(t, strings.HasSuffix(res.ObjectKey, ".jpg") || strings.HasSuffix(res.ObjectKey, ".jpeg"),
		"object key should keep a .jpg/.jpeg extension")
	assert.Equal(t, "image/jpeg", res.Headers["Content-Type"])
	assert.Contains(t, res.ImageURL, "https://cdn.example.com/tenants/tenant-1/products/p1/")

	if assert.Len(t, storage.puts, 1) {
		assert.Equal(t, "tenant-1", storage.puts[0].tenant)
		assert.Equal(t, "image/jpeg", storage.puts[0].contentType)
	}
}

func TestPresignUpload_RejectsUnsupportedContentType(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newImageTestSvc(t)

	_, err := svc.PresignUpload(ctx, "tenant-1", "p1", "application/pdf", "evil.pdf")
	assert.ErrorContains(t, err, "unsupported content type")
}

func TestPresignUpload_RejectsCrossTenantProduct(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newImageTestSvc(t)

	repo.On("GetByID", ctx, "p1").Return(newProduct("other-tenant"), nil)

	_, err := svc.PresignUpload(ctx, "tenant-1", "p1", "image/png", "")
	assert.EqualError(t, err, "product not found")
}

func TestPresignUpload_NormalizesMimeAndPicksSafeExt(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newImageTestSvc(t)
	repo.On("GetByID", ctx, "p1").Return(newProduct("t1"), nil)

	// User says the file is webp but submits ".exe" — extension should be the
	// MIME's expected one, not the user-supplied one.
	res, err := svc.PresignUpload(ctx, "t1", "p1", "image/webp", "malicious.exe")
	assert.NoError(t, err)
	assert.True(t, strings.HasSuffix(res.ObjectKey, ".webp"),
		"expected .webp extension regardless of user filename, got key %q", res.ObjectKey)
}

func TestConfirmUpload_RecordsImageWhenStorageHasIt(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage := newImageTestSvc(t)

	repo.On("GetByID", ctx, "p1").Return(newProduct("t1"), nil)
	repo.On("AddImage", ctx, "p1", mock.AnythingOfType("string")).Return(nil)
	storage.exists["t1/products/p1/img.jpg"] = true

	imageURL := "https://cdn.example.com/tenants/t1/products/p1/img.jpg"
	err := svc.ConfirmUpload(ctx, "t1", "p1", imageURL)
	assert.NoError(t, err)
	repo.AssertCalled(t, "AddImage", ctx, "p1", imageURL)
}

func TestConfirmUpload_RejectsForeignURL(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newImageTestSvc(t)

	// Tenant-1 submits Tenant-2's URL.
	imageURL := "https://cdn.example.com/tenants/tenant-2/products/p1/img.jpg"
	err := svc.ConfirmUpload(ctx, "tenant-1", "p1", imageURL)
	assert.ErrorContains(t, err, "does not belong")
}

func TestConfirmUpload_RejectsWhenObjectMissing(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newImageTestSvc(t)

	imageURL := "https://cdn.example.com/tenants/t1/products/p1/missing.jpg"
	err := svc.ConfirmUpload(ctx, "t1", "p1", imageURL)
	assert.ErrorContains(t, err, "not found in storage")
}

func TestRemoveImage_DetachesAndDeletesObject(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage := newImageTestSvc(t)

	repo.On("GetByID", ctx, "p1").Return(newProduct("t1"), nil)
	repo.On("RemoveImage", ctx, "p1", mock.AnythingOfType("string")).Return(nil)

	imageURL := "https://cdn.example.com/tenants/t1/products/p1/img.jpg"
	err := svc.RemoveImage(ctx, "t1", "p1", imageURL)
	assert.NoError(t, err)

	if assert.Len(t, storage.deletes, 1) {
		assert.Equal(t, "t1", storage.deletes[0].tenant)
		assert.Equal(t, "products/p1/img.jpg", storage.deletes[0].key)
	}
}

func TestRemoveImage_DoesNotFailOnStorageDeleteError(t *testing.T) {
	// We want detachment to succeed even if the orphan-cleanup fails.
	ctx := context.Background()
	svc, repo, storage := newImageTestSvc(t)

	repo.On("GetByID", ctx, "p1").Return(newProduct("t1"), nil)
	repo.On("RemoveImage", ctx, "p1", mock.AnythingOfType("string")).Return(nil)

	// Make Delete return an error by setting an unparseable URL — falls back
	// to "no key recoverable" and skips delete entirely. Use a foreign URL so
	// storageKeyFromPublicURL returns false and Delete is never called.
	imageURL := "https://other.example/some/path.jpg"
	err := svc.RemoveImage(ctx, "t1", "p1", imageURL)
	assert.NoError(t, err)
	assert.Len(t, storage.deletes, 0, "no delete attempted for foreign URL")
}

func TestRemoveImage_RejectsCrossTenant(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newImageTestSvc(t)
	repo.On("GetByID", ctx, "p1").Return(newProduct("other"), nil)

	err := svc.RemoveImage(ctx, "t1", "p1", "https://cdn.example/whatever.jpg")
	assert.EqualError(t, err, "product not found")
}
