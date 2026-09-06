package errors

import (
	stderrors "errors"
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

// IsAppError reports whether err is, or wraps, an *AppError.
func IsAppError(err error) bool {
	var appErr *AppError
	return stderrors.As(err, &appErr)
}

// GetAppError returns the *AppError in err's chain, or nil when there is none.
//
// It unwraps deliberately. response.Error is the single place every handler
// turns an error into a status code, and it reaches for this function first.
// A bare type assertion stopped matching as soon as any caller wrapped with
// %w, so a wrapped NotFound or TooManyRequests fell through to the 500 branch
// -- which also echoes err.Error() into the response body, exposing the whole
// wrapped chain to the client.
func GetAppError(err error) *AppError {
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr
	}
	return nil
}
