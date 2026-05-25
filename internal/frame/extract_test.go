package frame

import (
	"testing"
)

func TestExtractInvalidInput(t *testing.T) {
	buf := []byte("not a video")
	_, err := Extract(buf, 0)
	if err == nil {
		t.Fatal("Expected error for invalid input")
	}
}

func TestExtractEmptyInput(t *testing.T) {
	_, err := Extract([]byte{}, 0)
	if err == nil {
		t.Fatal("Expected error for empty input")
	}
}
