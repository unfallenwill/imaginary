package image_test

import (
	"testing"

	"github.com/h2non/imaginary/internal/image"
)

func TestDefaultError(t *testing.T) {
	err := image.NewError("oops!\n\n", image.KindProcessing)

	if err.Error() != "oops!" {
		t.Fatal("Invalid error message")
	}
	if err.Kind != image.KindProcessing {
		t.Fatal("Invalid error kind")
	}

	json := string(err.JSON())
	if json != `{"message":"oops!","status":10}` {
		t.Fatalf("Invalid JSON output: %s", json)
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		err  image.Error
		kind image.Kind
	}{
		{image.ErrNotFound, image.KindNotFound},
		{image.ErrInvalidAPIKey, image.KindUnauthorized},
		{image.ErrURLSignatureMismatch, image.KindForbidden},
		{image.ErrMethodNotAllowed, image.KindMethodNotAllowed},
		{image.ErrUnsupportedMedia, image.KindUnsupportedMedia},
		{image.ErrEmptyBody, image.KindEmptyBody},
		{image.ErrResolutionTooBig, image.KindResolutionTooBig},
		{image.ErrNotImplemented, image.KindNotImplemented},
		{image.ErrOutputFormat, image.KindInvalidParam},
	}

	for _, tt := range tests {
		if tt.err.Kind != tt.kind {
			t.Errorf("expected kind %d, got %d for error: %s", tt.kind, tt.err.Kind, tt.err.Message)
		}
	}
}
