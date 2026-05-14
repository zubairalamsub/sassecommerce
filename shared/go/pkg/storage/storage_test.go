package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// stubS3Server is a minimal HTTP server that implements just enough of the
// S3 API to exercise the storage client end-to-end (Put, Get, Head, Delete).
// We use this rather than mocking the AWS SDK because that gives us real
// signature/path-style coverage too.
type stubS3Server struct {
	mu      sync.Mutex
	objects map[string][]byte
	mime    map[string]string
	server  *httptest.Server
}

func newStubS3Server(t *testing.T) *stubS3Server {
	t.Helper()
	s := &stubS3Server{
		objects: map[string][]byte{},
		mime:    map[string]string{},
	}
	s.server = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.server.Close)
	return s
}

func (s *stubS3Server) handle(w http.ResponseWriter, r *http.Request) {
	// Path-style URLs look like /bucket/keyparts...
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)
	if len(parts) < 2 {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	key := parts[1]

	s.mu.Lock()
	defer s.mu.Unlock()

	switch r.Method {
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		s.objects[key] = body
		s.mime[key] = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		body, ok := s.objects[key]
		if !ok {
			http.Error(w, "<Error><Code>NoSuchKey</Code></Error>", http.StatusNotFound)
			return
		}
		if ct := s.mime[key]; ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		_, _ = w.Write(body)
	case http.MethodHead:
		if _, ok := s.objects[key]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		delete(s.objects, key)
		delete(s.mime, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func newTestClient(t *testing.T, srv *stubS3Server) Client {
	t.Helper()
	c, err := New(context.Background(), Config{
		Endpoint:      srv.server.URL,
		Region:        "ap-singapore-1",
		Bucket:        "test-bucket",
		AccessKey:     "AKID",
		SecretKey:     "SECRET",
		PublicBaseURL: "https://cdn.example.com",
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return c
}

func TestScopedKey_PrefixesByTenant(t *testing.T) {
	got, err := scopedKey("tenant-abc", "products/p1/main.jpg")
	if err != nil {
		t.Fatalf("scopedKey: %v", err)
	}
	want := "tenants/tenant-abc/products/p1/main.jpg"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestScopedKey_RejectsBadInput(t *testing.T) {
	cases := []struct {
		name, tenant, key string
	}{
		{"empty tenant", "", "k"},
		{"empty key", "t", ""},
		{"slash in tenant", "te/nant", "k"},
		{"dotdot in tenant", "..", "k"},
		{"dotdot in key", "t", "../other"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := scopedKey(tc.tenant, tc.key); err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestScopedKey_StripsLeadingSlash(t *testing.T) {
	got, _ := scopedKey("t1", "/leading/slash.png")
	if got != "tenants/t1/leading/slash.png" {
		t.Errorf("got %q", got)
	}
}

func TestPutGetExistsDelete_Roundtrip(t *testing.T) {
	srv := newStubS3Server(t)
	c := newTestClient(t, srv)
	ctx := context.Background()

	body := strings.NewReader("hello world")
	if err := c.Put(ctx, "tenant-1", "products/p1/main.jpg", body, "image/jpeg"); err != nil {
		t.Fatalf("put: %v", err)
	}

	exists, err := c.Exists(ctx, "tenant-1", "products/p1/main.jpg")
	if err != nil || !exists {
		t.Fatalf("exists: got %v, err %v", exists, err)
	}

	rc, err := c.Get(ctx, "tenant-1", "products/p1/main.jpg")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer rc.Close()
	got, _ := io.ReadAll(rc)
	if string(got) != "hello world" {
		t.Errorf("got %q", got)
	}

	if err := c.Delete(ctx, "tenant-1", "products/p1/main.jpg"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	exists, _ = c.Exists(ctx, "tenant-1", "products/p1/main.jpg")
	if exists {
		t.Error("expected object to be gone after delete")
	}
}

func TestExists_FalseForMissing(t *testing.T) {
	srv := newStubS3Server(t)
	c := newTestClient(t, srv)
	exists, err := c.Exists(context.Background(), "t1", "nope.jpg")
	if err != nil {
		t.Fatalf("expected no error for missing object, got %v", err)
	}
	if exists {
		t.Error("expected exists=false")
	}
}

func TestPut_TenantIsolation(t *testing.T) {
	srv := newStubS3Server(t)
	c := newTestClient(t, srv)
	ctx := context.Background()

	_ = c.Put(ctx, "tenant-A", "k.txt", strings.NewReader("A"), "text/plain")
	_ = c.Put(ctx, "tenant-B", "k.txt", strings.NewReader("B"), "text/plain")

	rc, _ := c.Get(ctx, "tenant-A", "k.txt")
	defer rc.Close()
	got, _ := io.ReadAll(rc)
	if string(got) != "A" {
		t.Errorf("tenant-A leaked tenant-B's content: got %q", got)
	}
}

func TestPresignPut_ReturnsUsableURL(t *testing.T) {
	srv := newStubS3Server(t)
	c := newTestClient(t, srv)
	ctx := context.Background()

	pre, err := c.PresignPut(ctx, "t1", "uploads/file.bin", "application/octet-stream", 5*time.Minute)
	if err != nil {
		t.Fatalf("presign: %v", err)
	}
	if !strings.Contains(pre.URL, "/test-bucket/tenants/t1/uploads/file.bin") {
		t.Errorf("unexpected URL: %s", pre.URL)
	}
	if pre.Method != "PUT" {
		t.Errorf("expected method PUT, got %s", pre.Method)
	}
	if pre.Headers["Content-Type"] != "application/octet-stream" {
		t.Errorf("expected Content-Type header in presign response, got %v", pre.Headers)
	}
	if time.Until(pre.ExpiresAt) < time.Minute {
		t.Errorf("ExpiresAt too close: %v", pre.ExpiresAt)
	}

	// Use the URL — the stub doesn't validate the signature so this just
	// proves the URL is structurally usable.
	req, err := http.NewRequest("PUT", pre.URL, strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT to presigned URL: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("presigned PUT status: %d", resp.StatusCode)
	}
}

func TestPresignGet_ReturnsURL(t *testing.T) {
	srv := newStubS3Server(t)
	c := newTestClient(t, srv)
	ctx := context.Background()

	pre, err := c.PresignGet(ctx, "t1", "uploads/file.bin", time.Minute)
	if err != nil {
		t.Fatalf("presign get: %v", err)
	}
	if !strings.Contains(pre.URL, "/test-bucket/tenants/t1/uploads/file.bin") {
		t.Errorf("unexpected URL: %s", pre.URL)
	}
	if pre.Method != "GET" {
		t.Errorf("expected method GET, got %s", pre.Method)
	}
}

func TestPublicURL_UsesCDNBaseWhenSet(t *testing.T) {
	srv := newStubS3Server(t)
	c := newTestClient(t, srv)
	url := c.PublicURL("t1", "products/p1/main.jpg")
	if !strings.HasPrefix(url, "https://cdn.example.com/") {
		t.Errorf("expected CDN base URL, got %q", url)
	}
	if !strings.Contains(url, "tenants/t1/products/p1/main.jpg") {
		t.Errorf("expected tenant prefix in URL, got %q", url)
	}
}

func TestPublicURL_FallsBackToEndpoint(t *testing.T) {
	c, err := New(context.Background(), Config{
		Endpoint:  "https://endpoint.example",
		Region:    "x",
		Bucket:    "b",
		AccessKey: "a",
		SecretKey: "s",
		// no PublicBaseURL
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	got := c.PublicURL("t1", "k.png")
	if got != "https://endpoint.example/b/tenants/t1/k.png" {
		t.Errorf("got %q", got)
	}
}

func TestPublicURL_EmptyOnInvalidInput(t *testing.T) {
	srv := newStubS3Server(t)
	c := newTestClient(t, srv)
	if got := c.PublicURL("", "k"); got != "" {
		t.Errorf("expected empty URL for blank tenant, got %q", got)
	}
}

func TestNewFromEnv_RequiresAllValues(t *testing.T) {
	env := map[string]string{
		"OCI_S3_ENDPOINT":   "https://x.example",
		"OCI_S3_REGION":     "ap-singapore-1",
		"OCI_S3_BUCKET":     "b",
		"OCI_S3_ACCESS_KEY": "",
		"OCI_S3_SECRET_KEY": "s",
	}
	_, err := NewFromEnv(func(k string) string { return env[k] })
	if err == nil || !strings.Contains(err.Error(), "OCI_S3_ACCESS_KEY") {
		t.Errorf("expected missing-access-key error, got %v", err)
	}
}

func TestNewFromEnv_AllValuesPresent(t *testing.T) {
	env := map[string]string{
		"OCI_S3_ENDPOINT":         "https://x.example",
		"OCI_S3_REGION":           "ap-singapore-1",
		"OCI_S3_BUCKET":           "b",
		"OCI_S3_ACCESS_KEY":       "a",
		"OCI_S3_SECRET_KEY":       "s",
		"OCI_S3_PUBLIC_BASE_URL":  "https://cdn.example",
	}
	cfg, err := NewFromEnv(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("NewFromEnv: %v", err)
	}
	if cfg.Bucket != "b" || cfg.PublicBaseURL != "https://cdn.example" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestNew_RejectsMissingBucket(t *testing.T) {
	_, err := New(context.Background(), Config{Region: "x"})
	if err == nil || !strings.Contains(err.Error(), "bucket") {
		t.Errorf("expected bucket-required error, got %v", err)
	}
}

func TestNew_RejectsMissingRegion(t *testing.T) {
	_, err := New(context.Background(), Config{Bucket: "b"})
	if err == nil || !strings.Contains(err.Error(), "region") {
		t.Errorf("expected region-required error, got %v", err)
	}
}

// Compile-time interface-satisfaction check.
var _ Client = (*client)(nil)

// Ensure errors-package import is preserved for future use.
var _ = errors.New
