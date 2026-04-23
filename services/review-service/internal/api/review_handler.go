package api

import (
	"net/http"
	"strconv"

	"github.com/ecommerce/review-service/internal/models"
	"github.com/ecommerce/review-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ReviewHandler struct {
	service service.ReviewService
	logger  *logrus.Logger
}

func NewReviewHandler(service service.ReviewService, logger *logrus.Logger) *ReviewHandler {
	return &ReviewHandler{
		service: service,
		logger:  logger,
	}
}

func (h *ReviewHandler) CreateReview(c *gin.Context) {
	var req models.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	review, err := h.service.CreateReview(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "user has already reviewed this product" {
			c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to create review")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, review)
}

func (h *ReviewHandler) GetReview(c *gin.Context) {
	id := c.Param("id")

	review, err := h.service.GetReview(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, review)
}

func (h *ReviewHandler) GetProductReviews(c *gin.Context) {
	productID := c.Param("productId")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "tenant_id query parameter is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	reviews, total, err := h.service.GetProductReviews(c.Request.Context(), tenantID, productID, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get product reviews")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListReviewsResponse{
		Data: reviews,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *ReviewHandler) GetUserReviews(c *gin.Context) {
	userID := c.Param("userId")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "tenant_id query parameter is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	reviews, total, err := h.service.GetUserReviews(c.Request.Context(), tenantID, userID, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user reviews")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListReviewsResponse{
		Data: reviews,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *ReviewHandler) UpdateReview(c *gin.Context) {
	id := c.Param("id")
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "user_id query parameter is required"})
		return
	}

	var req models.UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	review, err := h.service.UpdateReview(c.Request.Context(), id, userID, &req)
	if err != nil {
		if err.Error() == "review not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		if err.Error() == "unauthorized: you can only update your own reviews" {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, review)
}

func (h *ReviewHandler) DeleteReview(c *gin.Context) {
	id := c.Param("id")
	userID := c.Query("user_id")

	if err := h.service.DeleteReview(c.Request.Context(), id, userID); err != nil {
		if err.Error() == "review not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		if err.Error() == "unauthorized: you can only delete your own reviews" {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ReviewHandler) ModerateReview(c *gin.Context) {
	id := c.Param("id")

	var req models.ModerateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	review, err := h.service.ModerateReview(c.Request.Context(), id, &req)
	if err != nil {
		if err.Error() == "review not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, review)
}

func (h *ReviewHandler) AddHelpfulVote(c *gin.Context) {
	id := c.Param("id")

	var req models.HelpfulVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.service.AddHelpfulVote(c.Request.Context(), id, &req); err != nil {
		if err.Error() == "review not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "Vote recorded"})
}

func (h *ReviewHandler) RespondToReview(c *gin.Context) {
	id := c.Param("id")

	var req models.SellerResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	review, err := h.service.RespondToReview(c.Request.Context(), id, &req)
	if err != nil {
		if err.Error() == "review not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, review)
}

func (h *ReviewHandler) GetProductSummary(c *gin.Context) {
	productID := c.Param("productId")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "tenant_id query parameter is required"})
		return
	}

	summary, err := h.service.GetProductSummary(c.Request.Context(), tenantID, productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// RegisterRoutes sets up the review API routes
func RegisterRoutes(router *gin.Engine, handler *ReviewHandler) {
	v1 := router.Group("/api/v1")
	{
		reviews := v1.Group("/reviews")
		{
			reviews.POST("", handler.CreateReview)
			reviews.GET("/:id", handler.GetReview)
			reviews.PUT("/:id", handler.UpdateReview)
			reviews.DELETE("/:id", handler.DeleteReview)
			reviews.POST("/:id/helpful", handler.AddHelpfulVote)
			reviews.POST("/:id/moderate", handler.ModerateReview)
			reviews.POST("/:id/respond", handler.RespondToReview)
			reviews.GET("/product/:productId", handler.GetProductReviews)
			reviews.GET("/product/:productId/summary", handler.GetProductSummary)
			reviews.GET("/user/:userId", handler.GetUserReviews)
		}
	}
}

// Response types
type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type ListReviewsResponse struct {
	Data       []models.ReviewResponse `json:"data"`
	Pagination Pagination              `json:"pagination"`
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}
