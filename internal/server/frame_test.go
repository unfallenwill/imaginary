package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/h2non/imaginary/internal/source"
	bodysource "github.com/h2non/imaginary/internal/source/body"
	fssource "github.com/h2non/imaginary/internal/source/fs"
	httpsource "github.com/h2non/imaginary/internal/source/http"
	objectsource "github.com/h2non/imaginary/internal/source/object"
)

func TestFrameEmptyBody(t *testing.T) {
	cfg := testFrameConfig()
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Post(ts.URL+"/frame", "video/mp4", nil)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 400 {
		t.Fatalf("Expected 400 for empty body, got %d", res.StatusCode)
	}
}

func TestFrameMissingSource(t *testing.T) {
	cfg := Config{
		PathPrefix:       "/",
		MaxAllowedPixels: 18.0,
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(&source.SourceConfig{Type: bodysource.ImageSourceTypeBody}),
			objectsource.NewObjectImageSource(&source.SourceConfig{Type: objectsource.ImageSourceTypeObject}),
			fssource.NewFileSystemImageSource(&source.SourceConfig{Type: fssource.ImageSourceTypeFileSystem}),
			httpsource.NewHTTPImageSource(&source.SourceConfig{Type: httpsource.ImageSourceTypeHTTP}),
		),
	}

	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/frame")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 400 {
		t.Fatalf("Expected 400 for missing source, got %d", res.StatusCode)
	}
}

func TestFrameInvalidTimeParam(t *testing.T) {
	cfg := testFrameConfig()
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	// Send a small image as body (not a video, but the error check for time happens before that)
	buf := []byte("fake video content for param validation")

	res, err := http.Post(ts.URL+"/frame?time=abc", "video/mp4", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 400 {
		t.Fatalf("Expected 400 for invalid time param, got %d", res.StatusCode)
	}
}

func TestFrameNegativeTimeParam(t *testing.T) {
	cfg := testFrameConfig()
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	buf := []byte("fake video content for param validation")

	res, err := http.Post(ts.URL+"/frame?time=-1", "video/mp4", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 400 {
		t.Fatalf("Expected 400 for negative time param, got %d", res.StatusCode)
	}
}

func TestFrameImageInput(t *testing.T) {
	cfg := testFrameConfig()
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	buf, err := os.ReadFile(path.Join("../../testdata", "imaginary.jpg"))
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.Post(ts.URL+"/frame", "image/jpeg", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}

	// Should reject non-video input with 406
	if res.StatusCode != 406 {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("Expected 406 for image input, got %d: %s", res.StatusCode, body)
	}
}

func TestFrameDisabledEndpoint(t *testing.T) {
	cfg := testFrameConfig()
	cfg.Endpoints = EndpointSet{"frame"}

	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	buf, err := os.ReadFile(path.Join("../../testdata", "imaginary.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(ts.URL+"/frame", "video/mp4", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 501 {
		t.Fatalf("Expected 501 for disabled endpoint, got %d", res.StatusCode)
	}
}

func testFrameConfig() Config {
	return Config{
		PathPrefix:       "/",
		MaxAllowedPixels: 18.0,
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(&source.SourceConfig{Type: bodysource.ImageSourceTypeBody}),
			objectsource.NewObjectImageSource(&source.SourceConfig{Type: objectsource.ImageSourceTypeObject}),
			fssource.NewFileSystemImageSource(&source.SourceConfig{Type: fssource.ImageSourceTypeFileSystem}),
			httpsource.NewHTTPImageSource(&source.SourceConfig{Type: httpsource.ImageSourceTypeHTTP}),
		),
	}
}
