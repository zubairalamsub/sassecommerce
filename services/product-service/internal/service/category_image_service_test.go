package service

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/ecommerce/product-service/internal/mocks"
	"github.com/ecommerce/product-service/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func newCategoryImageTestSvc(t *testing.T) (*categoryImageService, *mocks.MockCategoryRepository, *fakeStorage) {
	t.Helper()
	repo := new(mocks.MockCategoryRepository)
	storage := newFakeStorage()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return NewCategoryImageService(repo, storage, logger).(*categoryImageService), repo, storage
}

func newCategory(tenantID string, image string) *models.Category {
	return &models.Category{
		ID:       primitive.NewObjectID(),
		TenantID: tenantID,
		Image:    image,
	}
}

func TestCategoryPresignUpload_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage := newCategoryImageTestSvc(t)

	repo.On("GetByID", ctx, "c1").Return(newCategory("tenant-1", ""), nil)

	res, err := svc.PresignUpload(ctx, "tenant-1", "c1", "image/jpeg", "hero.JPG")
	assert.NoError(t, err)
	assert.NotEmpty(t, res.UploadURL)
	assert.Equal(t, "PUT", res.UploadMethod)
	assert.Contains(t, res.ObjectKey, "categories/c1/")
	assert.True(t,
		strings.HasSuffix(res.ObjectKey, ".jpg") || strings.HasSuffix(res.ObjectKey, ".jpeg"),
		"object key should keep a .jpg/.jpeg extension, got %q", res.ObjectKey)
	assert.Equal(t, "image/jpeg", res.Headers["Content-Type"])
	assert.Contains(t, res.ImageURL, "https://cdn.example.com/tenants/tenant-1/categories/c1/")

	if assert.Len(t, storage.puts, 1) {
		assert.Equal(t, "tenant-1", storage.puts[0].tenant)
		assert.Equal(t, "image/jpeg", storage.puts[0].contentType)
	}
}

func TestCategoryPresignUpload_RejectsUnsupportedContentType(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newCategoryImageTestSvc(t)

	_, err := svc.PresignUpload(ctx, "tenant-1", "c1", "application/pdf", "evil.pdf")
	assert.ErrorContains(t, err, "unsupported content type")
}

func TestCategoryPresignUpload_RejectsCrossTenant(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newCategoryImageTestSvc(t)

	repo.On("GetByID", ctx, "c1").Return(newCategory("other-tenant", ""), nil)

	_, err := svc.PresignUpload(ctx, "tenant-1", "c1", "image/png", "")
	assert.EqualError(t, err, "category not found")
}

func TestCategoryPresignUpload_NormalizesMimeAndPicksSafeExt(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newCategoryImageTestSvc(t)
	repo.On("GetByID", ctx, "c1").Return(newCategory("t1", ""), nil)

	// User says it's webp but filename is .exe — extension should match MIME.
	res, err := svc.PresignUpload(ctx, "t1", "c1", "image/webp", "malicious.exe")
	assert.NoError(t, err)
	assert.True(t, strings.HasSuffix(res.ObjectKey, ".webp"),
		"expected .webp extension regardless of user filename, got %q", res.ObjectKey)
}

func TestCategoryConfirmUpload_RecordsImageWhenStorageHasIt(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage := newCategoryImageTestSvc(t)

	repo.On("GetByID", ctx, "c1").Return(newCategory("t1", ""), nil)
	repo.On("UpdateImage", ctx, "c1", mock.AnythingOfType("string")).Return(nil)
	storage.exists["t1/categories/c1/img.jpg"] = true

	imageURL := "https://cdn.example.com/tenants/t1/categories/c1/img.jpg"
	err := svc.ConfirmUpload(ctx, "t1", "c1", imageURL)
	assert.NoError(t, err)
	repo.AssertCalled(t, "UpdateImage", ctx, "c1", imageURL)
}

