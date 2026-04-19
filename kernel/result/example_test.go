package result_test

import (
	"fmt"

	kerrors "github.com/marcusPrado02/go-commons/kernel/errors"
	"github.com/marcusPrado02/go-commons/kernel/result"
)

func ExampleOk() {
	r := result.Ok(42)
	fmt.Println(r.IsOk(), r.Value())
	// Output:
	// true 42
}

func ExampleFail() {
	r := result.Fail[int](kerrors.ErrNotFound)
	fmt.Println(r.IsFail())
	// Output:
	// true
}

func ExampleResult_Or() {
	ok := result.Ok("hello")
	failed := result.Fail[string](kerrors.ErrNotFound)

	fmt.Println(ok.Or("default"))
	fmt.Println(failed.Or("default"))
	// Output:
	// hello
	// default
}

func ExampleMap() {
	r := result.Ok(5)
	doubled := result.Map(r, func(n int) int { return n * 2 })
	fmt.Println(doubled.Value())
	// Output:
	// 10
}

func ExampleFlatMap() {
	parse := func(s string) result.Result[int] {
		if s == "" {
			return result.Fail[int](kerrors.ErrValidation)
		}
		return result.Ok(len(s))
	}

	r := result.FlatMap(result.Ok("hello"), parse)
	fmt.Println(r.Value())
	// Output:
	// 5
}

func ExampleResult_Unwrap() {
	val, err := result.Ok("data").Unwrap()
	fmt.Println(val, err)
	// Output:
	// data <nil>
}
