package errors

import (
	"errors"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{
		Field:   "email",
		Message: "invalid format",
	}

	expected := "validation error on field 'email': invalid format"
	if err.Error() != expected {
		t.Errorf("Expected %s, got %s", expected, err.Error())
	}
}

func TestMultiError_Error(t *testing.T) {
	tests := []struct {
		name     string
		errors   []error
		expected string
	}{
		{
			name:     "No errors",
			errors:   []error{},
			expected: "no errors",
		},
		{
			name:     "Single error",
			errors:   []error{errors.New("first error")},
			expected: "first error",
		},
		{
			name:     "Multiple errors",
			errors:   []error{errors.New("first error"), errors.New("second error")},
			expected: "first error (and 1 more errors)",
		},
		{
			name: "Three errors",
			errors: []error{
				errors.New("first error"),
				errors.New("second error"),
				errors.New("third error"),
			},
			expected: "first error (and 2 more errors)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiErr := MultiError{Errors: tt.errors}
			result := multiErr.Error()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMultiError_Add(t *testing.T) {
	multiErr := &MultiError{}

	// Add nil error - should not be added
	multiErr.Add(nil)
	if len(multiErr.Errors) != 0 {
		t.Errorf("Expected 0 errors after adding nil, got %d", len(multiErr.Errors))
	}

	// Add real error
	err1 := errors.New("first error")
	multiErr.Add(err1)
	if len(multiErr.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(multiErr.Errors))
	}

	// Add another error
	err2 := errors.New("second error")
	multiErr.Add(err2)
	if len(multiErr.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(multiErr.Errors))
	}

	// Check errors are in correct order
	if multiErr.Errors[0] != err1 {
		t.Error("First error not in correct position")
	}
	if multiErr.Errors[1] != err2 {
		t.Error("Second error not in correct position")
	}
}

func TestMultiError_HasErrors(t *testing.T) {
	multiErr := &MultiError{}

	// No errors initially
	if multiErr.HasErrors() {
		t.Error("Expected HasErrors to return false for empty MultiError")
	}

	// Add an error
	multiErr.Add(errors.New("test error"))
	if !multiErr.HasErrors() {
		t.Error("Expected HasErrors to return true after adding error")
	}
}

func TestDatabaseError_Error(t *testing.T) {
	originalErr := errors.New("connection failed")
	dbErr := DatabaseError{
		Operation: "query",
		Err:       originalErr,
	}

	expected := "database error during query: connection failed"
	if dbErr.Error() != expected {
		t.Errorf("Expected %s, got %s", expected, dbErr.Error())
	}
}

func TestDatabaseError_Unwrap(t *testing.T) {
	originalErr := errors.New("connection failed")
	dbErr := DatabaseError{
		Operation: "query",
		Err:       originalErr,
	}

	unwrapped := dbErr.Unwrap()
	if unwrapped != originalErr {
		t.Error("Expected Unwrap to return original error")
	}
}

func TestPipelineError_Error(t *testing.T) {
	originalErr := errors.New("fetch failed")
	pipelineErr := PipelineError{
		Source: "rss-feed",
		Stage:  "fetch",
		Err:    originalErr,
	}

	expected := "pipeline error in rss-feed at stage fetch: fetch failed"
	if pipelineErr.Error() != expected {
		t.Errorf("Expected %s, got %s", expected, pipelineErr.Error())
	}
}

func TestPipelineError_Unwrap(t *testing.T) {
	originalErr := errors.New("fetch failed")
	pipelineErr := PipelineError{
		Source: "rss-feed",
		Stage:  "fetch",
		Err:    originalErr,
	}

	unwrapped := pipelineErr.Unwrap()
	if unwrapped != originalErr {
		t.Error("Expected Unwrap to return original error")
	}
}

func TestErrorConstants(t *testing.T) {
	// Test that error constants are defined
	errorConstants := []error{
		ErrNotFound,
		ErrInvalidInput,
		ErrUnauthorized,
		ErrForbidden,
		ErrConflict,
		ErrRateLimit,
		ErrServiceUnavailable,
		ErrTimeout,
		ErrNotImplemented,
	}

	for i, err := range errorConstants {
		if err == nil {
			t.Errorf("Error constant at index %d is nil", i)
		}
		if err.Error() == "" {
			t.Errorf("Error constant at index %d has empty message", i)
		}
	}
}