package errors_test

import (
	"fmt"

	kerrors "github.com/marcusPrado02/go-commons/kernel/errors"
)

func ExampleProblem_Error() {
	fmt.Println(kerrors.ErrNotFound.Error())
	// Output:
	// [NOT_FOUND] resource not found
}

func ExampleProblem_WithCause() {
	underlying := fmt.Errorf("connection refused")
	err := kerrors.ErrTechnical.WithCause(underlying)
	fmt.Println(err.Code)
	// Output:
	// TECHNICAL
}

func ExampleErrValidation() {
	p := kerrors.ErrValidation
	fmt.Println(p.Category, p.Severity)
	// Output:
	// VALIDATION WARNING
}
