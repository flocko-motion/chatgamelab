package functional

import "log"

// Require gets an error return from a function and logs it as a fatal error if it is not nil
func Require(err error, format string, args ...any) {
	if err != nil {
		log.Fatalf(format+": %s", append(args, err)...)
	}
}

func Ptr[T any](v T) *T {
	return &v
}
