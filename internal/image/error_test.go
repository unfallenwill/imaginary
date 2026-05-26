package image_test

import (
	"errors"
	"testing"

	"github.com/h2non/imaginary/internal/image"
)

func TestErrorMessages(t *testing.T) {
	err := image.NewProcessingError("oops!\n\n")
	if err.Error() != "oops!" {
		t.Fatalf("expected 'oops!', got %q", err.Error())
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		err error
		msg string
	}{
		{image.ErrNotFound, "Not found"},
		{image.ErrUnsupportedMedia, "Unsupported media type"},
		{image.ErrEmptyBody, "Empty or unreadable image"},
		{image.ErrResolutionTooBig, "Image resolution is too big"},
		{image.ErrNotImplemented, "Not implemented endpoint"},
		{image.ErrOutputFormat, "Unsupported output image format"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.msg {
			t.Errorf("expected message %q, got %q", tt.msg, tt.err.Error())
		}
	}
}

func TestErrorAsByType(t *testing.T) {
	// errors.As should match the specific error type
	err := image.NewInvalidParamError("bad param")
	var ipe *image.InvalidParamError
	if !errors.As(err, &ipe) {
		t.Error("expected errors.As to extract *image.InvalidParamError")
	}

	// The base *image.Error is stored in a named field, not embedded,
	// so errors.As cannot reach it directly. This is by design —
	// the server layer matches the outer type, not the inner *Error.
}

func TestWrapError(t *testing.T) {
	inner := errors.New("inner error")
	wrapped := image.WrapProcessingError("outer", inner)

	if wrapped.Error() != "outer: inner error" {
		t.Errorf("expected 'outer: inner error', got %q", wrapped.Error())
	}
	if !errors.Is(wrapped, inner) {
		t.Error("expected errors.Is to reach inner error via Unwrap")
	}
	if wrapped.Unwrap() != inner {
		t.Error("expected Unwrap to return inner error")
	}
}

func TestWrapErrorChain(t *testing.T) {
	inner := errors.New("root cause")
	wrapped := image.WrapNotFoundError("level1", inner)

	// errors.Is should match the inner error
	if !errors.Is(wrapped, inner) {
		t.Error("expected errors.Is to match inner error")
	}

	// errors.As should extract the specific type
	var nfe *image.NotFoundError
	if !errors.As(wrapped, &nfe) {
		t.Error("expected errors.As to extract *image.NotFoundError from wrapped error")
	}

	// The base *image.Error is a named field, not embedded, so errors.As
	// cannot reach it. The server layer matches the outer type (NotFoundError).
}

func TestNotFoundErrorIsType(t *testing.T) {
	err := image.NewNotFoundError("gone")
	var nfe *image.NotFoundError
	if !errors.As(err, &nfe) {
		t.Error("expected errors.As to match NotFoundError")
	}

	// Should NOT match a different type
	var ipe *image.InvalidParamError
	if errors.As(err, &ipe) {
		t.Error("expected errors.As NOT to match InvalidParamError")
	}
}
