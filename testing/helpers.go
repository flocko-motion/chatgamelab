// package: testing / test assertion helpers
// type:    logic
// job:     generic panic-based helpers for asserting success in tests
// limits:  success only; a refusal is asserted by its code (-> testutil refusals.go), never as a bare "it failed"
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

// MustSucceed expects no error and panics if there is one
// Use this for operations that return only error (like Delete)
func MustSucceed(err error) {
	if err != nil {
		panic(fmt.Sprintf("MustSucceed failed: %v", err))
	}
}
