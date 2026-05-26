package server

import (
	"fmt"
	"net/http"
	"testing"

	img "github.com/h2non/imaginary/internal/image"
)

func TestHTTPStatusFor(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		expect int
	}{
		// Transport-level kinds
		{"unauthorized", img.New(img.KindUnauthorized, "test"), http.StatusUnauthorized},
		{"forbidden", img.New(img.KindForbidden, "test"), http.StatusForbidden},
		{"method not allowed", img.New(img.KindMethodNotAllowed, "test"), http.StatusMethodNotAllowed},

		// Domain-level kinds
		{"not found", img.New(img.KindNotFound, "test"), http.StatusNotFound},
		{"unsupported media", img.New(img.KindUnsupportedMedia, "test"), http.StatusNotAcceptable},
		{"invalid param", img.New(img.KindInvalidParam, "test"), http.StatusBadRequest},
		{"empty body", img.New(img.KindEmptyBody, "test"), http.StatusBadRequest},
		{"resolution too big", img.New(img.KindResolutionTooBig, "test"), http.StatusUnprocessableEntity},
		{"not implemented", img.New(img.KindNotImplemented, "test"), http.StatusNotImplemented},
		{"processing", img.New(img.KindProcessing, "test"), http.StatusInternalServerError},
		{"upstream", img.New(img.KindUpstream, "test"), http.StatusBadGateway},

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
