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
	puts, deletes []string
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

func newSvc() (*attachmentService, *fakeStorage) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	storage := &fakeStorage{}
	return NewAttachmentService(storage, logger).(*attachmentService), storage
}

func TestPresignAttachment_BuildsUserScopedKey(t *testing.T) {
	svc, _ := newSvc()
	res, err := svc.PresignUpload(context.Background(), "t1", "u42", "image/jpeg", "photo.JPG")
	assert.NoError(t, err)
	assert.Contains(t, res.ImageURL, "/tenants/t1/reviews/users/u42/")
	assert.True(t, strings.HasSuffix(res.ImageURL, ".jpg") || strings.HasSuffix(res.ImageURL, ".JPG"))
}

func TestPresignAttachment_RejectsBadMime(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.PresignUpload(context.Background(), "t1", "u1", "application/javascript", "evil.js")
	assert.ErrorContains(t, err, "unsupported content type")
}

func TestPresignAttachment_RequiresIDs(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.PresignUpload(context.Background(), "", "u1", "image/png", "")
	assert.Error(t, err)
	_, err = svc.PresignUpload(context.Background(), "t1", "", "image/png", "")
	assert.Error(t, err)
}

func TestRemoveAttachment_DeletesObject(t *testing.T) {
	svc, storage := newSvc()
	url := "https://cdn.example.com/tenants/t1/reviews/users/u42/abc.jpg"
	assert.NoError(t, svc.Remove(context.Background(), "t1", url))
	assert.Equal(t, []string{"t1/reviews/users/u42/abc.jpg"}, storage.deletes)
}

func TestRemoveAttachment_RejectsForeign(t *testing.T) {
	svc, _ := newSvc()
	err := svc.Remove(context.Background(), "t1", "https://cdn.example.com/tenants/other/reviews/users/u42/abc.jpg")
	assert.ErrorContains(t, err, "does not belong")
}

var _ AttachmentService = (*attachmentService)(nil)
