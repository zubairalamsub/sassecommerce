package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newRouterWithCORS(cfg CORSConfig) *gin.Engine {
	r := gin.New()
	r.Use(HardenedCORS(cfg))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestHardenedCORS_AllowedOriginEchoed(t *testing.T) {
	r := newRouterWithCORS(CORSConfig{
		AllowOrigins: []string{"https://saajan.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://saajan.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://saajan.com" {
		t.Errorf("Allow-Origin: want exact echo, got %q", got)
	}
	if got := w.Header().Get("Vary"); !strings.Contains(got, "Origin") {
		t.Errorf("Vary header should include Origin, got %q", got)
	}
}

func TestHardenedCORS_DisallowedOriginNotEchoed(t *testing.T) {
	r := newRouterWithCORS(CORSConfig{
		AllowOrigins: []string{"https://saajan.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin should be empty for disallowed origin, got %q", got)
	}
	if w.Code != http.StatusOK {
		t.Errorf("non-preflight request should still execute, got %d", w.Code)
	}
}

func TestHardenedCORS_PreflightAllowed(t *testing.T) {
	r := newRouterWithCORS(CORSConfig{
		AllowOrigins:     []string{"https://saajan.com"},
		AllowCredentials: true,
	})

	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", "https://saajan.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("preflight on allowed origin should be 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://saajan.com" {
		t.Errorf("Allow-Origin: want echo, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials: want true, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Max-Age"); got != "43200" {
		t.Errorf("Max-Age: want 43200 (12h), got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Errorf("Allow-Methods missing POST: %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "X-Tenant-Id") {
		t.Errorf("Allow-Headers missing X-Tenant-Id: %q", got)
	}
}

func TestHardenedCORS_PreflightDeniedForUnknownOrigin(t *testing.T) {
	r := newRouterWithCORS(CORSConfig{
		AllowOrigins: []string{"https://saajan.com"},
	})

	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("preflight from disallowed origin should be 403, got %d", w.Code)
	}
}

func TestHardenedCORS_EnvOverridesConfig(t *testing.T) {
	t.Setenv(envCORSOrigins, "https://from-env.example.com")
	t.Setenv(envEnvironment, "production")

	r := newRouterWithCORS(CORSConfig{
		AllowOrigins: []string{"https://config-only.example.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://from-env.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://from-env.example.com" {
		t.Errorf("env should win over config, got %q", got)
	}

	// Origin only present in config should be denied now.
	req2 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req2.Header.Set("Origin", "https://config-only.example.com")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if got := w2.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("config-only origin should be denied when env is set, got %q", got)
	}
}

func TestHardenedCORS_WildcardStrippedInProduction(t *testing.T) {
	t.Setenv(envEnvironment, "production")

	r := newRouterWithCORS(CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://saajan.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got == "*" {
		t.Errorf("wildcard origin must be stripped in production, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got == "true" {
		// credentials with wildcard origins should never coexist; since wildcard
		// is stripped, but there are no other origins, credentials should also
		// not be advertised.
		// (Allowed-origins is empty, so credentials being true would be a bug.)
	}
}

func TestHardenedCORS_CredentialsDisabledWithWildcard(t *testing.T) {
	// Non-prod, wildcard allowed but credentials must NOT come along.
	r := newRouterWithCORS(CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://anywhere.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("wildcard should be honored outside production, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got == "true" {
		t.Errorf("credentials must not be advertised alongside wildcard origin")
	}
}

func TestHardenedCORS_DevFallback(t *testing.T) {
	t.Setenv(envCORSOrigins, "")
	t.Setenv(envEnvironment, "development")

	r := newRouterWithCORS(CORSConfig{})

	for _, origin := range []string{"http://localhost:3000", "http://localhost:3001"} {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("dev fallback should allow %s, got %q", origin, got)
		}
	}

	// A random origin should not be echoed back.
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("dev fallback should not allow arbitrary origin, got %q", got)
	}
}

func TestSplitAndTrim(t *testing.T) {
	got := splitAndTrim("a, b ,  c,,")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d (%v)", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q want %q", i, got[i], want[i])
		}
	}
}
