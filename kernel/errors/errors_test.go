package errors_test

import (
	stderrors "errors"
	"testing"

	"github.com/marcusPrado02/go-commons/kernel/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewErrorCode_valid(t *testing.T) {
	code, err := errors.NewErrorCode("USER_NOT_FOUND")
	require.NoError(t, err)
	assert.Equal(t, errors.ErrorCode("USER_NOT_FOUND"), code)
}

func TestNewErrorCode_empty(t *testing.T) {
	_, err := errors.NewErrorCode("")
	assert.Error(t, err)
}

func TestProblem_Error(t *testing.T) {
	p := errors.NewProblem("TEST_CODE", errors.CategoryBusiness, errors.SeverityError, "test message")
	assert.Equal(t, "[TEST_CODE] test message", p.Error())
}

func TestProblem_ErrorWithCause(t *testing.T) {
	cause := stderrors.New("underlying error")
	p := errors.NewProblem("TEST_CODE", errors.CategoryTechnical, errors.SeverityError, "wrapped").
		WithCause(cause)
	assert.Contains(t, p.Error(), "underlying error")
	assert.Equal(t, cause, stderrors.Unwrap(p))
}

func TestProblem_WithDetail(t *testing.T) {
	p := errors.NewProblem("CODE", errors.CategoryValidation, errors.SeverityWarning, "msg")
	p2 := p.WithDetail("field", "email")

	// original is unchanged
	assert.Empty(t, p.Details)
	// copy has the detail
	assert.Equal(t, "email", p2.Details["field"])

	// chaining: p2 already has a detail — covers the non-empty map branch
	p3 := p2.WithDetail("count", 3)
	assert.Equal(t, "email", p3.Details["field"])
	assert.Equal(t, 3, p3.Details["count"])
	assert.NotContains(t, p2.Details, "count") // p2 unchanged
}

func TestProblem_WithDetails_merges(t *testing.T) {
	p := errors.NewProblem("CODE", errors.CategoryValidation, errors.SeverityWarning, "msg").
		WithDetail("a", 1)
	p2 := p.WithDetails(map[string]any{"b": 2})

	assert.Equal(t, 1, p2.Details["a"])
	assert.Equal(t, 2, p2.Details["b"])
	assert.NotContains(t, p.Details, "b")
}

func TestProblem_ImplementsError(t *testing.T) {
	var err error = errors.NewProblem("CODE", errors.CategoryBusiness, errors.SeverityError, "msg")
	assert.NotNil(t, err)
}

func TestSentinelErrors_defined(t *testing.T) {
	assert.Equal(t, errors.ErrorCode("NOT_FOUND"), errors.ErrNotFound.Code)
	assert.Equal(t, errors.ErrorCode("UNAUTHORIZED"), errors.ErrUnauthorized.Code)
	assert.Equal(t, errors.ErrorCode("VALIDATION"), errors.ErrValidation.Code)
	assert.Equal(t, errors.ErrorCode("TECHNICAL"), errors.ErrTechnical.Code)
}
