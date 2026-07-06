package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newMetricsRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/metrics", MetricsAuth(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return r
}

func getMetrics(r *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestMetricsAuth_TokenSet(t *testing.T) {
	t.Setenv(envMetricsToken, "scrape-token")
	r := newMetricsRouter(t)

	if w := getMetrics(r, "Bearer scrape-token"); w.Code != http.StatusOK {
		t.Errorf("valid token: got %d, want 200", w.Code)
	}
	if w := getMetrics(r, "Bearer wrong"); w.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: got %d, want 401", w.Code)
	}
	if w := getMetrics(r, ""); w.Code != http.StatusUnauthorized {
		t.Errorf("missing header: got %d, want 401", w.Code)
	}
}

func TestMetricsAuth_NoTokenDev(t *testing.T) {
	t.Setenv(envMetricsToken, "")
	t.Setenv(envEnvironment, "development")
	r := newMetricsRouter(t)

	if w := getMetrics(r, ""); w.Code != http.StatusOK {
		t.Errorf("dev without token: got %d, want 200", w.Code)
	}
}

func TestMetricsAuth_NoTokenProduction(t *testing.T) {
	t.Setenv(envMetricsToken, "")
	t.Setenv(envEnvironment, "production")
	r := newMetricsRouter(t)

	if w := getMetrics(r, ""); w.Code != http.StatusNotFound {
		t.Errorf("production without token: got %d, want 404", w.Code)
	}
}
