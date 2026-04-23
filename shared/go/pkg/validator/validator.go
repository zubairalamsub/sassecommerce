package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps the validator instance
type Validator struct {
	validate *validator.Validate
}

// New creates a new validator instance
func New() *Validator {
	v := validator.New()

	// Register custom validators
	v.RegisterValidation("slug", validateSlug)
	v.RegisterValidation("phone", validatePhone)
	v.RegisterValidation("color", validateColor)

	return &Validator{validate: v}
}

// Validate validates a struct
func (v *Validator) Validate(data interface{}) error {
	return v.validate.Struct(data)
}

// ValidateVar validates a single variable
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	return v.validate.Var(field, tag)
}

// ValidationError represents validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}

// FormatValidationErrors formats validator errors into a readable format
func FormatValidationErrors(err error) []ValidationError {
	var errors []ValidationError

	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrs {
			errors = append(errors, ValidationError{
				Field:   getJSONFieldName(e.Field()),
				Tag:     e.Tag(),
				Value:   fmt.Sprintf("%v", e.Value()),
				Message: getErrorMessage(e),
			})
		}
	}

	return errors
}

// Custom validators

// validateSlug validates a slug (lowercase, alphanumeric, hyphens)
func validateSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	matched, _ := regexp.MatchString(`^[a-z0-9]+(?:-[a-z0-9]+)*$`, slug)
	return matched
}

// validatePhone validates a phone number
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	// Basic phone validation (digits, +, -, spaces, parentheses)
	matched, _ := regexp.MatchString(`^[\d\s\+\-\(\)]+$`, phone)
	return matched && len(phone) >= 10
}

// validateColor validates a hex color code
func validateColor(fl validator.FieldLevel) bool {
	color := fl.Field().String()
	matched, _ := regexp.MatchString(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`, color)
	return matched
}

// Helper functions

// getJSONFieldName converts struct field name to JSON field name
func getJSONFieldName(field string) string {
	// Convert to snake_case
	var result strings.Builder
	for i, r := range field {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// getErrorMessage returns a human-readable error message
func getErrorMessage(e validator.FieldError) string {
	field := getJSONFieldName(e.Field())

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", field, e.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long", field, e.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, e.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, e.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, e.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, e.Param())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "uri":
		return fmt.Sprintf("%s must be a valid URI", field)
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "slug":
		return fmt.Sprintf("%s must be a valid slug (lowercase, alphanumeric, hyphens)", field)
	case "phone":
		return fmt.Sprintf("%s must be a valid phone number", field)
	case "color":
		return fmt.Sprintf("%s must be a valid hex color code", field)
	default:
		return fmt.Sprintf("%s failed validation (%s)", field, e.Tag())
	}
}
