package result_test

import (
	"errors"
	"fmt"
	"testing"

	kerrors "github.com/marcusPrado02/go-commons/kernel/errors"
	"github.com/marcusPrado02/go-commons/kernel/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOk_IsOk(t *testing.T) {
	r := result.Ok(42)
	assert.True(t, r.IsOk())
	assert.False(t, r.IsFail())
	assert.Equal(t, 42, r.Must())
}

func TestFail_IsFail(t *testing.T) {
	p := kerrors.NewProblem("ERR", kerrors.CategoryBusiness, kerrors.SeverityError, "bad")
	r := result.Fail[int](p)
	assert.True(t, r.IsFail())
	assert.False(t, r.IsOk())
	assert.Equal(t, p, r.MustProblem())
}

func TestMust_PanicsOnFail(t *testing.T) {
	p := kerrors.ErrNotFound
	r := result.Fail[string](p)
	assert.Panics(t, func() { r.Must() })
}

func TestMustProblem_PanicsOnOk(t *testing.T) {
	r := result.Ok("hello")
	assert.Panics(t, func() { r.MustProblem() })
}

// Backward-compatibility: deprecated Value() and Problem() still delegate correctly.
func TestValue_BackwardCompat(t *testing.T) {
	assert.Equal(t, 42, result.Ok(42).Value())
	assert.Panics(t, func() { result.Fail[int](kerrors.ErrNotFound).Value() })
}

func TestProblem_BackwardCompat(t *testing.T) {
	p := kerrors.ErrNotFound
	assert.Equal(t, p, result.Fail[int](p).Problem())
	assert.Panics(t, func() { result.Ok(0).Problem() })
}

func TestValueOrZero_ReturnsZeroOnFail(t *testing.T) {
	r := result.Fail[int](kerrors.ErrTechnical)
	assert.Equal(t, 0, r.ValueOrZero())
}

func TestUnwrap_Success(t *testing.T) {
	r := result.Ok("val")
	v, err := r.Unwrap()
	require.NoError(t, err)
	assert.Equal(t, "val", v)
}

func TestUnwrap_Failure(t *testing.T) {
	r := result.Fail[string](kerrors.ErrNotFound)
	v, err := r.Unwrap()
	assert.Error(t, err)
	assert.Empty(t, v)
}

func TestFromError_WithNilError(t *testing.T) {
	r := result.FromError("hello", nil)
	assert.True(t, r.IsOk())
	assert.Equal(t, "hello", r.Must())
}

func TestFromError_WithError(t *testing.T) {
	r := result.FromError("", errors.New("something went wrong"))
	assert.True(t, r.IsFail())
	assert.Equal(t, kerrors.CategoryTechnical, r.MustProblem().Category)
}

func TestFromError_WithProblem(t *testing.T) {
	p := kerrors.ErrNotFound
	r := result.FromError("", p)
	assert.True(t, r.IsFail())
	assert.Equal(t, kerrors.CategoryNotFound, r.MustProblem().Category)
}

func TestMap_TransformsValue(t *testing.T) {
	r := result.Ok(2)
	doubled := result.Map(r, func(n int) string { return fmt.Sprintf("%dx", n) })
	assert.True(t, doubled.IsOk())
	assert.Equal(t, "2x", doubled.Must())
}

func TestMap_PropagatesFail(t *testing.T) {
	r := result.Fail[int](kerrors.ErrNotFound)
	mapped := result.Map(r, func(n int) string { return "should not run" })
	assert.True(t, mapped.IsFail())
}

func TestFlatMap_ChainsSuccess(t *testing.T) {
	r := result.Ok(5)
	chained := result.FlatMap(r, func(n int) result.Result[string] {
		if n > 3 {
			return result.Ok("big")
		}
		return result.Fail[string](kerrors.ErrValidation)
	})
	assert.True(t, chained.IsOk())
	assert.Equal(t, "big", chained.Must())
}

func TestFlatMap_PropagatesFail(t *testing.T) {
	r := result.Fail[int](kerrors.ErrUnauthorized)
	chained := result.FlatMap(r, func(n int) result.Result[string] { return result.Ok("x") })
	assert.True(t, chained.IsFail())
	assert.Equal(t, kerrors.CategoryUnauthorized, chained.MustProblem().Category)
}
