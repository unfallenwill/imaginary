package metadata

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/h2non/bimg"
)

func TestExtractImage(t *testing.T) {
	buf, err := os.ReadFile(path.Join("../../testdata", "imaginary.jpg"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := ExtractImage(buf)
	if err != nil {
		t.Fatalf("ExtractImage failed: %v", err)
	}

	if len(result.Body) == 0 {
		t.Fatal("Empty result body")
	}

	var meta Metadata
	if err := json.Unmarshal(result.Body, &meta); err != nil {
		t.Fatalf("Cannot unmarshal result: %v", err)
	}

	if meta.MediaType != "image" {
		t.Fatalf("Expected mediaType=image, got %s", meta.MediaType)
	}
	if meta.Image == nil {
		t.Fatal("Image metadata is nil")
	}
	if meta.Video != nil {
		t.Fatal("Video metadata should be nil for image input")
	}

	img := meta.Image
	if img.Width != 550 {
		t.Errorf("Expected width=550, got %d", img.Width)
	}
	if img.Height != 740 {
		t.Errorf("Expected height=740, got %d", img.Height)
	}
	if img.Type != "jpeg" {
		t.Errorf("Expected type=jpeg, got %s", img.Type)
	}
	if img.Channels == 0 {
		t.Error("Expected non-zero channels")
	}
}

func TestExtractImagePNG(t *testing.T) {
	buf, err := os.ReadFile(path.Join("../../testdata", "test.png"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := ExtractImage(buf)
	if err != nil {
		t.Fatalf("ExtractImage failed: %v", err)
	}

	var meta Metadata
	if err := json.Unmarshal(result.Body, &meta); err != nil {
		t.Fatalf("Cannot unmarshal result: %v", err)
	}

	if meta.MediaType != "image" {
		t.Fatalf("Expected mediaType=image, got %s", meta.MediaType)
	}
	if meta.Image == nil {
		t.Fatal("Image metadata is nil")
	}
	if meta.Image.Type != "png" {
		t.Errorf("Expected type=png, got %s", meta.Image.Type)
	}
}

func TestExtractImageWebP(t *testing.T) {
	buf, err := os.ReadFile(path.Join("../../testdata", "test.webp"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := ExtractImage(buf)
	if err != nil {
		t.Fatalf("ExtractImage failed: %v", err)
	}

	var meta Metadata
	if err := json.Unmarshal(result.Body, &meta); err != nil {
		t.Fatalf("Cannot unmarshal result: %v", err)
	}

	if meta.MediaType != "image" {
		t.Fatalf("Expected mediaType=image, got %s", meta.MediaType)
	}
	if meta.Image.Type != "webp" {
		t.Errorf("Expected type=webp, got %s", meta.Image.Type)
	}
}

func TestExtract(t *testing.T) {
	buf, err := os.ReadFile(path.Join("../../testdata", "imaginary.jpg"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := Extract(buf, MediaTypeImage)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	var meta Metadata
	if err := json.Unmarshal(result.Body, &meta); err != nil {
		t.Fatalf("Cannot unmarshal result: %v", err)
	}

	if meta.MediaType != "image" {
		t.Fatalf("Expected mediaType=image, got %s", meta.MediaType)
	}
}

func TestExtractUnsupported(t *testing.T) {
	buf := []byte("not a valid media file")
	_, err := Extract(buf, MediaTypeUnknown)
	if err == nil {
		t.Fatal("Expected error for unsupported media type")
	}
}

func TestExtractImageEXIF(t *testing.T) {
	buf, err := os.ReadFile(path.Join("../../testdata", "imaginary.jpg"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := ExtractImage(buf)
	if err != nil {
		t.Fatalf("ExtractImage failed: %v", err)
	}

	var meta Metadata
	if err := json.Unmarshal(result.Body, &meta); err != nil {
		t.Fatalf("Cannot unmarshal result: %v", err)
	}

	// EXIF struct should be present (even if fields are zero values)
	_ = meta.Image.EXIF
}

func TestImageMetadataMatchesBimg(t *testing.T) {
	buf, err := os.ReadFile(path.Join("../../testdata", "large.jpg"))
	if err != nil {
		t.Fatal(err)
	}

	// Get bimg metadata directly
	bimgMeta, err := bimg.Metadata(buf)
	if err != nil {
		t.Fatal(err)
	}

	// Get metadata via our package
	result, err := ExtractImage(buf)
	if err != nil {
		t.Fatal(err)
	}

	var meta Metadata
	if err := json.Unmarshal(result.Body, &meta); err != nil {
		t.Fatal(err)
	}

	// Verify values match
	if meta.Image.Width != bimgMeta.Size.Width {
		t.Errorf("Width mismatch: got %d, expected %d", meta.Image.Width, bimgMeta.Size.Width)
	}
	if meta.Image.Height != bimgMeta.Size.Height {
		t.Errorf("Height mismatch: got %d, expected %d", meta.Image.Height, bimgMeta.Size.Height)
	}
	if meta.Image.Type != bimgMeta.Type {
		t.Errorf("Type mismatch: got %s, expected %s", meta.Image.Type, bimgMeta.Type)
	}
	if meta.Image.Space != bimgMeta.Space {
		t.Errorf("Space mismatch: got %s, expected %s", meta.Image.Space, bimgMeta.Space)
	}
}
