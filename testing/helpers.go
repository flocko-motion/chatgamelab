package testing

import "fmt"

// Must is a generic helper that panics on error, otherwise returns the typed value
// The panic will be caught by the test framework and fail the test
func Must[T any](value T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("Must failed: %v", err))
	}
	return value
}

// Fail expects an error and panics if there isn't one
// Use this to verify that an operation should fail
func Fail(v any, err error) {
	if err == nil {
		panic("Fail: expected an error but got none")
	}
}

// MustSucceed expects no error and panics if there is one
// Use this for operations that return only error (like Delete)
func MustSucceed(err error) {
	if err != nil {
		panic(fmt.Sprintf("MustSucceed failed: %v", err))
	}
}

func MustFail(err error) {
	if err == nil {
		panic("MustFail: expected an error but got none")
	}
}
