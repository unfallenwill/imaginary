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

func TestPathFrameNoObjectStorage(t *testing.T) {
	cfg := testFrameConfig()
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/frame/uploads/video.mp4")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 501 {
		t.Fatalf("Expected 501 for missing object storage, got %d", res.StatusCode)
	}
}

func TestPathFrameDisabledEndpoint(t *testing.T) {
	buf := []byte("fake video content")
	cfg := testPathFrameConfig(buf)
	cfg.Endpoints = EndpointSet{"frame"}

	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/frame/uploads/video.mp4")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 501 {
		t.Fatalf("Expected 501 for disabled endpoint, got %d", res.StatusCode)
	}
}

func TestPathFramePostMethod(t *testing.T) {
	cfg := testPathFrameConfig([]byte("video"))
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Post(ts.URL+"/frame/uploads/video.mp4", "video/mp4", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 405 {
		t.Fatalf("Expected 405 for POST method, got %d", res.StatusCode)
	}
}

func TestPathFrameMissingKey(t *testing.T) {
	cfg := testPathFrameConfig([]byte("video"))
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	// Request to /frame/ with no key after the prefix
	res, err := http.Get(ts.URL + "/frame/")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 400 {
		t.Fatalf("Expected 400 for missing object key, got %d", res.StatusCode)
	}
}

func TestPathFrameNonVideoInput(t *testing.T) {
	buf, err := os.ReadFile(path.Join("../../testdata", "imaginary.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := testPathFrameConfig(buf)
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/frame/uploads/photo.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 406 {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("Expected 406 for non-video input, got %d: %s", res.StatusCode, body)
	}
}

func TestPathFrameInvalidTimeParam(t *testing.T) {
	cfg := testPathFrameConfig([]byte("fake video content"))
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/frame/uploads/video.mp4?time=abc")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 400 {
		t.Fatalf("Expected 400 for invalid time param, got %d", res.StatusCode)
	}
}

func TestPathFrameNegativeTimeParam(t *testing.T) {
	cfg := testPathFrameConfig([]byte("fake video content"))
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/frame/uploads/video.mp4?time=-1")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 400 {
		t.Fatalf("Expected 400 for negative time param, got %d", res.StatusCode)
	}
}

func TestPathFrameNestedKey(t *testing.T) {
	buf := []byte("fake video content")
	cfg := testPathFrameConfig(buf)
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	// Nested paths should parse but return 406 since the fake content isn't a real video
	res, err := http.Get(ts.URL + "/frame/videos/2026/05/clip.mp4?time=1.5")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 406 {
		t.Fatalf("Expected 406 for non-video nested key, got %d", res.StatusCode)
	}
}

func testPathFrameConfig(body []byte) Config {
	obj := fakeObjectStorage{body: body}
	return Config{
		PathPrefix:       "/",
		MaxAllowedPixels: 18.0,
		ObjectStorage:    obj,
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(&source.SourceConfig{Type: bodysource.ImageSourceTypeBody}),
			objectsource.NewObjectImageSource(&source.SourceConfig{
				Type:          objectsource.ImageSourceTypeObject,
				ObjectStorage: obj,
			}),
			fssource.NewFileSystemImageSource(&source.SourceConfig{Type: fssource.ImageSourceTypeFileSystem}),
			httpsource.NewHTTPImageSource(&source.SourceConfig{Type: httpsource.ImageSourceTypeHTTP}),
		),
	}
}
