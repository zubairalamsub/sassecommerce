package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// doTemplateRequest issues a request against the template router. A nil body
// sends no payload at all, which several endpoints treat as "use defaults".
func doTemplateRequest(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var payload io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		payload = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, path, payload)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func doTemplateRawRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func templateFixture() *models.NotificationTemplate {
	return &models.NotificationTemplate{
		ID:              "tmpl-1",
		TenantID:        "tenant-1",
		Type:            models.TypeWelcome,
		Channel:         models.ChannelEmail,
		Name:            "Welcome email",
		SubjectTemplate: "Welcome, {{.UserName}}!",
		BodyTemplate:    "<p>Hello</p>",
		IsActive:        true,
	}
}

// Every template endpoint is admin-only and tenant-scoped; without a JWT
// tenant claim they must all refuse before touching the service.
func TestTemplateEndpointsRequireATenantClaim(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{name: "list", method: http.MethodGet, path: "/api/v1/notification-templates"},
		{name: "get", method: http.MethodGet, path: "/api/v1/notification-templates/tmpl-1"},
		{name: "create", method: http.MethodPost, path: "/api/v1/notification-templates", body: map[string]interface{}{"type": "welcome", "channel": "email", "name": "n", "body_template": "b"}},
		{name: "update", method: http.MethodPut, path: "/api/v1/notification-templates/tmpl-1", body: map[string]interface{}{"name": "n"}},
		{name: "delete", method: http.MethodDelete, path: "/api/v1/notification-templates/tmpl-1"},
		{name: "preview", method: http.MethodPost, path: "/api/v1/notification-templates/tmpl-1/preview", body: map[string]interface{}{}},
		{name: "test-send", method: http.MethodPost, path: "/api/v1/notification-templates/tmpl-1/test-send", body: map[string]interface{}{"email": "a@b.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(MockTemplateService)
			// The X-Tenant-Id header is deliberately not honoured.
			router := setupTemplateRouterWithTenant(mockSvc, "")

			w := doTemplateRequest(router, tt.method, tt.path, tt.body)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Empty(t, mockSvc.Calls, "no service call may happen without a tenant claim")
		})
	}
}

// ---------------------------------------------------------------------- List

