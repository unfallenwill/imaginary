//go:build cgo && ffmpeg

package frame

import (
	"os"
	"path"
	"testing"
)

func TestExtractFirstFrame(t *testing.T) {
	buf, err := os.ReadFile(path.Join("../../testdata", "sample.mp4"))
	if err != nil {
		t.Skip("No test video file available")
	}

	result, err := Extract(buf, 0)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(result.Body) == 0 {
		t.Fatal("Empty frame body")
	}
	if result.Mime != "image/jpeg" {
		t.Errorf("Expected image/jpeg, got %s", result.Mime)
	}
	// Verify it's valid JPEG (starts with FFD8)
	if len(result.Body) < 2 || result.Body[0] != 0xFF || result.Body[1] != 0xD8 {
		t.Fatal("Result is not a valid JPEG")
	}
}

func TestExtractAtTimeOffset(t *testing.T) {
	buf, err := os.ReadFile(path.Join("../../testdata", "sample.mp4"))
	if err != nil {
		t.Skip("No test video file available")
	}

	result, err := Extract(buf, 0.5)
	if err != nil {
		t.Fatalf("Extract at 0.5s failed: %v", err)
	}
	if len(result.Body) == 0 {
		t.Fatal("Empty frame body")
	}
}

func TestExtractBeyondDuration(t *testing.T) {
	buf, err := os.ReadFile(path.Join("../../testdata", "sample.mp4"))
	if err != nil {
		t.Skip("No test video file available")
	}

	// Requesting a frame at a very large timestamp should still return
	// something (typically the last frame or a nearby keyframe)
	result, err := Extract(buf, 9999)
	if err != nil {
		t.Fatalf("Extract beyond duration should not fail: %v", err)
	}
	if len(result.Body) == 0 {
		t.Fatal("Empty frame body")
	}
}
