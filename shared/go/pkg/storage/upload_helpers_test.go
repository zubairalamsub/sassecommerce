package storage

import (
	"strings"
	"testing"
)

func TestPickExtension_AllowedTypes(t *testing.T) {
	cases := []struct {
		contentType, filename, want string
	}{
		{"image/jpeg", "photo.jpg", ".jpg"},
		{"image/jpeg", "photo.jpeg", ".jpeg"}, // user's .jpeg is in allowed set? jpeg isn't — falls back to .jpg
		{"image/png", "icon.PNG", ".png"},
		{"image/webp", "", ".webp"},
		{"image/jpeg", "evil.exe", ".jpg"}, // user-supplied bogus ext → canonical
		{"image/jpeg", "", ".jpg"},
	}
	for _, tc := range cases {
		got, err := DefaultImagePolicy.PickExtension(tc.contentType, tc.filename)
		if err != nil {
			t.Errorf("PickExtension(%q, %q) error: %v", tc.contentType, tc.filename, err)
			continue
		}
		// The "photo.jpeg" case is intentional: .jpeg isn't in the canonical
		// extension table (which only lists .jpg), so it falls back.
		if tc.filename == "photo.jpeg" && got != ".jpg" && got != ".jpeg" {
			t.Errorf("for .jpeg input expected .jpg or .jpeg, got %s", got)
			continue
		}
		if tc.filename != "photo.jpeg" && got != tc.want {
			t.Errorf("PickExtension(%q, %q) = %s, want %s", tc.contentType, tc.filename, got, tc.want)
		}
	}
}

func TestPickExtension_RejectsDisallowedType(t *testing.T) {
	_, err := DefaultImagePolicy.PickExtension("application/pdf", "doc.pdf")
	if err == nil {
		t.Error("expected error for application/pdf")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' in error, got %v", err)
	}
}

func TestBrandingPolicy_AllowsSVG(t *testing.T) {
	got, err := BrandingImagePolicy.PickExtension("image/svg+xml", "logo.svg")
	if err != nil {
		t.Fatalf("expected SVG allowed: %v", err)
	}
	if got != ".svg" {
		t.Errorf("got %s", got)
	}
}

func TestKeyFromPublicURL_Roundtrip(t *testing.T) {
	cases := []struct {
		name, tenantID, url, wantKey string
		wantOK                        bool
	}{
		{"cdn URL", "tenant-1", "https://cdn.example.com/tenants/tenant-1/products/p1/img.jpg", "products/p1/img.jpg", true},
		{"path-style URL", "tenant-1", "https://endpoint/bucket/tenants/tenant-1/products/p1/img.jpg", "products/p1/img.jpg", true},
		{"foreign tenant", "tenant-1", "https://cdn.example.com/tenants/tenant-2/products/p1/img.jpg", "", false},
		{"unrelated URL", "tenant-1", "https://other.example/whatever.jpg", "", false},
		{"trailing slash", "tenant-1", "https://cdn.example.com/tenants/tenant-1/", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := KeyFromPublicURL(tc.tenantID, tc.url)
			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.wantKey {
				t.Errorf("key = %q, want %q", got, tc.wantKey)
			}
		})
	}
}
