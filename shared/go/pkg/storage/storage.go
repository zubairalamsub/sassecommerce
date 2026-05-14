// Package storage provides an S3-compatible object storage client used by
// services that need to upload/serve files (product images, shipping labels,
// invoices, exports, etc.).
//
// It targets Oracle Cloud Infrastructure (OCI) Object Storage in S3
// compatibility mode by default but works with any S3-compatible backend
// (AWS S3, Cloudflare R2, MinIO) — switch by changing the endpoint.
//
// All keys are automatically prefixed with `tenants/{tenantID}/` so a tenant
// can never read another tenant's objects through this client.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client is a thin wrapper over the S3 client. Methods accept a tenantID and
// scope every operation under tenants/{tenantID}/.
type Client interface {
	// Put uploads an object. ContentType should be a real MIME type (image/jpeg,
	// application/pdf, etc.) — it's set on the object so CDN responses serve it
	// correctly without sniffing.
	Put(ctx context.Context, tenantID, key string, body io.Reader, contentType string) error

	// Get fetches an object's body. Caller must close the returned ReadCloser.
	Get(ctx context.Context, tenantID, key string) (io.ReadCloser, error)

	// Delete removes an object.
	Delete(ctx context.Context, tenantID, key string) error

	// Exists reports whether an object is present.
	Exists(ctx context.Context, tenantID, key string) (bool, error)

	// PresignPut returns a URL the browser can PUT to directly. The URL is
	// scoped to the supplied content type and expires after expiresIn.
	PresignPut(ctx context.Context, tenantID, key, contentType string, expiresIn time.Duration) (PresignedURL, error)

	// PresignGet returns a short-lived URL for reading a private object.
	// Use the public CDN URL for objects that are meant to be public.
	PresignGet(ctx context.Context, tenantID, key string, expiresIn time.Duration) (PresignedURL, error)

	// PublicURL returns the URL a browser can hit to read a public object.
	// When PublicBaseURL is configured (CDN), it's used; otherwise the bucket
	// endpoint is used.
	PublicURL(tenantID, key string) string

	// Bucket returns the underlying bucket name. Useful for diagnostics.
	Bucket() string
}

// PresignedURL is the response from PresignPut/PresignGet. Headers are the
// headers the client MUST send on the PUT (commonly Content-Type) for the
// signature to validate.
type PresignedURL struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// Config configures the storage client.
type Config struct {
	// Endpoint is the S3-compatible endpoint URL.
	// OCI:        https://<namespace>.compat.objectstorage.<region>.oraclecloud.com
	// AWS S3:     leave blank
	// R2:         https://<account-id>.r2.cloudflarestorage.com
	// MinIO:      http://minio:9000
	Endpoint string

	// Region. For OCI use the OCI region (e.g. ap-singapore-1). For AWS use
	// the AWS region. For R2 use "auto".
	Region string

	// Bucket is the target bucket name.
	Bucket string

	// AccessKey / SecretKey authenticate requests. For OCI these come from
	// "Customer Secret Keys" generated under your IAM user.
	AccessKey string
	SecretKey string

	// UsePathStyle forces path-style addressing (https://endpoint/bucket/key)
	// instead of virtual-hosted (https://bucket.endpoint/key). Required for
	// OCI and MinIO. Defaults to true when an Endpoint is set.
	UsePathStyle *bool

	// PublicBaseURL is the public/CDN URL prefix, used by PublicURL().
	// Examples:
	//   https://cdn.saajan.com
	//   https://<namespace>.objectstorage.<region>.oci.customer-oci.com/n/<namespace>/b/<bucket>/o
	// If empty, PublicURL returns a presigned GET URL of 1h expiry.
	PublicBaseURL string
}

// NewFromEnv builds a Config from standard env vars. Returns an error
// listing all missing required vars at once.
//
//	OCI_S3_ENDPOINT
//	OCI_S3_REGION
//	OCI_S3_BUCKET
//	OCI_S3_ACCESS_KEY
//	OCI_S3_SECRET_KEY
//	OCI_S3_PUBLIC_BASE_URL  (optional)
func NewFromEnv(getenv func(string) string) (*Config, error) {
	required := []string{"OCI_S3_ENDPOINT", "OCI_S3_REGION", "OCI_S3_BUCKET", "OCI_S3_ACCESS_KEY", "OCI_S3_SECRET_KEY"}
	var missing []string
	values := map[string]string{}
	for _, k := range required {
		v := getenv(k)
		if v == "" {
			missing = append(missing, k)
		}
		values[k] = v
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("storage: missing required env vars: %s", strings.Join(missing, ", "))
	}
	return &Config{
		Endpoint:      values["OCI_S3_ENDPOINT"],
		Region:        values["OCI_S3_REGION"],
		Bucket:        values["OCI_S3_BUCKET"],
		AccessKey:     values["OCI_S3_ACCESS_KEY"],
		SecretKey:     values["OCI_S3_SECRET_KEY"],
		PublicBaseURL: getenv("OCI_S3_PUBLIC_BASE_URL"),
	}, nil
}

