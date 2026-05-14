package storage

import (
	"fmt"
	"path"
	"strings"
)

// ImageMIMEPolicy maps allowed MIME types to their preferred file extension.
// Each service that accepts image uploads should declare its own policy
// (e.g. avatars typically restrict to JPEG/PNG/WebP only; logos may allow SVG).
type ImageMIMEPolicy map[string]string

// DefaultImagePolicy is the common set of web image formats — appropriate for
// product images, review attachments, user avatars.
var DefaultImagePolicy = ImageMIMEPolicy{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
	"image/avif": ".avif",
}

// BrandingImagePolicy adds SVG, used for logos/favicons.
var BrandingImagePolicy = ImageMIMEPolicy{
	"image/jpeg":    ".jpg",
	"image/png":     ".png",
	"image/webp":    ".webp",
	"image/svg+xml": ".svg",
	"image/x-icon":  ".ico",
	"image/gif":     ".gif",
}

// PickExtension returns a safe extension for the given content type. It prefers
// an extension matching the user's filename (if it's in the allowed set) and
// otherwise falls back to the canonical extension for the MIME type. Returns
// an error if the content type isn't allowed.
//
// This defends against a user uploading "evil.exe" with content_type=image/jpeg
// — the resulting key always ends in a .jpg/.png/etc., never the user's input.
func (p ImageMIMEPolicy) PickExtension(contentType, filename string) (string, error) {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	canonical, ok := p[contentType]
	if !ok {
		allowed := make([]string, 0, len(p))
		for k := range p {
			allowed = append(allowed, k)
		}
		return "", fmt.Errorf("unsupported content type: %s (allowed: %s)", contentType, strings.Join(allowed, ", "))
	}

	if filename == "" {
		return canonical, nil
	}
	userExt := strings.ToLower(path.Ext(filename))
	for _, ext := range p {
		if userExt == ext {
			return userExt, nil
		}
	}
	return canonical, nil
}

// KeyFromPublicURL recovers the tenant-relative storage key from a public URL
// the Client previously returned. Returns false if the URL doesn't reference
// the supplied tenant's prefix — defends against confused-deputy where one
// tenant submits another tenant's URL.
//
// Works with both CDN-style URLs (https://cdn.example.com/tenants/T/k) and
// the path-style fallback (https://endpoint/bucket/tenants/T/k).
func KeyFromPublicURL(tenantID, publicURL string) (string, bool) {
	prefix := fmt.Sprintf("tenants/%s/", tenantID)
	idx := strings.Index(publicURL, prefix)
	if idx < 0 {
		return "", false
	}
	return publicURL[idx+len(prefix):], true
}
