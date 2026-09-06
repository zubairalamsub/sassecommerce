package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// fakeProviderRepo is an in-memory EmailProviderRepository for handler tests.
type fakeProviderRepo struct {
	byScope map[string][]models.EmailProviderConfig
	encrypt bool
}

func newFakeProviderRepo() *fakeProviderRepo {
	return &fakeProviderRepo{byScope: map[string][]models.EmailProviderConfig{}, encrypt: true}
}

func (r *fakeProviderRepo) List(ctx context.Context, tenantID string) ([]models.EmailProviderConfig, error) {
	return r.byScope[tenantID], nil
}
func (r *fakeProviderRepo) ListEnabled(ctx context.Context, tenantID string) ([]models.EmailProviderConfig, error) {
	return r.byScope[tenantID], nil
}
func (r *fakeProviderRepo) Get(ctx context.Context, tenantID, provider string) (*models.EmailProviderConfig, error) {
	for i := range r.byScope[tenantID] {
		if r.byScope[tenantID][i].Provider == provider {
			// A copy, matching Mongo's decode. Returning a pointer into the
			// slice would let a caller mutate stored state by accident and
			// mask exactly the bug this file is guarding.
			cfg := r.byScope[tenantID][i]
			return &cfg, nil
		}
	}
	return nil, repository.ErrProviderConfigNotFound
}
func (r *fakeProviderRepo) Upsert(ctx context.Context, cfg *models.EmailProviderConfig) error {
	for i := range r.byScope[cfg.TenantID] {
		if r.byScope[cfg.TenantID][i].Provider == cfg.Provider {
			// Mirror the real repository: an empty secret keeps the stored one.
			if cfg.Secret == "" {
				cfg.Secret = r.byScope[cfg.TenantID][i].Secret
			}
			r.byScope[cfg.TenantID][i] = *cfg
			return nil
		}
	}
	r.byScope[cfg.TenantID] = append(r.byScope[cfg.TenantID], *cfg)
	return nil
}
func (r *fakeProviderRepo) Delete(ctx context.Context, tenantID, provider string) error {
	for i := range r.byScope[tenantID] {
		if r.byScope[tenantID][i].Provider == provider {
			r.byScope[tenantID] = append(r.byScope[tenantID][:i], r.byScope[tenantID][i+1:]...)
			return nil
		}
	}
	return repository.ErrProviderConfigNotFound
}
func (r *fakeProviderRepo) EncryptSecret(p string) (string, error) { return "enc(" + p + ")", nil }
func (r *fakeProviderRepo) DecryptSecret(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if len(s) > 5 && s[:4] == "enc(" {
		return s[4 : len(s)-1], nil
	}
	return "", errors.New("not decryptable")
}
func (r *fakeProviderRepo) UsingEncryption() bool { return r.encrypt }

// setupProviderRouter stubs the auth middleware with headers: X-Test-Tenant
// supplies the JWT tenant claim, X-Test-Role the role.
func setupProviderRouter(repo repository.EmailProviderRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	h := NewEmailProviderHandler(repo, nil, logger)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if t := c.GetHeader("X-Test-Tenant"); t != "" {
			c.Set("tenant_id", t)
		}
		if r := c.GetHeader("X-Test-Role"); r != "" {
			c.Set("role", r)
		}
		c.Next()
	})
	// Registered without RequireRole so the handler bodies are exercised
	// directly; the role gate is a router-level concern.
	group := router.Group("/api/v1/email-providers")
	{
		group.GET("", h.ListProviders)
		group.PUT("", h.UpsertProvider)
		group.POST("/test", h.TestProvider)
		group.DELETE("/:provider", h.DeleteProvider)
	}
	return router
}

func doProviderRequest(router *gin.Engine, method, path, tenant, role string, body interface{}) *httptest.ResponseRecorder {
	var payload io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		payload = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, path, payload)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tenant != "" {
		req.Header.Set("X-Test-Tenant", tenant)
	}
	if role != "" {
		req.Header.Set("X-Test-Role", role)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeProviderBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return out
}

// Regression: a super_admin's JWT carries an empty tenant_id, so requiring a
// tenant before checking the platform branch made the platform scope
// unreachable by the only role allowed to configure it.
func TestPlatformScopeReachableBySuperAdminWithNoTenant(t *testing.T) {
	repo := newFakeProviderRepo()
	router := setupProviderRouter(repo)

	w := doProviderRequest(router, http.MethodGet, "/api/v1/email-providers?scope=platform", "", "super_admin", nil)

	assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	assert.Equal(t, models.PlatformScope, decodeProviderBody(t, w)["scope"])
}