// New constructs a Client from a Config.
func New(ctx context.Context, cfg Config) (Client, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("storage: bucket is required")
	}
	if cfg.Region == "" {
		return nil, errors.New("storage: region is required")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
		// AWS SDK v2 recently started defaulting checksum calculation to
		// WhenSupported, which triggers `aws-chunked` Content-Encoding on
		// uploads. OCI Object Storage (and several other S3-compatible
		// backends) reject that with: "NotImplemented: AWS chunked encoding
		// not supported." Force WhenRequired to fall back to plain content.
		awsconfig.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
		awsconfig.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: load aws config: %w", err)
	}

	pathStyle := true
	if cfg.UsePathStyle != nil {
		pathStyle = *cfg.UsePathStyle
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = pathStyle
	})

	return &client{
		s3:            s3Client,
		presigner:     s3.NewPresignClient(s3Client),
		bucket:        cfg.Bucket,
		endpoint:      cfg.Endpoint,
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
	}, nil
}

type client struct {
	s3            *s3.Client
	presigner     *s3.PresignClient
	bucket        string
	endpoint      string
	publicBaseURL string
}

func (c *client) Bucket() string { return c.bucket }

// scopedKey prefixes a tenant-relative key with tenants/{tenantID}/.
// Returns an error for blank tenantID/key or any attempt at path traversal.
func scopedKey(tenantID, key string) (string, error) {
	tenantID = strings.TrimSpace(tenantID)
	key = strings.TrimSpace(key)
	if tenantID == "" {
		return "", errors.New("storage: tenantID is required")
	}
	if key == "" {
		return "", errors.New("storage: key is required")
	}
	if strings.Contains(tenantID, "/") || strings.Contains(tenantID, "..") {
		return "", errors.New("storage: invalid tenantID")
	}
	if strings.Contains(key, "..") {
		return "", errors.New("storage: invalid key (path traversal)")
	}
	key = strings.TrimLeft(key, "/")
	return "tenants/" + tenantID + "/" + key, nil
}

func (c *client) Put(ctx context.Context, tenantID, key string, body io.Reader, contentType string) error {
	full, err := scopedKey(tenantID, key)
	if err != nil {
		return err
	}
	_, err = c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &c.bucket,
		Key:         &full,
		Body:        body,
		ContentType: optString(contentType),
	})
	if err != nil {
		return fmt.Errorf("storage: put %s: %w", full, err)
	}
	return nil
}

func (c *client) Get(ctx context.Context, tenantID, key string) (io.ReadCloser, error) {
	full, err := scopedKey(tenantID, key)
	if err != nil {
		return nil, err
	}
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &full,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: get %s: %w", full, err)
	}
	return out.Body, nil
}

func (c *client) Delete(ctx context.Context, tenantID, key string) error {
	full, err := scopedKey(tenantID, key)
	if err != nil {
		return err
	}
	_, err = c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &c.bucket,
		Key:    &full,
	})
	if err != nil {
		return fmt.Errorf("storage: delete %s: %w", full, err)
	}
	return nil
}

func (c *client) Exists(ctx context.Context, tenantID, key string) (bool, error) {
	full, err := scopedKey(tenantID, key)
	if err != nil {
		return false, err
	}
	_, err = c.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &c.bucket,
		Key:    &full,
	})
	if err != nil {
		// Treat NotFound as "doesn't exist"; surface other errors.
		if isNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("storage: head %s: %w", full, err)
	}
	return true, nil
}

func (c *client) PresignPut(ctx context.Context, tenantID, key, contentType string, expiresIn time.Duration) (PresignedURL, error) {
	full, err := scopedKey(tenantID, key)
	if err != nil {
		return PresignedURL{}, err
	}
	if expiresIn <= 0 {
		expiresIn = 15 * time.Minute
	}
	req, err := c.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      &c.bucket,
		Key:         &full,
		ContentType: optString(contentType),
	}, func(o *s3.PresignOptions) {
		o.Expires = expiresIn
	})
	if err != nil {
		return PresignedURL{}, fmt.Errorf("storage: presign put %s: %w", full, err)
	}
	headers := map[string]string{}
	if contentType != "" {
		headers["Content-Type"] = contentType
	}
	return PresignedURL{
		URL:       req.URL,
		Method:    req.Method,
		Headers:   headers,
		ExpiresAt: time.Now().Add(expiresIn),
	}, nil
}

func (c *client) PresignGet(ctx context.Context, tenantID, key string, expiresIn time.Duration) (PresignedURL, error) {
	full, err := scopedKey(tenantID, key)
	if err != nil {
		return PresignedURL{}, err
	}
	if expiresIn <= 0 {
		expiresIn = time.Hour
	}
	req, err := c.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &full,
	}, func(o *s3.PresignOptions) {
		o.Expires = expiresIn
	})
	if err != nil {
		return PresignedURL{}, fmt.Errorf("storage: presign get %s: %w", full, err)
	}
	return PresignedURL{
		URL:       req.URL,
		Method:    req.Method,
		ExpiresAt: time.Now().Add(expiresIn),
	}, nil
}

func (c *client) PublicURL(tenantID, key string) string {
	full, err := scopedKey(tenantID, key)
	if err != nil {
		return ""
	}
	if c.publicBaseURL != "" {
		return c.publicBaseURL + "/" + escapePath(full)
	}
	// Fallback: build a path-style URL against the configured endpoint.
	if c.endpoint == "" {
		return ""
	}
	return strings.TrimRight(c.endpoint, "/") + "/" + c.bucket + "/" + full
}

func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// escapePath URL-escapes each segment of a slash-delimited path while
// preserving the slashes — url.PathEscape would escape the slashes too.
func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}

// isNotFound matches the S3 SDK's "object not found" / "no such key" responses.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "NotFound") ||
		strings.Contains(msg, "NoSuchKey") ||
		strings.Contains(msg, "404") ||
		strings.Contains(msg, "status code: 404")
}
