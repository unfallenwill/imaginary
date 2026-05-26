package image_test

import (
	"errors"
	"testing"

	"github.com/h2non/imaginary/internal/image"
)

func TestErrorMessageSanitized(t *testing.T) {
	err := image.New(image.KindProcessing, "oops!\n\n")
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

func TestErrorKindClassification(t *testing.T) {
	err := image.New(image.KindInvalidParam, "bad param")
	var imgErr *image.Error
	if !errors.As(err, &imgErr) {
		t.Fatal("expected errors.As to extract *image.Error")
	}
	if imgErr.Kind != image.KindInvalidParam {
		t.Errorf("expected KindInvalidParam, got %v", imgErr.Kind)
	}

	// A different kind should not match
	err2 := image.New(image.KindNotFound, "gone")
	if !errors.As(err2, &imgErr) {
		t.Fatal("expected errors.As to extract *image.Error")
	}
	if imgErr.Kind == image.KindInvalidParam {
		t.Error("expected Kind to differ between error instances")
	}
}

func TestWrapError(t *testing.T) {
	inner := errors.New("inner error")
	wrapped := image.Wrap(image.KindProcessing, "outer", inner)

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
	wrapped := image.Wrap(image.KindNotFound, "level1", inner)

	if !errors.Is(wrapped, inner) {
		t.Error("expected errors.Is to match inner error")
	}

	var imgErr *image.Error
	if !errors.As(wrapped, &imgErr) {
		t.Fatal("expected errors.As to extract *image.Error from wrapped error")
	}
	if imgErr.Kind != image.KindNotFound {
		t.Errorf("expected KindNotFound, got %v", imgErr.Kind)
	}
}

func TestErrorUnwrapToRoot(t *testing.T) {
	root := errors.New("root")
	mid := image.Wrap(image.KindProcessing, "mid", root)
	top := image.Wrap(image.KindUpstream, "top", mid)

	if !errors.Is(top, root) {
		t.Error("expected errors.Is to traverse full chain to root")
	}
	if top.Error() != "top: mid: root" {
		t.Errorf("expected 'top: mid: root', got %q", top.Error())
	}
}