// A tenant admin must not be able to reach the platform default by asking.
func TestPlatformScopeForbiddenForTenantAdmin(t *testing.T) {
	router := setupProviderRouter(newFakeProviderRepo())

	w := doProviderRequest(router, http.MethodGet, "/api/v1/email-providers?scope=platform", "tenant-1", "admin", nil)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// A tenant-less super_admin asking for tenant scope gets a pointer to the
// right query rather than a bare 401 that looks like an auth failure.
func TestTenantScopeWithoutTenantExplainsItself(t *testing.T) {
	router := setupProviderRouter(newFakeProviderRepo())

	w := doProviderRequest(router, http.MethodGet, "/api/v1/email-providers", "", "super_admin", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := decodeProviderBody(t, w)
	assert.Equal(t, "tenant_required", body["error"])
	assert.Contains(t, body["message"], "scope=platform")
}

func TestUnauthenticatedIsRejected(t *testing.T) {
	router := setupProviderRouter(newFakeProviderRepo())

	w := doProviderRequest(router, http.MethodGet, "/api/v1/email-providers", "", "", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// Scope comes from the JWT, never a parameter — so a tenant admin reads and
// writes only its own row no matter what it asks for.
func TestTenantScopeIsTakenFromTheJWT(t *testing.T) {
	repo := newFakeProviderRepo()
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{{TenantID: "tenant-1", Provider: "mailjet", Enabled: true}}
	repo.byScope["tenant-victim"] = []models.EmailProviderConfig{{TenantID: "tenant-victim", Provider: "brevo", Enabled: true}}
	router := setupProviderRouter(repo)

	w := doProviderRequest(router, http.MethodGet, "/api/v1/email-providers?tenant_id=tenant-victim&scope=tenant-victim", "tenant-1", "admin", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeProviderBody(t, w)
	assert.Equal(t, "tenant-1", body["scope"], "scope must come from the JWT, not the query")

	data, _ := body["data"].([]interface{})
	if assert.Len(t, data, 1) {
		first, _ := data[0].(map[string]interface{})
		assert.Equal(t, "mailjet", first["provider"], "another tenant's provider leaked")
	}
}

// The stored credential must never come back, in any field.
func TestSecretIsNeverReturned(t *testing.T) {
	repo := newFakeProviderRepo()
	const plaintext = "super-secret-vendor-key"
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{{
		TenantID: "tenant-1", Provider: "mailjet", Enabled: true,
		Username: "api-key", Secret: "enc(" + plaintext + ")",
	}}
	router := setupProviderRouter(repo)

	w := doProviderRequest(router, http.MethodGet, "/api/v1/email-providers", "tenant-1", "admin", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	raw := w.Body.String()
	assert.NotContains(t, raw, plaintext, "the plaintext secret was returned")
	assert.NotContains(t, raw, "enc("+plaintext+")", "the ciphertext was returned")

	data, _ := decodeProviderBody(t, w)["data"].([]interface{})
	first, _ := data[0].(map[string]interface{})
	assert.Equal(t, true, first["secret_set"], "the UI still needs to know a credential exists")
	assert.NotEmpty(t, first["secret_hint"])
}

// A tenant with nothing of its own sees the platform chain, flagged so the UI
// can render it read-only.
func TestTenantInheritsPlatformDefault(t *testing.T) {
	repo := newFakeProviderRepo()
	repo.byScope[models.PlatformScope] = []models.EmailProviderConfig{
		{TenantID: models.PlatformScope, Provider: "brevo", Enabled: true},
	}
	router := setupProviderRouter(repo)

	w := doProviderRequest(router, http.MethodGet, "/api/v1/email-providers", "tenant-with-nothing", "admin", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeProviderBody(t, w)
	assert.Equal(t, true, body["inherited"])

	data, _ := body["data"].([]interface{})
	if assert.Len(t, data, 1) {
		first, _ := data[0].(map[string]interface{})
		assert.Equal(t, true, first["inherited"])
	}
}

func TestUpsertRejectsUnknownProvider(t *testing.T) {
	router := setupProviderRouter(newFakeProviderRepo())

	w := doProviderRequest(router, http.MethodPut, "/api/v1/email-providers", "tenant-1", "admin",
		map[string]interface{}{"provider": "carrierpigeon"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "unknown_provider", decodeProviderBody(t, w)["error"])
}

func TestUpsertEncryptsTheSecret(t *testing.T) {
	repo := newFakeProviderRepo()
	router := setupProviderRouter(repo)

	w := doProviderRequest(router, http.MethodPut, "/api/v1/email-providers", "tenant-1", "admin",
		map[string]interface{}{"provider": "mailjet", "username": "api-key", "secret": "raw-key", "enabled": true})

	assert.Equal(t, http.StatusOK, w.Code)
	stored := repo.byScope["tenant-1"][0]
	assert.Equal(t, "enc(raw-key)", stored.Secret, "the secret was not sealed before storage")
	assert.NotContains(t, w.Body.String(), "raw-key", "the secret came back in the response")
}

// Editing a port or toggling Enabled must not require re-entering the key.
func TestUpsertWithoutSecretKeepsTheStoredOne(t *testing.T) {
	repo := newFakeProviderRepo()
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{{
		TenantID: "tenant-1", Provider: "mailjet", Username: "api-key",
		Secret: "enc(original-key)", Enabled: false,
	}}
	router := setupProviderRouter(repo)

	w := doProviderRequest(router, http.MethodPut, "/api/v1/email-providers", "tenant-1", "admin",
		map[string]interface{}{"provider": "mailjet", "enabled": true})

	assert.Equal(t, http.StatusOK, w.Code)
	stored := repo.byScope["tenant-1"][0]
	assert.Equal(t, "enc(original-key)", stored.Secret, "the stored credential was wiped by a partial update")
	assert.True(t, stored.Enabled)
}

func TestDeleteUnknownProviderIs404(t *testing.T) {
	router := setupProviderRouter(newFakeProviderRepo())

	w := doProviderRequest(router, http.MethodDelete, "/api/v1/email-providers/mailjet", "tenant-1", "admin", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteRemovesOnlyTheCallersRow(t *testing.T) {
	repo := newFakeProviderRepo()
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{{TenantID: "tenant-1", Provider: "mailjet"}}
	repo.byScope["tenant-2"] = []models.EmailProviderConfig{{TenantID: "tenant-2", Provider: "mailjet"}}
	router := setupProviderRouter(repo)

	w := doProviderRequest(router, http.MethodDelete, "/api/v1/email-providers/mailjet", "tenant-1", "admin", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, repo.byScope["tenant-1"])
	assert.Len(t, repo.byScope["tenant-2"], 1, "another tenant's config was deleted")
}

func TestTestProviderRequiresConfiguredProvider(t *testing.T) {
	router := setupProviderRouter(newFakeProviderRepo())

	w := doProviderRequest(router, http.MethodPost, "/api/v1/email-providers/test", "tenant-1", "admin",
		map[string]interface{}{"provider": "mailjet", "to": "ops@example.test"})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTestProviderValidatesRecipient(t *testing.T) {
	router := setupProviderRouter(newFakeProviderRepo())

	w := doProviderRequest(router, http.MethodPost, "/api/v1/email-providers/test", "tenant-1", "admin",
		map[string]interface{}{"provider": "mailjet", "to": "not-an-email"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// A credential that no longer decrypts (key rotated) must tell the operator to
// re-enter it rather than surfacing a decryption error.
func TestTestProviderReportsUndecryptableSecret(t *testing.T) {
	repo := newFakeProviderRepo()
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{{
		TenantID: "tenant-1", Provider: "mailjet", Username: "u", Secret: "garbage-not-enc",
	}}
	router := setupProviderRouter(repo)

	w := doProviderRequest(router, http.MethodPost, "/api/v1/email-providers/test", "tenant-1", "admin",
		map[string]interface{}{"provider": "mailjet", "to": "ops@example.test"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, decodeProviderBody(t, w)["message"], "re-enter")
}

// The simulated provider delivers without a network call, so it exercises the
// success path of test-send.
func TestTestProviderSucceedsWithSimulated(t *testing.T) {
	repo := newFakeProviderRepo()
	repo.byScope["tenant-1"] = []models.EmailProviderConfig{{
		TenantID: "tenant-1", Provider: "simulated", Enabled: true,
	}}
	router := setupProviderRouter(repo)

	w := doProviderRequest(router, http.MethodPost, "/api/v1/email-providers/test", "tenant-1", "admin",
		map[string]interface{}{"provider": "simulated", "to": "ops@example.test"})

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeProviderBody(t, w)
	assert.Equal(t, true, body["success"])
	assert.Equal(t, "ops@example.test", body["sent_to"])
}

// The list response surfaces whether credentials are genuinely encrypted, so
// an operator can see a misconfigured deployment.
func TestListReportsEncryptionState(t *testing.T) {
	repo := newFakeProviderRepo()
	repo.encrypt = false
	router := setupProviderRouter(repo)

	w := doProviderRequest(router, http.MethodGet, "/api/v1/email-providers", "tenant-1", "admin", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, false, decodeProviderBody(t, w)["encrypted_at_rest"])
}
