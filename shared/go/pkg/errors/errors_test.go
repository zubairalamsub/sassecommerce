package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"
)

// The bug these tests pin down: IsAppError and GetAppError used a bare type
// assertion, so they stopped recognising an AppError the moment any caller
// wrapped it with %w. response.Error asks GetAppError for the status code
// first, so a wrapped NotFound was served as 500 -- and the 500 branch echoes
// err.Error() into the response body, handing the client the whole wrapped
// chain.
func TestGetAppErrorFindsWrappedError(t *testing.T) {
	want := NotFound("order")

	err := fmt.Errorf("loading order for tenant: %w", want)

	got := GetAppError(err)
	if got == nil {
		t.Fatal("GetAppError returned nil for a wrapped AppError; response.Error would serve this as 500")
	}
	if got.Status != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", got.Status, http.StatusNotFound)
	}
	if got.Code != ErrCodeNotFound {
		t.Errorf("Code = %q, want %q", got.Code, ErrCodeNotFound)
	}
}

func TestGetAppErrorFindsDeeplyWrappedError(t *testing.T) {
	want := TooManyRequests("slow down")

	err := fmt.Errorf("handler: %w", fmt.Errorf("service: %w", fmt.Errorf("repo: %w", want)))

	got := GetAppError(err)
	if got == nil {
		t.Fatal("GetAppError returned nil three levels down")
	}
	if got.Status != http.StatusTooManyRequests {
		t.Errorf("Status = %d, want %d", got.Status, http.StatusTooManyRequests)
	}
}

func TestIsAppErrorFindsWrappedError(t *testing.T) {
	if !IsAppError(fmt.Errorf("context: %w", Conflict("duplicate sku"))) {
		t.Error("IsAppError = false for a wrapped AppError")
	}
}

func TestGetAppErrorOnUnwrappedError(t *testing.T) {
	want := BadRequest("missing tenant")

	got := GetAppError(want)
	if got != want {
		t.Errorf("GetAppError(%v) = %v, want the same pointer back", want, got)
	}
}

func TestGetAppErrorReturnsNilForOtherErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{name: "nil", err: nil},
		{name: "plain error", err: stderrors.New("boom")},
		{name: "wrapped plain error", err: fmt.Errorf("ctx: %w", stderrors.New("boom"))},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GetAppError(c.err); got != nil {
				t.Errorf("GetAppError = %v, want nil", got)
			}
			if IsAppError(c.err) {
				t.Error("IsAppError = true, want false")
			}
		})
	}
}

// %w only wraps when the value is an error; errors.As must not be fooled by a
// value merely formatted into the message.
//
// The %v below is the point of the test, so errorlint is silenced on it --
// `golangci-lint run --fix` rewrote it to %w once already, which quietly
// inverted what this asserts while leaving the name and comment intact.
func TestGetAppErrorIgnoresFormattedNotWrapped(t *testing.T) {
	err := fmt.Errorf("context: %v", NotFound("order")) //nolint:errorlint // %v is deliberate: this asserts non-wrapping

	if got := GetAppError(err); got != nil {
		t.Errorf("GetAppError = %v, want nil for a %%v-formatted error", got)
	}
}

// errors.As walks a multi-error join too, which is how a batch handler
// aggregates per-item failures.
func TestGetAppErrorFindsErrorInJoin(t *testing.T) {
	err := stderrors.Join(stderrors.New("first"), Forbidden("not your tenant"))

	got := GetAppError(err)
	if got == nil {
		t.Fatal("GetAppError returned nil for a joined AppError")
	}
	if got.Status != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", got.Status, http.StatusForbidden)
	}
}
