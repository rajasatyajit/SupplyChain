package errors

import (
	"errors"
	"fmt"
)

// Application-specific errors
var (
	ErrNotFound           = errors.New("resource not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrConflict           = errors.New("resource conflict")
	ErrRateLimit          = errors.New("rate limit exceeded")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrTimeout            = errors.New("operation timeout")
	ErrNotImplemented     = errors.New("not implemented")
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// MultiError represents multiple errors
type MultiError struct {
	Errors []error `json:"errors"`
}

func (e MultiError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", e.Errors[0].Error(), len(e.Errors)-1)
}

// Add adds an error to the MultiError
func (e *MultiError) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// HasErrors returns true if there are any errors
func (e *MultiError) HasErrors() bool {
	return len(e.Errors) > 0
}

// DatabaseError represents a database-related error
type DatabaseError struct {
	Operation string
	Err       error
}

func (e DatabaseError) Error() string {
	return fmt.Sprintf("database error during %s: %v", e.Operation, e.Err)
}

func (e DatabaseError) Unwrap() error {
	return e.Err
}

// PipelineError represents a pipeline-related error
type PipelineError struct {
	Source string
	Stage  string
	Err    error
}

func (e PipelineError) Error() string {
	return fmt.Sprintf("pipeline error in %s at stage %s: %v", e.Source, e.Stage, e.Err)
}

func (e PipelineError) Unwrap() error {
	return e.Err
}