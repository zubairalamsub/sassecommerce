package api

import (
	"net/http"
	"strconv"

	"github.com/ecommerce/tenant-service/internal/models"
	"github.com/ecommerce/tenant-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type TenantHandler struct {
	service service.TenantService
	logger  *logrus.Logger
}

func NewTenantHandler(service service.TenantService, logger *logrus.Logger) *TenantHandler {
	return &TenantHandler{
		service: service,
		logger:  logger,
	}
}

// CreateTenant godoc
// @Summary Create a new tenant
// @Description Create a new tenant with the provided information
// @Tags tenants
// @Accept json
// @Produce json
// @Param request body models.CreateTenantRequest true "Tenant creation request"
// @Success 201 {object} models.TenantResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tenants [post]
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req models.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	tenant, err := h.service.CreateTenant(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create tenant")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tenant)
}

// GetTenant godoc
// @Summary Get a tenant by ID
// @Description Get detailed information about a tenant
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 200 {object} models.TenantResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tenants/{id} [get]
func (h *TenantHandler) GetTenant(c *gin.Context) {
	id := c.Param("id")

	tenant, err := h.service.GetTenant(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// GetTenantBySlug godoc
// @Summary Get a tenant by slug
// @Description Get tenant information by slug
// @Tags tenants
// @Accept json
// @Produce json
// @Param slug path string true "Tenant Slug"
// @Success 200 {object} models.TenantResponse
// @Failure 404 {object} ErrorResponse
// @Router /tenants/slug/{slug} [get]
func (h *TenantHandler) GetTenantBySlug(c *gin.Context) {
	slug := c.Param("slug")

	tenant, err := h.service.GetTenantBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// GetTenantByDomain godoc
// @Summary Get a tenant by custom domain
// @Description Get tenant information by custom domain
// @Tags tenants
// @Accept json
// @Produce json
// @Param domain query string true "Custom Domain"
// @Success 200 {object} models.TenantResponse
// @Failure 404 {object} ErrorResponse
// @Router /tenants/domain [get]
func (h *TenantHandler) GetTenantByDomain(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "domain query parameter is required"})
		return
	}

	tenant, err := h.service.GetTenantByDomain(c.Request.Context(), domain)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// ListTenants godoc
// @Summary List all tenants
// @Description Get a paginated list of all tenants
// @Tags tenants
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} ListTenantsResponse
// @Failure 500 {object} ErrorResponse
// @Router /tenants [get]
func (h *TenantHandler) ListTenants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	tenants, total, err := h.service.ListTenants(c.Request.Context(), page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list tenants")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListTenantsResponse{
		Data: tenants,
		Pagination: Pagination{
			Page:      page,
			PageSize:  pageSize,
			Total:     total,
			TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// UpdateTenant godoc
// @Summary Update a tenant
// @Description Update tenant information
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Param request body models.UpdateTenantRequest true "Tenant update request"
// @Success 200 {object} models.TenantResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tenants/{id} [put]
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	tenant, err := h.service.UpdateTenant(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update tenant")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// UpdateTenantConfig godoc
// @Summary Update tenant configuration
// @Description Update tenant-specific configuration
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Param config body models.TenantConfig true "Tenant configuration"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /tenants/{id}/config [patch]
func (h *TenantHandler) UpdateTenantConfig(c *gin.Context) {
	id := c.Param("id")

	var config models.TenantConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.service.UpdateTenantConfig(c.Request.Context(), id, &config); err != nil {
		h.logger.WithError(err).Error("Failed to update tenant config")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "Configuration updated successfully"})
}

// DeleteTenant godoc
// @Summary Delete a tenant
// @Description Soft delete a tenant
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tenants/{id} [delete]
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteTenant(c.Request.Context(), id); err != nil {
		h.logger.WithError(err).Error("Failed to delete tenant")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// Response types
type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type ListTenantsResponse struct {
	Data       []models.TenantResponse `json:"data"`
	Pagination Pagination              `json:"pagination"`
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}
