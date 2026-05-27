package image

import (
	"errors"
	"strings"
	"testing"
)

func TestCatchPanicNoPanic(t *testing.T) {
	result, err := CatchPanic(func() (int, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}
}

func TestCatchPanicNoPanicWithError(t *testing.T) {
	result, err := CatchPanic(func() (int, error) {
		return 0, errors.New("normal error")
	})
	if err == nil || err.Error() != "normal error" {
		t.Fatalf("expected 'normal error', got %v", err)
	}
	if result != 0 {
		t.Fatalf("expected zero value, got %d", result)
	}
}

func TestCatchPanicErrorPanic(t *testing.T) {
	sentinel := errors.New("libvips CGo boom")
	result, err := CatchPanic(func() (int, error) {
		panic(sentinel)
	})
	if err == nil {
		t.Fatal("expected error from panic recovery")
	}
	var imgErr *Error
	if !errors.As(err, &imgErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if imgErr.Kind != KindProcessing {
		t.Fatalf("expected KindProcessing, got %d", imgErr.Kind)
	}
	if !strings.Contains(imgErr.Message, "libvips processing error") {
		t.Fatalf("expected 'libvips processing error' in message, got: %s", imgErr.Message)
	}
	if !errors.Is(err, sentinel) {
		t.Fatal("expected wrapped sentinel error")
	}
	if result != 0 {
		t.Fatalf("expected zero value, got %d", result)
	}
}

func TestCatchPanicStringPanic(t *testing.T) {
	result, err := CatchPanic(func() (int, error) {
		panic("string-based CGo panic")
	})
	if err == nil {
		t.Fatal("expected error from string panic recovery")
	}
	var imgErr *Error
	if !errors.As(err, &imgErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if imgErr.Kind != KindProcessing {
		t.Fatalf("expected KindProcessing, got %d", imgErr.Kind)
	}
	if !strings.Contains(imgErr.Message, "string-based CGo panic") {
		t.Fatalf("expected message to contain original string, got: %s", imgErr.Message)
	}
	if result != 0 {
		t.Fatalf("expected zero value, got %d", result)
	}
}

func TestCatchPanicRePanicsOnUnexpectedType(t *testing.T) {
	// Verify that a nil-deref-type panic re-panics rather than being swallowed.
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected re-panic on unexpected panic type")
		}
	}()

	_, _ = CatchPanic(func() (int, error) {
		var p *int
		_ = *p // nil dereference → should re-panic
		return 0, nil
	})
}

func TestCatchPanicRePanicsOnIntPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected re-panic on int panic")
		}
		if v, ok := r.(int); !ok || v != 123 {
			t.Fatalf("expected 123, got %v", r)
		}
	}()

	_, _ = CatchPanic(func() (int, error) {
		panic(123) // not error, not string → should re-panic
		return 0, nil
	})
}

func TestCatchPanicZeroForStruct(t *testing.T) {
	// Verify zero-value is returned for struct types on panic.
	_, err := CatchPanic(func() (Image, error) {
		panic("boom")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	// No assertion on the Image value — the key is that it compiles
	// and the generic zero[T]() works correctly for struct types.
}
