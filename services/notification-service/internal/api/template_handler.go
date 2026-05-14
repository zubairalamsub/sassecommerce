package api

import (
	"net/http"
	"strings"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/service"
	deftpl "github.com/ecommerce/notification-service/internal/templates"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// TemplateHandler exposes CRUD + preview/test-send endpoints for
// admin-managed notification templates. Routes live under
// /api/v1/notification-templates and require an authenticated admin
// (the auth middleware is applied at the router level).
type TemplateHandler struct {
	service service.TemplateService
	logger  *logrus.Logger
}

func NewTemplateHandler(s service.TemplateService, logger *logrus.Logger) *TemplateHandler {
	return &TemplateHandler{service: s, logger: logger}
}

// tenantIDFrom resolves the tenant the request operates on. We require
// X-Tenant-Id (set by the admin shell) but also tolerate a ?tenant_id
// query param for convenience when hitting the API from tooling.
func tenantIDFrom(c *gin.Context) string {
	if v := c.GetHeader("X-Tenant-Id"); v != "" {
		return v
	}
	if v := c.GetHeader("X-Tenant-ID"); v != "" {
		return v
	}
	return strings.TrimSpace(c.Query("tenant_id"))
}

func (h *TemplateHandler) List(c *gin.Context) {
	tenantID := tenantIDFrom(c)
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "X-Tenant-Id header is required"})
		return
	}
	items, err := h.service.List(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list templates")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *TemplateHandler) Get(c *gin.Context) {
	id := c.Param("id")
	t, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *TemplateHandler) Create(c *gin.Context) {
	tenantID := tenantIDFrom(c)
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "X-Tenant-Id header is required"})
		return
	}
	var req models.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	t, err := h.service.Create(c.Request.Context(), tenantID, &req)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (h *TemplateHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	t, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		// Distinguish not-found from validation errors so the admin UI can
		// surface useful messages.
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *TemplateHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Message: "Template deleted"})
}

func (h *TemplateHandler) Preview(c *gin.Context) {
	id := c.Param("id")
	var req models.PreviewTemplateRequest
	// Empty body is OK; preview will fall back to defaults.
	_ = c.ShouldBindJSON(&req)
	rendered, err := h.service.Preview(c.Request.Context(), id, req.SampleVars)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, rendered)
}

// ListDefaults returns the read-only catalogue of starter-pack templates.
// The admin UI's "Start from template" picker calls this so authors can
// pre-fill the editor with a known-good HTML body and tweak from there. No
// tenant scoping is needed — the catalogue is global.
func (h *TemplateHandler) ListDefaults(c *gin.Context) {
	defaults := deftpl.Defaults()
	// Shape the response so the frontend can render a select directly.
	type entry struct {
		Type            string `json:"type"`
		Channel         string `json:"channel"`
		Name            string `json:"name"`
		SubjectTemplate string `json:"subject_template"`
		BodyTemplate    string `json:"body_template"`
	}
	out := make([]entry, 0, len(defaults))
	for _, d := range defaults {
		out = append(out, entry{
			Type:            string(d.Type),
			Channel:         string(d.Channel),
			Name:            d.Name,
			SubjectTemplate: d.SubjectTemplate,
			BodyTemplate:    d.BodyTemplate,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// InstallDefaultsRequest carries the optional force flag for /install-defaults.
// Body is allowed to be empty; "force" defaults to false in that case.
type InstallDefaultsRequest struct {
	Force bool `json:"force"`
}

// InstallDefaults seeds the calling tenant with the starter-pack of
// pre-designed templates. Pass `{ "force": true }` to overwrite existing
// templates that share the same (type, channel); otherwise duplicates are
// skipped and counted in the response.
func (h *TemplateHandler) InstallDefaults(c *gin.Context) {
	tenantID := tenantIDFrom(c)
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "X-Tenant-Id header is required"})
		return
	}
	var req InstallDefaultsRequest
	// Empty body is fine — defaults to force=false.
	_ = c.ShouldBindJSON(&req)

	result, err := h.service.InstallDefaults(c.Request.Context(), tenantID, req.Force)
	if err != nil {
		h.logger.WithError(err).Error("Failed to install default templates")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *TemplateHandler) TestSend(c *gin.Context) {
	id := c.Param("id")
	var req models.TestSendTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	if err := h.service.TestSend(c.Request.Context(), id, req.Email, req.SampleVars); err != nil {
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, SuccessResponse{Message: "Test email sent"})
}

// RegisterTemplateRoutes wires the template endpoints onto an existing
// router. Kept separate from RegisterRoutes so callers can selectively expose
// templates only on the admin port if/when we split traffic.
func RegisterTemplateRoutes(router *gin.Engine, h *TemplateHandler) {
	v1 := router.Group("/api/v1")
	{
		tpl := v1.Group("/notification-templates")
		{
			tpl.GET("", h.List)
			// Static routes must be declared before ":id" or gin treats
			// "install-defaults"/"defaults" as a template ID and 404s the request.
			tpl.GET("/defaults", h.ListDefaults)
			tpl.POST("/install-defaults", h.InstallDefaults)
			tpl.GET("/:id", h.Get)
			tpl.POST("", h.Create)
			tpl.PUT("/:id", h.Update)
			tpl.DELETE("/:id", h.Delete)
			tpl.POST("/:id/preview", h.Preview)
			tpl.POST("/:id/test-send", h.TestSend)
		}
	}
}
