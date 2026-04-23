package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ecommerce/shared/go/pkg/errors"
)

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Success   bool        `json:"success"`
	Error     string      `json:"error"`
	Code      string      `json:"code,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination holds pagination metadata
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// Success sends a successful response
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMessage sends a successful response with a message
func SuccessWithMessage(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// Created sends a 201 Created response
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Paginated sends a paginated response
func Paginated(c *gin.Context, data interface{}, page, pageSize int, totalItems int64) {
	totalPages := int(totalItems) / pageSize
	if int(totalItems)%pageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    data,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	})
}

// Error sends an error response
func Error(c *gin.Context, err error) {
	// Check if it's an AppError
	if appErr := errors.GetAppError(err); appErr != nil {
		c.JSON(appErr.Status, ErrorResponse{
			Success:   false,
			Error:     appErr.Message,
			Code:      appErr.Code,
			Details:   appErr.Details,
			RequestID: getRequestID(c),
		})
		return
	}

	// Default to internal server error
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Success:   false,
		Error:     err.Error(),
		Code:      errors.ErrCodeInternal,
		RequestID: getRequestID(c),
	})
}

// BadRequest sends a 400 Bad Request response
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Success:   false,
		Error:     message,
		Code:      errors.ErrCodeBadRequest,
		RequestID: getRequestID(c),
	})
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "Unauthorized access"
	}
	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Success:   false,
		Error:     message,
		Code:      errors.ErrCodeUnauthorized,
		RequestID: getRequestID(c),
	})
}

// Forbidden sends a 403 Forbidden response
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "Access forbidden"
	}
	c.JSON(http.StatusForbidden, ErrorResponse{
		Success:   false,
		Error:     message,
		Code:      errors.ErrCodeForbidden,
		RequestID: getRequestID(c),
	})
}

// NotFound sends a 404 Not Found response
func NotFound(c *gin.Context, resource string) {
	c.JSON(http.StatusNotFound, ErrorResponse{
		Success:   false,
		Error:     resource + " not found",
		Code:      errors.ErrCodeNotFound,
		RequestID: getRequestID(c),
	})
}

// Conflict sends a 409 Conflict response
func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, ErrorResponse{
		Success:   false,
		Error:     message,
		Code:      errors.ErrCodeConflict,
		RequestID: getRequestID(c),
	})
}

// ValidationError sends a 422 Unprocessable Entity response
func ValidationError(c *gin.Context, details interface{}) {
	c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
		Success:   false,
		Error:     "Validation failed",
		Code:      errors.ErrCodeValidation,
		Details:   details,
		RequestID: getRequestID(c),
	})
}

// InternalError sends a 500 Internal Server Error response
func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = "Internal server error"
	}
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Success:   false,
		Error:     message,
		Code:      errors.ErrCodeInternal,
		RequestID: getRequestID(c),
	})
}

// Helper to get request ID from context
func getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
