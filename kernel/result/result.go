// Package result provides a generic Result[T] type for functional pipelines.
// Port interfaces use (T, error) — Result[T] is an opt-in utility for
// cases where chaining transformations is more expressive than sequential if-err checks.
package result

import (
	stderrors "errors"

	kerrors "github.com/marcusPrado02/go-commons/kernel/errors"
)

// Result represents a computation that either succeeded with a value or failed with a Problem.
type Result[T any] struct {
	value   T
	problem *kerrors.Problem
}

// Ok creates a successful Result holding the given value.
func Ok[T any](value T) Result[T] {
	return Result[T]{value: value}
}

// Fail creates a failed Result holding the given Problem.
//
//nolint:gocritic // hugeParam: Problem is a value type by design; changing to pointer would alter the call-site API.
func Fail[T any](problem kerrors.Problem) Result[T] {
	return Result[T]{problem: &problem}
}

// FromError bridges idiomatic Go (T, error) into Result[T].
// If err is nil, returns Ok(value). If err is a Problem, wraps it directly.
// Otherwise wraps it in ErrTechnical.
func FromError[T any](value T, err error) Result[T] {
	if err == nil {
		return Ok(value)
	}
	var prob kerrors.Problem
	if stderrors.As(err, &prob) {
		return Fail[T](prob)
	}
	return Fail[T](kerrors.ErrTechnical.WithCause(err))
}

// IsOk returns true if the Result holds a value.
func (r Result[T]) IsOk() bool { return r.problem == nil }

// IsFail returns true if the Result holds a Problem.
func (r Result[T]) IsFail() bool { return r.problem != nil }

// Must returns the held value. Panics if IsFail().
// The Must prefix follows Go convention (regexp.MustCompile, template.Must) to signal
// that this method panics — making the risk visible at the call site.
// Prefer Or, OrElse, or Unwrap for safe extraction.
func (r Result[T]) Must() T {
	if r.IsFail() {
		panic("result: called Must() on a failed Result — check IsOk() first")
	}
	return r.value
}

// Value returns the held value. Panics if IsFail().
//
// Deprecated: Use Must() instead. Must() is identical but follows the Go convention
// of using "Must" to signal a panicking method (regexp.MustCompile, template.Must).
func (r Result[T]) Value() T { return r.Must() }

// ValueOrZero returns the held value, or the zero value of T if IsFail().
func (r Result[T]) ValueOrZero() T { return r.value }

// MustProblem returns the held Problem. Panics if IsOk().
// The Must prefix signals that this method panics — prefer checking IsFail() first.
func (r Result[T]) MustProblem() kerrors.Problem {
	if r.IsOk() {
		panic("result: called MustProblem() on a successful Result — check IsFail() first")
	}
	return *r.problem
}

// Problem returns the held Problem. Panics if IsOk().
//
// Deprecated: Use MustProblem() instead. MustProblem() is identical but follows the Go
// convention of using "Must" to signal a panicking method.
func (r Result[T]) Problem() kerrors.Problem { return r.MustProblem() }

// Unwrap returns (value, nil) on success or (zero, problem) on failure.
// Use this when integrating with code that expects idiomatic (T, error).
func (r Result[T]) Unwrap() (T, error) {
	if r.IsFail() {
		var zero T
		return zero, *r.problem
	}
	return r.value, nil
}

// Or returns the held value on success, or fallback if failed.
func (r Result[T]) Or(fallback T) T {
	if r.IsFail() {
		return fallback
	}
	return r.value
}

// OrElse returns the held value on success, or the result of calling f if failed.
// Use this instead of Or when the fallback value is expensive to compute.
func (r Result[T]) OrElse(f func() T) T {
	if r.IsFail() {
		return f()
	}
	return r.value
}

// Map transforms a successful Result[T] into Result[U] by applying f.
// If r is failed, the failure propagates unchanged.
func Map[T, U any](r Result[T], f func(T) U) Result[U] {
	if r.IsFail() {
		return Fail[U](r.MustProblem())
	}
	return Ok(f(r.value))
}

// FlatMap chains a successful Result[T] with a function returning Result[U].
// If r is failed, the failure propagates unchanged and f is never called.
func FlatMap[T, U any](r Result[T], f func(T) Result[U]) Result[U] {
	if r.IsFail() {
		return Fail[U](r.MustProblem())
	}
	return f(r.value)
}