func TestTemplateHandlerList(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("List", mock.Anything, "tenant-1").Return([]models.NotificationTemplate{*templateFixture()}, nil)
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodGet, "/api/v1/notification-templates", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data []models.NotificationTemplate `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body.Data, 1)
	assert.Equal(t, "tmpl-1", body.Data[0].ID)
	mockSvc.AssertExpectations(t)
}

func TestTemplateHandlerListFailure(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("List", mock.Anything, "tenant-1").Return(nil, errors.New("mongo down"))
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodGet, "/api/v1/notification-templates", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ----------------------------------------------------------------------- Get

func TestTemplateHandlerGet(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("Get", mock.Anything, "tenant-1", "tmpl-1").Return(templateFixture(), nil)
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodGet, "/api/v1/notification-templates/tmpl-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var got models.NotificationTemplate
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "tmpl-1", got.ID)
	mockSvc.AssertExpectations(t)
}

// The service lookup is tenant-scoped, so another tenant's id surfaces as a
// plain 404 rather than revealing that it exists.
func TestTemplateHandlerGetNotFound(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("Get", mock.Anything, "tenant-1", "other-tenants-template").Return(nil, errors.New("not found"))
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodGet, "/api/v1/notification-templates/other-tenants-template", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// -------------------------------------------------------------------- Create

func TestTemplateHandlerCreate(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("Create", mock.Anything, "tenant-1", mock.AnythingOfType("*models.CreateTemplateRequest")).
		Return(templateFixture(), nil)
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodPost, "/api/v1/notification-templates", map[string]interface{}{
		"type": "welcome", "channel": "email", "name": "Welcome email", "body_template": "<p>Hello</p>",
	})

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestTemplateHandlerCreateRejectsMalformedJSON(t *testing.T) {
	mockSvc := new(MockTemplateService)
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRawRequest(router, http.MethodPost, "/api/v1/notification-templates", `{"type":`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, mockSvc.Calls)
}

func TestTemplateHandlerCreateRejectsMissingRequiredFields(t *testing.T) {
	mockSvc := new(MockTemplateService)
	router := setupTemplateRouter(mockSvc)

	// body_template is required by the binding tag.
	w := doTemplateRequest(router, http.MethodPost, "/api/v1/notification-templates", map[string]interface{}{
		"type": "welcome", "channel": "email", "name": "n",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, mockSvc.Calls)
}

// A template the service rejects (bad channel, unparsable body) is the
// caller's mistake, so it maps to 422 rather than 500.
func TestTemplateHandlerCreateValidationFailureIsUnprocessable(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("Create", mock.Anything, "tenant-1", mock.Anything).Return(nil, errors.New("invalid channel: pigeon"))
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodPost, "/api/v1/notification-templates", map[string]interface{}{
		"type": "welcome", "channel": "pigeon", "name": "n", "body_template": "b",
	})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// -------------------------------------------------------------------- Update

func TestTemplateHandlerUpdate(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("Update", mock.Anything, "tenant-1", "tmpl-1", mock.AnythingOfType("*models.UpdateTemplateRequest")).
		Return(templateFixture(), nil)
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodPut, "/api/v1/notification-templates/tmpl-1", map[string]interface{}{
		"name": "Renamed", "is_active": false,
	})

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestTemplateHandlerUpdateRejectsMalformedJSON(t *testing.T) {
	mockSvc := new(MockTemplateService)
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRawRequest(router, http.MethodPut, "/api/v1/notification-templates/tmpl-1", `{"name":`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, mockSvc.Calls)
}

// The handler splits the service's errors so the admin UI can tell "gone" from
// "your input is wrong".
func TestTemplateHandlerUpdateDistinguishesNotFoundFromValidation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{name: "not found", err: errors.New("template not found"), wantCode: http.StatusNotFound},
		{name: "validation", err: errors.New("body template invalid: unexpected EOF"), wantCode: http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(MockTemplateService)
			mockSvc.On("Update", mock.Anything, "tenant-1", "tmpl-1", mock.Anything).Return(nil, tt.err)
			router := setupTemplateRouter(mockSvc)

			w := doTemplateRequest(router, http.MethodPut, "/api/v1/notification-templates/tmpl-1", map[string]interface{}{"name": "x"})

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

// -------------------------------------------------------------------- Delete

func TestTemplateHandlerDelete(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("Delete", mock.Anything, "tenant-1", "tmpl-1").Return(nil)
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodDelete, "/api/v1/notification-templates/tmpl-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestTemplateHandlerDeleteNotFound(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("Delete", mock.Anything, "tenant-1", "ghost").Return(errors.New("not found"))
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodDelete, "/api/v1/notification-templates/ghost", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ------------------------------------------------------------------- Preview

func TestTemplateHandlerPreviewWithSampleVars(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("Preview", mock.Anything, "tenant-1", "tmpl-1", map[string]interface{}{"UserName": "Alice"}).
		Return(&models.RenderedTemplate{Subject: "Welcome, Alice!", Body: "<p>Hi Alice</p>"}, nil)
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodPost, "/api/v1/notification-templates/tmpl-1/preview", map[string]interface{}{
		"sample_vars": map[string]interface{}{"UserName": "Alice"},
	})

	assert.Equal(t, http.StatusOK, w.Code)
	var got models.RenderedTemplate
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "Welcome, Alice!", got.Subject)
	mockSvc.AssertExpectations(t)
}

// Preview is explicitly tolerant of an absent or unparsable body — the admin
// UI opens a preview before anything has been typed.
func TestTemplateHandlerPreviewToleratesEmptyBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "no body at all", body: ""},
		{name: "empty object", body: "{}"},
		{name: "malformed json", body: `{"sample_vars":`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(MockTemplateService)
			mockSvc.On("Preview", mock.Anything, "tenant-1", "tmpl-1", map[string]interface{}(nil)).
				Return(&models.RenderedTemplate{Subject: "Sample", Body: "<p>Sample</p>"}, nil)
			router := setupTemplateRouter(mockSvc)

			w := doTemplateRawRequest(router, http.MethodPost, "/api/v1/notification-templates/tmpl-1/preview", tt.body)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestTemplateHandlerPreviewRenderFailure(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("Preview", mock.Anything, "tenant-1", "tmpl-1", map[string]interface{}(nil)).
		Return(nil, errors.New("subject: executing template"))
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRawRequest(router, http.MethodPost, "/api/v1/notification-templates/tmpl-1/preview", "{}")

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// -------------------------------------------------------------- ListDefaults

// The starter-pack catalogue is global and read-only, so it needs no tenant.
func TestTemplateHandlerListDefaults(t *testing.T) {
	mockSvc := new(MockTemplateService)
	router := setupTemplateRouterWithTenant(mockSvc, "")

	w := doTemplateRequest(router, http.MethodGet, "/api/v1/notification-templates/defaults", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Data []struct {
			Type            string `json:"type"`
			Channel         string `json:"channel"`
			Name            string `json:"name"`
			SubjectTemplate string `json:"subject_template"`
			BodyTemplate    string `json:"body_template"`
		} `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotEmpty(t, body.Data, "the starter pack should not be empty")
	for _, entry := range body.Data {
		assert.NotEmpty(t, entry.Type, "every entry needs a type for the picker")
		assert.NotEmpty(t, entry.Channel)
		assert.NotEmpty(t, entry.Name)
		assert.NotEmpty(t, entry.BodyTemplate, "the picker pre-fills the editor from this")
	}
	assert.Empty(t, mockSvc.Calls, "the catalogue is static and needs no service call")
}

// ------------------------------------------------------------------ TestSend

func TestTemplateHandlerTestSend(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("TestSend", mock.Anything, "tenant-1", "tmpl-1", "admin@example.com", map[string]interface{}{"UserName": "Alice"}).
		Return(nil)
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodPost, "/api/v1/notification-templates/tmpl-1/test-send", map[string]interface{}{
		"email":       "admin@example.com",
		"sample_vars": map[string]interface{}{"UserName": "Alice"},
	})

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestTemplateHandlerTestSendRequiresEmail(t *testing.T) {
	mockSvc := new(MockTemplateService)
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodPost, "/api/v1/notification-templates/tmpl-1/test-send", map[string]interface{}{})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, mockSvc.Calls, "no send may be attempted without a recipient")
}

func TestTemplateHandlerTestSendRejectsMalformedJSON(t *testing.T) {
	mockSvc := new(MockTemplateService)
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRawRequest(router, http.MethodPost, "/api/v1/notification-templates/tmpl-1/test-send", `{"email":`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, mockSvc.Calls)
}

func TestTemplateHandlerTestSendFailure(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("TestSend", mock.Anything, "tenant-1", "tmpl-1", "admin@example.com", map[string]interface{}(nil)).
		Return(errors.New("test send failed: smtp refused"))
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRequest(router, http.MethodPost, "/api/v1/notification-templates/tmpl-1/test-send", map[string]interface{}{
		"email": "admin@example.com",
	})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestTemplateHandlerInstallDefaultsFailure(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("InstallDefaults", mock.Anything, "tenant-1", false).Return(nil, errors.New("mongo down"))
	router := setupTemplateRouter(mockSvc)

	w := doTemplateRawRequest(router, http.MethodPost, "/api/v1/notification-templates/install-defaults", "{}")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// Static segments are registered before ":id"; if that ordering regressed,
// gin would treat "defaults" as a template id and these would 404.
func TestTemplateRoutesKeepStaticSegmentsAheadOfTheIDParam(t *testing.T) {
	mockSvc := new(MockTemplateService)
	mockSvc.On("InstallDefaults", mock.Anything, "tenant-1", false).
		Return(&service.InstallDefaultsResult{Created: 1}, nil)
	router := setupTemplateRouter(mockSvc)

	defaultsResp := doTemplateRequest(router, http.MethodGet, "/api/v1/notification-templates/defaults", nil)
	assert.Equal(t, http.StatusOK, defaultsResp.Code)
	// A Get by id would have been recorded on the mock; nothing should be.
	mockSvc.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything)

	installResp := doTemplateRawRequest(router, http.MethodPost, "/api/v1/notification-templates/install-defaults", "{}")
	assert.Equal(t, http.StatusOK, installResp.Code)
}
