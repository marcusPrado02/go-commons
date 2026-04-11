// Package errors provides domain error types for go-commons.
// It defines Problem, DomainError, and pre-defined sentinel errors.
package errors

import (
	"errors"
	"fmt"
)

// ErrorCode identifies a specific domain error condition.
type ErrorCode string

// NewErrorCode validates and creates an ErrorCode.
func NewErrorCode(code string) (ErrorCode, error) {
	if code == "" {
		return "", errors.New("error code cannot be empty")
	}
	return ErrorCode(code), nil
}

// ErrorCategory classifies the nature of a domain error.
type ErrorCategory string

const (
	CategoryValidation   ErrorCategory = "VALIDATION"
	CategoryBusiness     ErrorCategory = "BUSINESS"
	CategoryTechnical    ErrorCategory = "TECHNICAL"
	CategoryNotFound     ErrorCategory = "NOT_FOUND"
	CategoryUnauthorized ErrorCategory = "UNAUTHORIZED"
)

// Severity indicates how critical a domain error is.
type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityError    Severity = "ERROR"
	SeverityCritical Severity = "CRITICAL"
)

// Problem is an immutable, rich domain error. Use the With* builders to add context.
// All With* methods return a new copy — the receiver is never modified.
//
//nolint:errname // "Problem" is an intentional domain name; "ProblemError" would be redundant.
type Problem struct {
	Code     ErrorCode
	Category ErrorCategory
	Severity Severity
	Message  string
	// Details is a defensive copy — safe to read, not to mutate.
	Details map[string]any
	// Cause is the underlying error, preserved for logging and errors.Is/As chaining.
	Cause error
}

// NewProblem creates a Problem with an empty Details map.
func NewProblem(code ErrorCode, category ErrorCategory, severity Severity, message string) Problem {
	return Problem{
		Code:     code,
		Category: category,
		Severity: severity,
		Message:  message,
		Details:  make(map[string]any),
	}
}

// Error implements the error interface.
//
//nolint:gocritic // hugeParam: Problem is a value type by design; pointer receivers would break error interface callers.
func (p Problem) Error() string {
	if p.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", p.Code, p.Message, p.Cause)
	}
	return fmt.Sprintf("[%s] %s", p.Code, p.Message)
}

// Unwrap returns the cause, enabling errors.Is/As chaining.
//
//nolint:gocritic // hugeParam: value receiver intentional — see Error().
func (p Problem) Unwrap() error { return p.Cause }

// WithDetail returns a new Problem with the key-value pair added to Details.
//
//nolint:gocritic // hugeParam: value receiver intentional — see Error().
func (p Problem) WithDetail(key string, value any) Problem {
	details := make(map[string]any, len(p.Details)+1)
	for k, v := range p.Details {
		details[k] = v
	}
	details[key] = value
	p.Details = details
	return p
}

// WithDetails returns a new Problem with the given map merged into Details.
//
//nolint:gocritic // hugeParam: value receiver intentional — see Error().
func (p Problem) WithDetails(extra map[string]any) Problem {
	merged := make(map[string]any, len(p.Details)+len(extra))
	for k, v := range p.Details {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	p.Details = merged
	return p
}

// WithCause returns a new Problem with the given cause attached.
//
//nolint:gocritic // hugeParam: value receiver intentional — see Error().
func (p Problem) WithCause(err error) Problem {
	p.Cause = err
	return p
}

// DomainError is the interface implemented by errors returned from ports.
// Adapters wrap SDK errors into DomainError before returning them.
// No type in this package implements DomainError — Problem is a concrete value
// type; adapters define their own DomainError implementations.
type DomainError interface {
	error
	Code() ErrorCode
	Category() ErrorCategory
	Severity() Severity
	Details() map[string]any
	Unwrap() error
}

// Pre-defined sentinel errors for common domain conditions.
var (
	ErrNotFound     = NewProblem("NOT_FOUND", CategoryNotFound, SeverityError, "resource not found")
	ErrUnauthorized = NewProblem("UNAUTHORIZED", CategoryUnauthorized, SeverityWarning, "unauthorized access")
	ErrValidation   = NewProblem("VALIDATION", CategoryValidation, SeverityWarning, "validation failed")
	ErrTechnical    = NewProblem("TECHNICAL", CategoryTechnical, SeverityError, "technical error")
)
