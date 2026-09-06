package validator

import (
	"fmt"
	"testing"

	govalidator "github.com/go-playground/validator/v10"
)

type slugged struct {
	Slug string `validate:"slug"`
}

// New() used to drop the error from RegisterValidation. An unregistered tag
// does not fail closed -- go-playground reports an unknown tag only when the
// field is validated, and the three custom tags would have silently accepted
// anything.
func TestNewRegistersCustomTags(t *testing.T) {
	v := New()

	if err := v.Validate(slugged{Slug: "Not A Slug!"}); err == nil {
		t.Fatal("Validate accepted an invalid slug; the `slug` tag is not registered")
	}
	if err := v.Validate(slugged{Slug: "a-valid-slug"}); err != nil {
		t.Errorf("Validate rejected a valid slug: %v", err)
	}
}

func TestValidateVarUsesCustomTags(t *testing.T) {
	v := New()

	for _, c := range []struct {
		tag, value string
		wantOK     bool
	}{
		{tag: "slug", value: "hello-world", wantOK: true},
		{tag: "slug", value: "Hello World", wantOK: false},
		{tag: "phone", value: "+880 1711-000000", wantOK: true},
		{tag: "phone", value: "12", wantOK: false},
		{tag: "color", value: "#a1b2c3", wantOK: true},
		{tag: "color", value: "not-a-color", wantOK: false},
	} {
		t.Run(c.tag+"/"+c.value, func(t *testing.T) {
			err := v.ValidateVar(c.value, c.tag)
			if gotOK := err == nil; gotOK != c.wantOK {
				t.Errorf("ValidateVar(%q, %q) err = %v, want ok=%v", c.value, c.tag, err, c.wantOK)
			}
		})
	}
}

// FormatValidationErrors used a bare type assertion, so a handler that added
// context to the bind error got an empty slice back -- a validation failure
// reported with no fields in it.
func TestFormatValidationErrorsUnwraps(t *testing.T) {
	v := New()
	raw := v.Validate(slugged{Slug: "Not A Slug!"})
	if raw == nil {
		t.Fatal("expected a validation error to format")
	}

	wrapped := fmt.Errorf("binding request body: %w", raw)

	got := FormatValidationErrors(wrapped)
	if len(got) == 0 {
		t.Fatal("FormatValidationErrors returned nothing for a wrapped ValidationErrors")
	}
	if got[0].Tag != "slug" {
		t.Errorf("Tag = %q, want \"slug\"", got[0].Tag)
	}
}

func TestFormatValidationErrorsOnUnrelatedError(t *testing.T) {
	if got := FormatValidationErrors(fmt.Errorf("not a validation problem")); got != nil {
		t.Errorf("FormatValidationErrors = %v, want nil", got)
	}
}

// Guards the type used in the errors.As call above from drifting.
var _ govalidator.ValidationErrors = govalidator.ValidationErrors{}