func TestCategoryConfirmUpload_RejectsForeignURL(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newCategoryImageTestSvc(t)

	// Tenant-1 submits Tenant-2's URL.
	imageURL := "https://cdn.example.com/tenants/tenant-2/categories/c1/img.jpg"
	err := svc.ConfirmUpload(ctx, "tenant-1", "c1", imageURL)
	assert.ErrorContains(t, err, "does not belong")
}

func TestCategoryConfirmUpload_RejectsWhenObjectMissing(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newCategoryImageTestSvc(t)

	// storage.exists not set → false
	imageURL := "https://cdn.example.com/tenants/t1/categories/c1/missing.jpg"
	err := svc.ConfirmUpload(ctx, "t1", "c1", imageURL)
	assert.ErrorContains(t, err, "not found in storage")
}

func TestCategoryConfirmUpload_RejectsCrossTenantCategory(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage := newCategoryImageTestSvc(t)

	// URL prefix passes the tenant check, but the category itself belongs to
	// another tenant — defends against a tenant submitting an URL that they
	// somehow obtained but for a category they don't own.
	repo.On("GetByID", ctx, "c1").Return(newCategory("other-tenant", ""), nil)
	storage.exists["t1/categories/c1/img.jpg"] = true

	imageURL := "https://cdn.example.com/tenants/t1/categories/c1/img.jpg"
	err := svc.ConfirmUpload(ctx, "t1", "c1", imageURL)
	assert.EqualError(t, err, "category not found")
}

func TestCategoryConfirmUpload_ReplacesAndDeletesPrevious(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage := newCategoryImageTestSvc(t)

	// Existing image already set — uploading a new one should detach the old.
	existing := newCategory("t1", "https://cdn.example.com/tenants/t1/categories/c1/old.jpg")
	repo.On("GetByID", ctx, "c1").Return(existing, nil)
	repo.On("UpdateImage", ctx, "c1", mock.AnythingOfType("string")).Return(nil)
	storage.exists["t1/categories/c1/new.jpg"] = true

	imageURL := "https://cdn.example.com/tenants/t1/categories/c1/new.jpg"
	err := svc.ConfirmUpload(ctx, "t1", "c1", imageURL)
	assert.NoError(t, err)
	if assert.Len(t, storage.deletes, 1) {
		assert.Equal(t, "categories/c1/old.jpg", storage.deletes[0].key)
	}
}

func TestCategoryRemoveImage_ClearsAndDeletesObject(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage := newCategoryImageTestSvc(t)

	existing := newCategory("t1", "https://cdn.example.com/tenants/t1/categories/c1/img.jpg")
	repo.On("GetByID", ctx, "c1").Return(existing, nil)
	repo.On("UpdateImage", ctx, "c1", "").Return(nil)

	err := svc.RemoveImage(ctx, "t1", "c1")
	assert.NoError(t, err)

	if assert.Len(t, storage.deletes, 1) {
		assert.Equal(t, "t1", storage.deletes[0].tenant)
		assert.Equal(t, "categories/c1/img.jpg", storage.deletes[0].key)
	}
	repo.AssertCalled(t, "UpdateImage", ctx, "c1", "")
}

func TestCategoryRemoveImage_NoOpWhenAlreadyEmpty(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage := newCategoryImageTestSvc(t)

	// No image set — clearing the field still succeeds; nothing to delete.
	repo.On("GetByID", ctx, "c1").Return(newCategory("t1", ""), nil)
	repo.On("UpdateImage", ctx, "c1", "").Return(nil)

	err := svc.RemoveImage(ctx, "t1", "c1")
	assert.NoError(t, err)
	assert.Len(t, storage.deletes, 0, "no delete attempted for empty image")
}

func TestCategoryRemoveImage_RejectsCrossTenant(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newCategoryImageTestSvc(t)
	repo.On("GetByID", ctx, "c1").Return(newCategory("other", ""), nil)

	err := svc.RemoveImage(ctx, "t1", "c1")
	assert.EqualError(t, err, "category not found")
}
