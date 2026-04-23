package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application error
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
	Details interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Common error codes
const (
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeValidation          = "VALIDATION_ERROR"
	ErrCodeInternal            = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	ErrCodeTooManyRequests     = "TOO_MANY_REQUESTS"
	ErrCodeUnprocessableEntity = "UNPROCESSABLE_ENTITY"
)

// NewAppError creates a new application error
func NewAppError(code, message string, status int) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return e
}

// Common errors
func BadRequest(message string) *AppError {
	return NewAppError(ErrCodeBadRequest, message, http.StatusBadRequest)
}

func Unauthorized(message string) *AppError {
	if message == "" {
		message = "Unauthorized access"
	}
	return NewAppError(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

func Forbidden(message string) *AppError {
	if message == "" {
		message = "Access forbidden"
	}
	return NewAppError(ErrCodeForbidden, message, http.StatusForbidden)
}

func NotFound(resource string) *AppError {
	return NewAppError(ErrCodeNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound)
}

func Conflict(message string) *AppError {
	return NewAppError(ErrCodeConflict, message, http.StatusConflict)
}

func ValidationError(message string) *AppError {
	return NewAppError(ErrCodeValidation, message, http.StatusUnprocessableEntity)
}

func Internal(message string) *AppError {
	if message == "" {
		message = "Internal server error"
	}
	return NewAppError(ErrCodeInternal, message, http.StatusInternalServerError)
}

func ServiceUnavailable(service string) *AppError {
	return NewAppError(ErrCodeServiceUnavailable, fmt.Sprintf("%s service unavailable", service), http.StatusServiceUnavailable)
}

func TooManyRequests(message string) *AppError {
	if message == "" {
		message = "Too many requests"
	}
	return NewAppError(ErrCodeTooManyRequests, message, http.StatusTooManyRequests)
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError extracts AppError from error
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return nil
}
