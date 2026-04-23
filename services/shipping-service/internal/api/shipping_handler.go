package api

import (
	"net/http"
	"strconv"

	"github.com/ecommerce/shipping-service/internal/models"
	"github.com/ecommerce/shipping-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ShippingHandler struct {
	service service.ShippingService
	logger  *logrus.Logger
}

func NewShippingHandler(service service.ShippingService, logger *logrus.Logger) *ShippingHandler {
	return &ShippingHandler{
		service: service,
		logger:  logger,
	}
}

func (h *ShippingHandler) CreateShipment(c *gin.Context) {
	var req models.CreateShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	shipment, err := h.service.CreateShipment(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create shipment")
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, shipment)
}

func (h *ShippingHandler) GetShipment(c *gin.Context) {
	id := c.Param("id")

	shipment, err := h.service.GetShipment(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, shipment)
}

func (h *ShippingHandler) GetShipmentByTracking(c *gin.Context) {
	trackingNumber := c.Param("trackingNumber")

	shipment, err := h.service.GetShipmentByTracking(c.Request.Context(), trackingNumber)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, shipment)
}

func (h *ShippingHandler) GetShipmentByOrderID(c *gin.Context) {
	orderID := c.Param("orderId")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "tenant_id query parameter is required"})
		return
	}

	shipment, err := h.service.GetShipmentByOrderID(c.Request.Context(), tenantID, orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, shipment)
}

func (h *ShippingHandler) ListShipments(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "tenant_id query parameter is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	shipments, total, err := h.service.ListShipments(c.Request.Context(), tenantID, page, pageSize, status)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list shipments")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListShipmentsResponse{
		Data: shipments,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *ShippingHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	shipment, err := h.service.UpdateStatus(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update shipment status")
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, shipment)
}

func (h *ShippingHandler) CancelShipment(c *gin.Context) {
	id := c.Param("id")

	shipment, err := h.service.CancelShipment(c.Request.Context(), id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to cancel shipment")
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, shipment)
}

func (h *ShippingHandler) CalculateRates(c *gin.Context) {
	var req models.CalculateRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	rates, err := h.service.CalculateRates(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to calculate rates")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, rates)
}

// RegisterRoutes sets up the shipping API routes
func RegisterRoutes(router *gin.Engine, handler *ShippingHandler) {
	v1 := router.Group("/api/v1")
	{
		shipments := v1.Group("/shipments")
		{
			shipments.POST("", handler.CreateShipment)
			shipments.GET("", handler.ListShipments)
			shipments.GET("/:id", handler.GetShipment)
			shipments.GET("/tracking/:trackingNumber", handler.GetShipmentByTracking)
			shipments.GET("/order/:orderId", handler.GetShipmentByOrderID)
			shipments.PUT("/:id/status", handler.UpdateStatus)
			shipments.POST("/:id/cancel", handler.CancelShipment)
		}

		v1.POST("/rates", handler.CalculateRates)
	}
}

// Response types
type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type ListShipmentsResponse struct {
	Data       []models.ShipmentResponse `json:"data"`
	Pagination Pagination                `json:"pagination"`
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}
