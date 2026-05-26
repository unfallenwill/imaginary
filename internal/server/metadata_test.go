package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/h2non/imaginary/internal/metadata"
	"github.com/h2non/imaginary/internal/source"
	bodysource "github.com/h2non/imaginary/internal/source/body"
	fssource "github.com/h2non/imaginary/internal/source/fs"
	httpsource "github.com/h2non/imaginary/internal/source/http"
	objectsource "github.com/h2non/imaginary/internal/source/object"
)

func TestMetadataImagePOST(t *testing.T) {
	cfg := testMetadataConfig()
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	buf, err := os.ReadFile(path.Join("../../testdata", "imaginary.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(ts.URL+"/metadata", "image/jpeg", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("Expected 200, got %d: %s", res.StatusCode, body)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("Expected application/json, got %s", res.Header.Get("Content-Type"))
	}

	body, _ := io.ReadAll(res.Body)
	var meta metadata.Metadata
	if err := json.Unmarshal(body, &meta); err != nil {
		t.Fatalf("Cannot unmarshal response: %v", err)
	}

	if meta.MediaType != "image" {
		t.Errorf("Expected mediaType=image, got %s", meta.MediaType)
	}
	if meta.Image == nil {
		t.Fatal("Image metadata should not be nil")
	}
	if meta.Image.Width != 550 {
		t.Errorf("Expected width=550, got %d", meta.Image.Width)
	}
	if meta.Image.Height != 740 {
		t.Errorf("Expected height=740, got %d", meta.Image.Height)
	}
}

func TestMetadataImageFileSource(t *testing.T) {
	cfg := Config{
		PathPrefix:       "/",
		MaxAllowedPixels: 18.0,
		Mount:            "../../testdata",
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(),
			objectsource.NewObjectImageSource(nil, 0),
			fssource.NewFileSystemImageSource("../../testdata"),
			httpsource.NewHTTPImageSource(httpsource.Config{}),
		),
	}

	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/metadata?file=imaginary.jpg")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("Expected 200, got %d: %s", res.StatusCode, body)
	}

	body, _ := io.ReadAll(res.Body)
	var meta metadata.Metadata
	if err := json.Unmarshal(body, &meta); err != nil {
		t.Fatalf("Cannot unmarshal response: %v", err)
	}

	if meta.MediaType != "image" {
		t.Errorf("Expected mediaType=image, got %s", meta.MediaType)
	}
}

func TestMetadataEmptyBody(t *testing.T) {
	cfg := testMetadataConfig()
	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Post(ts.URL+"/metadata", "image/jpeg", nil)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 400 {
		t.Fatalf("Expected 400 for empty body, got %d", res.StatusCode)
	}
}

func TestMetadataMissingSource(t *testing.T) {
	cfg := Config{
		PathPrefix:       "/",
		MaxAllowedPixels: 18.0,
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(),
			objectsource.NewObjectImageSource(nil, 0),
			fssource.NewFileSystemImageSource(""),
			httpsource.NewHTTPImageSource(httpsource.Config{}),
		),
	}

	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	// GET without any source params should fail
	res, err := http.Get(ts.URL + "/metadata")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 400 {
		t.Fatalf("Expected 400 for missing source, got %d", res.StatusCode)
	}
}

func TestMetadataObjectStorageSource(t *testing.T) {
	buf, err := os.ReadFile(path.Join("../../testdata", "large.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	obj := fakeObjectStorage{body: buf}

	cfg := Config{
		PathPrefix:       "/",
		MaxAllowedPixels: 18.0,
		ObjectStorage:    obj,
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(),
			objectsource.NewObjectImageSource(obj, 0),
			fssource.NewFileSystemImageSource(""),
			httpsource.NewHTTPImageSource(httpsource.Config{}),
		),
	}

	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/metadata?object=test/image.jpg")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("Expected 200, got %d: %s", res.StatusCode, body)
	}

	body, _ := io.ReadAll(res.Body)
	var meta metadata.Metadata
	if err := json.Unmarshal(body, &meta); err != nil {
		t.Fatalf("Cannot unmarshal response: %v", err)
	}

	if meta.MediaType != "image" {
		t.Errorf("Expected mediaType=image, got %s", meta.MediaType)
	}
}

func TestMetadataDisabledEndpoint(t *testing.T) {
	cfg := testMetadataConfig()
	cfg.Endpoints = EndpointSet{"metadata"}

	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	buf, err := os.ReadFile(path.Join("../../testdata", "imaginary.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(ts.URL+"/metadata", "image/jpeg", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 501 {
		t.Fatalf("Expected 501 for disabled endpoint, got %d", res.StatusCode)
	}
}

func testMetadataConfig() Config {
	return Config{
		PathPrefix:       "/",
		MaxAllowedPixels: 18.0,
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(),
			objectsource.NewObjectImageSource(nil, 0),
			fssource.NewFileSystemImageSource(""),
			httpsource.NewHTTPImageSource(httpsource.Config{}),
		),
	}
}
