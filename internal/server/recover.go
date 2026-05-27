package server

import (
	"log"
	"net/http"
	"runtime/debug"
)

// RecoverHandler wraps an http.Handler with a panic recovery that prevents a
// single panicking request from crashing the entire server. It logs the full
// stack trace and returns HTTP 500.
//
// This is the production safety net. Unlike image.CatchPanic (which re-panics
// on unexpected panic types so they are visible in CI), RecoverHandler catches
// ALL panics — in production you never want to crash on a single bad request.
func RecoverHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC recovered: %v\n%s", rec, debug.Stack())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
