package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newRouterWithSecurityHeaders(cfg SecurityHeadersConfig) *gin.Engine {
	r := gin.New()
	r.Use(SecurityHeaders(cfg))
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestSecurityHeaders_Defaults(t *testing.T) {
	r := newRouterWithSecurityHeaders(SecurityHeadersConfig{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	checks := map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "camera=(), microphone=(), geolocation=(), interest-cohort=()",
		"X-XSS-Protection":          "0",
		"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
	}

	for header, want := range checks {
		got := w.Header().Get(header)
		if got != want {
			t.Errorf("header %s: want %q, got %q", header, want, got)
		}
	}
}

func TestSecurityHeaders_Overrides(t *testing.T) {
	customCSP := "default-src 'self'; script-src 'self' 'unsafe-inline'"
	r := newRouterWithSecurityHeaders(SecurityHeadersConfig{
		FrameOptions:          "SAMEORIGIN",
		ContentSecurityPolicy: customCSP,
		ReferrerPolicy:        "no-referrer",
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options: want SAMEORIGIN, got %q", got)
	}
	if got := w.Header().Get("Content-Security-Policy"); got != customCSP {
		t.Errorf("CSP: want %q, got %q", customCSP, got)
	}
	if got := w.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy: want no-referrer, got %q", got)
	}
	// Unset fields fall back to defaults.
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options default lost: got %q", got)
	}
}

func TestSecurityHeaders_DisableCSP(t *testing.T) {
	r := newRouterWithSecurityHeaders(SecurityHeadersConfig{
		DisableContentSecurityPolicy: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Security-Policy"); got != "" {
		t.Errorf("CSP should be omitted when disabled, got %q", got)
	}
	// Other headers should still be present.
	if got := w.Header().Get("Strict-Transport-Security"); got == "" {
		t.Errorf("HSTS should still be set when only CSP is disabled")
	}
}

func TestSecurityHeaders_LandsOnErrorResponses(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders(SecurityHeadersConfig{}))
	r.GET("/boom", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("security headers should apply to error responses too, got %q", got)
	}
}
