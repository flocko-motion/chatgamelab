package functional

import "log"

func Must(err error, format string, args ...any) {
	if err != nil {
		log.Fatalf(format+": %s", append(args, err)...)
	}
}

func Ptr[T any](v T) *T {
	return &v
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
