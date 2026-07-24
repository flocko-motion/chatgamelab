// package: functional / generic helper primitives
// type:    logic
// job:     small generic helpers for error handling and pointer wrapping
// limits:  stateless pure helpers; no domain types or IO
package functional

import "log"

// Must logs the formatted message and exits if err is non-nil.
func Must(err error, format string, args ...any) {
	if err != nil {
		log.Fatalf(format+": %s", append(args, err)...)
	}
}

// Ptr returns a pointer to the given value.
func Ptr[T any](v T) *T {
	return &v
}

// Deref dereferences a pointer, returning the default value if the pointer is nil.
func Deref[T any](ptr *T, defaultValue T) T {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// MustReturn returns v if err is nil, otherwise logs the error and exits.
func MustReturn[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}

// First returns the first value from a multi-return function, discarding the rest.
// If the last argument is an error, it logs it.
func First[T any](v T, rest ...any) T {
	if len(rest) > 0 {
		if err, ok := rest[len(rest)-1].(error); ok && err != nil {
			log.Printf("First: discarded error: %v", err)
		}
	}
	return v
}
