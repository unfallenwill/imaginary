package server

import (
	"fmt"
	"net/http"
	"testing"

	img "github.com/h2non/imaginary/internal/image"
)

func TestHTTPStatusFor(t *testing.T) {
	newErr := img.NewError("test")

	cases := []struct {
		name   string
		err    error
		expect int
	}{
		// Transport-layer errors
		{"unauthorized", &UnauthorizedError{Err: newErr}, http.StatusUnauthorized},
		{"forbidden", &ForbiddenError{Err: newErr}, http.StatusForbidden},
		{"method not allowed", &MethodNotAllowedError{Err: newErr}, http.StatusMethodNotAllowed},

		// Domain-layer errors
		{"not found", &img.NotFoundError{Err: newErr}, http.StatusNotFound},
		{"unsupported media", &img.UnsupportedMediaError{Err: newErr}, http.StatusNotAcceptable},
		{"invalid param", &img.InvalidParamError{Err: newErr}, http.StatusBadRequest},
		{"empty body", &img.EmptyBodyError{Err: newErr}, http.StatusBadRequest},
		{"resolution too big", &img.ResolutionTooBigError{Err: newErr}, http.StatusUnprocessableEntity},
		{"not implemented", &img.NotImplementedError{Err: newErr}, http.StatusNotImplemented},
		{"processing", &img.ProcessingError{Err: newErr}, http.StatusInternalServerError},
		{"upstream", &img.UpstreamError{Err: newErr}, http.StatusBadGateway},

		// Fallback
		{"unknown error", fmt.Errorf("something broke"), http.StatusInternalServerError},

		// Sentinel errors
		{"sentinel: invalid API key", errInvalidAPIKey, http.StatusUnauthorized},
		{"sentinel: method not allowed", errMethodNotAllowed, http.StatusMethodNotAllowed},
		{"sentinel: URL signature mismatch", errURLSignatureMismatch, http.StatusForbidden},
		{"sentinel: invalid URL signature", errInvalidURLSignature, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := httpStatusFor(tc.err)
			if got != tc.expect {
				t.Errorf("httpStatusFor(%T) = %d, want %d", tc.err, got, tc.expect)
			}
		})
	}
}
