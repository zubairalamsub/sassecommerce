package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	sharedstorage "github.com/ecommerce/shared/go/pkg/storage"
	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// fakeStorage records calls and lets tests force specific behaviors.
// Reused across avatar/branding/review tests via copy — kept local to each
// service so tests stay independent.
type fakeStorage struct {
	puts    []string
	deletes []string
	exists  map[string]bool
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{exists: map[string]bool{}}
}

func (f *fakeStorage) Bucket() string                                                    { return "test" }
func (f *fakeStorage) Put(ctx context.Context, t, k string, b io.Reader, c string) error { return nil }
func (f *fakeStorage) Get(ctx context.Context, t, k string) (io.ReadCloser, error) {
	return nil, errors.New("nope")
}
func (f *fakeStorage) Delete(ctx context.Context, t, k string) error {
	f.deletes = append(f.deletes, t+"/"+k)
	return nil
}
func (f *fakeStorage) Exists(ctx context.Context, t, k string) (bool, error) {
	return f.exists[t+"/"+k], nil
}
func (f *fakeStorage) PresignPut(ctx context.Context, t, k, c string, e time.Duration) (sharedstorage.PresignedURL, error) {
	f.puts = append(f.puts, t+"/"+k)
	return sharedstorage.PresignedURL{
		URL: "https://upload.example.com/" + k, Method: "PUT",
		Headers: map[string]string{"Content-Type": c}, ExpiresAt: time.Now().Add(e),
	}, nil
}
func (f *fakeStorage) PresignGet(ctx context.Context, t, k string, e time.Duration) (sharedstorage.PresignedURL, error) {
	return sharedstorage.PresignedURL{}, nil
}
func (f *fakeStorage) PublicURL(t, k string) string {
	return "https://cdn.example.com/tenants/" + t + "/" + k
}

func newAvatarTestSvc(t *testing.T) (*avatarService, *mocks.MockUserRepository, *fakeStorage) {
	t.Helper()
	repo := new(mocks.MockUserRepository)
	storage := newFakeStorage()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return NewAvatarService(repo, storage, logger).(*avatarService), repo, storage
}

func TestAvatarPresign_BuildsScopedKeyAndCDNURL(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newAvatarTestSvc(t)
	repo.On("GetByID", ctx, "u1").Return(&models.User{ID: "u1", TenantID: "t1"}, nil)

	res, err := svc.PresignUpload(ctx, "t1", "u1", "image/png", "selfie.PNG")
	assert.NoError(t, err)
	assert.NotEmpty(t, res.UploadURL)
	assert.Equal(t, "PUT", res.UploadMethod)
	assert.Equal(t, "image/png", res.Headers["Content-Type"])
	assert.Contains(t, res.ImageURL, "https://cdn.example.com/tenants/t1/users/u1/avatar-")
	assert.True(t, strings.HasSuffix(res.ImageURL, ".png") || strings.HasSuffix(res.ImageURL, ".PNG") || strings.HasSuffix(res.ImageURL, ".jpg"))
}

func TestAvatarPresign_RejectsCrossTenantUser(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newAvatarTestSvc(t)
	repo.On("GetByID", ctx, "u1").Return(&models.User{ID: "u1", TenantID: "other"}, nil)

	_, err := svc.PresignUpload(ctx, "t1", "u1", "image/png", "")
	assert.EqualError(t, err, "user not found")
}

func TestAvatarPresign_RejectsBadMime(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newAvatarTestSvc(t)
	_, err := svc.PresignUpload(ctx, "t1", "u1", "application/zip", "bad.zip")
	assert.ErrorContains(t, err, "unsupported content type")
}

func TestAvatarConfirm_DeletesPreviousAvatarObject(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage := newAvatarTestSvc(t)

	previous := "https://cdn.example.com/tenants/t1/users/u1/avatar-old.jpg"
	current := "https://cdn.example.com/tenants/t1/users/u1/avatar-new.jpg"

	repo.On("GetByID", ctx, "u1").Return(&models.User{ID: "u1", TenantID: "t1", Avatar: previous}, nil)
	repo.On("UpdateAvatar", ctx, "u1", current).Return(nil)
	storage.exists["t1/users/u1/avatar-new.jpg"] = true

	err := svc.ConfirmUpload(ctx, "t1", "u1", current)
	assert.NoError(t, err)

	// Old avatar object should have been deleted.
	if assert.Len(t, storage.deletes, 1) {
		assert.Equal(t, "t1/users/u1/avatar-old.jpg", storage.deletes[0])
	}
}

func TestAvatarConfirm_RejectsForeignURL(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newAvatarTestSvc(t)
	err := svc.ConfirmUpload(ctx, "t1", "u1", "https://cdn.example.com/tenants/other/users/u1/avatar.jpg")
	assert.ErrorContains(t, err, "does not belong")
}

func TestAvatarConfirm_RejectsMissingObject(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newAvatarTestSvc(t)
	err := svc.ConfirmUpload(ctx, "t1", "u1", "https://cdn.example.com/tenants/t1/users/u1/avatar-missing.jpg")
	assert.ErrorContains(t, err, "not found in storage")
}

func TestAvatarRemove_ClearsAndDeletes(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage := newAvatarTestSvc(t)

	url := "https://cdn.example.com/tenants/t1/users/u1/avatar.jpg"
	repo.On("GetByID", ctx, "u1").Return(&models.User{ID: "u1", TenantID: "t1", Avatar: url}, nil)
	repo.On("UpdateAvatar", ctx, "u1", "").Return(nil)

	err := svc.Remove(ctx, "t1", "u1")
	assert.NoError(t, err)
	assert.Len(t, storage.deletes, 1)
}

func TestAvatarRemove_NoOpIfNotSet(t *testing.T) {
	ctx := context.Background()
	svc, repo, storage := newAvatarTestSvc(t)
	repo.On("GetByID", ctx, "u1").Return(&models.User{ID: "u1", TenantID: "t1"}, nil)
	repo.On("UpdateAvatar", ctx, "u1", "").Return(nil)

	assert.NoError(t, svc.Remove(ctx, "t1", "u1"))
	assert.Len(t, storage.deletes, 0)
}

// Compile-time interface satisfaction check.
var _ AvatarService = (*avatarService)(nil)
var _ = mock.Anything
