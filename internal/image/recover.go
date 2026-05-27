package image

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// CatchPanic calls fn and recovers from panics that originate from CGo/libraries
// (panicking with error or string values). These are converted to *Error with
// KindProcessing so callers can treat them as normal errors.
//
// Panics of any other type (nil pointer dereference, index out of range, etc.)
// are programmer bugs — CatchPanic re-panics with the original value so they
// are visible during development and CI.
//
// Usage:
//
//	result, err := CatchPanic(func() (Image, error) {
//	    return Process(buf, opts)
//	})
func CatchPanic[T any](fn func() (T, error)) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch value := r.(type) {
			case runtime.Error:
				// Go runtime panics (nil deref, index bounds, type assertion, etc.)
				// are programmer bugs. Re-panic so they are visible in CI.
				_, _ = fmt.Fprintln(panicWriter, "unexpected Go runtime panic in image operation:", r)
				_, _ = fmt.Fprintln(panicWriter, string(debug.Stack()))
				panic(r)
			case error:
				// CGo/libvips panics that carry an error value — convert to *Error.
				err = Wrap(KindProcessing, "libvips processing error", value)
			case string:
				// CGo/libvips panics that carry a string — convert to *Error.
				err = New(KindProcessing, value)
			default:
				// Any other type — also a programmer bug, re-panic.
				_, _ = fmt.Fprintln(panicWriter, "unexpected panic in image operation:", r)
				_, _ = fmt.Fprintln(panicWriter, string(debug.Stack()))
				panic(r)
			}
			result = zero[T]()
		}
	}()
	return fn()
}

// panicWriter is the writer used for logging unexpected panics before re-panicking.
// Defaults to a discard writer; tests may override it.
var panicWriter interface{ Write([]byte) (int, error) } = discardWriter{}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// zero returns the zero value for type T.
func zero[T any]() T {
	var z T
	return z
}
