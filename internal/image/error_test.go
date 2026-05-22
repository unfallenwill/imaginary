package image_test

import (
	"errors"
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

	jsn := string(err.JSON())
	if jsn != `{"message":"oops!","status":10}` {
		t.Fatalf("Invalid JSON output: %s", jsn)
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		err  *image.Error
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

func TestErrorIs(t *testing.T) {
	err := image.NewError("some not found", image.KindNotFound)
	if !errors.Is(err, image.ErrNotFound) {
		t.Error("expected errors.Is to match sentinel by Kind")
	}

	// Different Kind should not match
	if errors.Is(err, image.ErrUnsupportedMedia) {
		t.Error("expected errors.Is not to match different Kind")
	}
}

func TestErrorAs(t *testing.T) {
	err := image.NewError("bad param", image.KindInvalidParam)
	var imgErr *image.Error
	if !errors.As(err, &imgErr) {
		t.Error("expected errors.As to extract *image.Error")
	}
	if imgErr.Kind != image.KindInvalidParam {
		t.Errorf("expected KindInvalidParam, got %d", imgErr.Kind)
	}
}

func TestWrapError(t *testing.T) {
	inner := errors.New("inner error")
	wrapped := image.WrapError("outer", image.KindProcessing, inner)

	if wrapped.Error() != "outer: inner error" {
		t.Errorf("expected 'outer: inner error', got %q", wrapped.Error())
	}
	if !errors.Is(wrapped, inner) {
		t.Error("expected errors.Is to reach inner error via Unwrap")
	}
	if wrapped.Unwrap() != inner {
		t.Error("expected Unwrap to return inner error")
	}
	if !errors.Is(wrapped, image.NewError("any", image.KindProcessing)) {
		t.Error("expected errors.Is to match by Kind")
	}
}

func TestWrapErrorChain(t *testing.T) {
	inner := errors.New("root cause")
	wrapped := image.WrapError("level1", image.KindNotFound, inner)

	// errors.Is should match both the sentinel and the inner error
	if !errors.Is(wrapped, image.ErrNotFound) {
		t.Error("expected errors.Is to match ErrNotFound sentinel")
	}
	if !errors.Is(wrapped, inner) {
		t.Error("expected errors.Is to match inner error")
	}

	// errors.As should extract the *image.Error
	var imgErr *image.Error
	if !errors.As(wrapped, &imgErr) {
		t.Error("expected errors.As to extract *image.Error from wrapped error")
	}
}
