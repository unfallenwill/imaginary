package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/h2non/bimg"

	"github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
	bodysource "github.com/h2non/imaginary/internal/source/body"
	fssource "github.com/h2non/imaginary/internal/source/fs"
	httpsource "github.com/h2non/imaginary/internal/source/http"
	objectsource "github.com/h2non/imaginary/internal/source/object"
)

type fakeObjectStorage struct {
	body []byte
	err  error
}

func (s fakeObjectStorage) Open(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	if s.err != nil {
		return nil, 0, s.err
	}
	return io.NopCloser(bytes.NewReader(s.body)), int64(len(s.body)), nil
}

func TestIndex(t *testing.T) {
	cfg := Config{PathPrefix: "/", MaxAllowedPixels: 18.0}
	ts := testServer(indexController(cfg.PathPrefix, cfg.Error))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("Invalid response status: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(body), "imaginary") == false {
		t.Fatalf("Invalid body response: %s", body)
	}
}

func TestImageEndpoints(t *testing.T) {
	cases := []struct {
		name    string
		op      image.Operation
		query   string
		expectW int
		expectH int
	}{
		{"Crop", image.Crop, "width=300", 300, 1080},
		{"Resize", image.Resize, "width=300&nocrop=false", 300, 1080},
		{"Enlarge", image.Enlarge, "width=300&height=200&nocrop=false", 300, 200},
		{"Extract", image.Extract, "top=100&left=100&areawidth=200&areaheight=120", 200, 120},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := testServer(controller(tc.op))
			buf := readFile("large.jpg")
			defer ts.Close()

			res, err := http.Post(ts.URL+"?"+tc.query, "image/jpeg", buf)
			if err != nil {
				t.Fatal("Cannot perform the request")
			}

			if res.StatusCode != 200 {
				t.Fatalf("Invalid response status: %s", res.Status)
			}

			if res.Header.Get("Content-Length") == "" {
				t.Fatal("Empty content length response")
			}

			img, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if len(img) == 0 {
				t.Fatalf("Empty response body")
			}

			if err := assertSize(img, tc.expectW, tc.expectH); err != nil {
				t.Error(err)
			}

			if bimg.DetermineImageTypeName(img) != "jpeg" {
				t.Fatalf("Invalid image type")
			}
		})
	}
}

func TestTypeAuto(t *testing.T) {
	cases := []struct {
		acceptHeader string
		expected     string
	}{
		{"", "jpeg"},
		{"image/webp,*/*", "webp"},
		{"image/png,*/*", "png"},
		{"image/webp;q=0.8,image/jpeg", "webp"},
		{"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8", "webp"}, // Chrome
	}

	for _, test := range cases {
		ts := testServer(controller(image.Crop))
		buf := readFile("large.jpg")
		url := ts.URL + "?width=300&type=auto"
		defer ts.Close()

		req, _ := http.NewRequest(http.MethodPost, url, buf)
		req.Header.Add("Content-Type", "image/jpeg")
		req.Header.Add("Accept", test.acceptHeader)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal("Cannot perform the request")
		}

		if res.StatusCode != 200 {
			t.Fatalf("Invalid response status: %s", res.Status)
		}

		if res.Header.Get("Content-Length") == "" {
			t.Fatal("Empty content length response")
		}

		img, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if len(img) == 0 {
			t.Fatalf("Empty response body")
		}

		err = assertSize(img, 300, 1080)
		if err != nil {
			t.Error(err)
		}

		if bimg.DetermineImageTypeName(img) != test.expected {
			t.Fatalf("Invalid image type")
		}

		if res.Header.Get("Vary") != "Accept" {
			t.Fatal("Vary header not set correctly")
		}
	}
}

func TestFit(t *testing.T) {
	var err error

	buf := readFile("large.jpg")
	original, _ := io.ReadAll(buf)
	err = assertSize(original, 1920, 1080)
	if err != nil {
		t.Errorf("Reference image expecations weren't met")
	}

	ts := testServer(controller(image.Fit))
	url := ts.URL + "?width=300&height=300"
	defer ts.Close()

	res, err := http.Post(url, "image/jpeg", bytes.NewReader(original))
	if err != nil {
		t.Fatal("Cannot perform the request")
	}

	if res.StatusCode != 200 {
		t.Fatalf("Invalid response status: %s", res.Status)
	}

	img, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(img) == 0 {
		t.Fatalf("Empty response body")
	}

	// The reference image has a ratio of 1.778, this should produce a height of 168.75
	err = assertSize(img, 300, 169)
	if err != nil {
		t.Error(err)
	}

	if bimg.DetermineImageTypeName(img) != "jpeg" {
		t.Fatalf("Invalid image type")
	}
}

func TestRemoteHTTPSource(t *testing.T) {
	cfg := Config{
		EnableURLSource:  true,
		MaxAllowedPixels: 18.0,
		MaxAllowedSize:   0,
		PathPrefix:       "/",
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(&source.SourceConfig{Type: bodysource.ImageSourceTypeBody}),
			objectsource.NewObjectImageSource(&source.SourceConfig{Type: objectsource.ImageSourceTypeObject}),
			fssource.NewFileSystemImageSource(&source.SourceConfig{Type: fssource.ImageSourceTypeFileSystem}),
			httpsource.NewHTTPImageSource(&source.SourceConfig{Type: httpsource.ImageSourceTypeHTTP}),
		),
	}
	fn := ImageMiddleware(cfg, cfg.Resolver)(image.Crop)

	tsImage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, _ := os.ReadFile(path.Join("../../testdata", "large.jpg"))
		_, _ = w.Write(b)
	}))
	defer tsImage.Close()

	ts := httptest.NewServer(fn)
	url := ts.URL + "?width=200&height=200&url=" + tsImage.URL
	defer ts.Close()

	res, err := http.Get(url)
	if err != nil {
		t.Fatal("Cannot perform the request")
	}
	if res.StatusCode != 200 {
		t.Fatalf("Invalid response status: %d", res.StatusCode)
	}

	img, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(img) == 0 {
		t.Fatalf("Empty response body")
	}

	err = assertSize(img, 200, 200)
	if err != nil {
		t.Error(err)
	}

	if bimg.DetermineImageTypeName(img) != "jpeg" {
		t.Fatalf("Invalid image type")
	}
}

func TestInvalidRemoteHTTPSource(t *testing.T) {
	cfg := Config{
		EnableURLSource:  true,
		MaxAllowedPixels: 18.0,
		PathPrefix:       "/",
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(&source.SourceConfig{Type: bodysource.ImageSourceTypeBody}),
			objectsource.NewObjectImageSource(&source.SourceConfig{Type: objectsource.ImageSourceTypeObject}),
			fssource.NewFileSystemImageSource(&source.SourceConfig{Type: fssource.ImageSourceTypeFileSystem}),
			httpsource.NewHTTPImageSource(&source.SourceConfig{Type: httpsource.ImageSourceTypeHTTP}),
		),
	}
	fn := ImageMiddleware(cfg, cfg.Resolver)(image.Crop)

	tsImage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(400)
	}))
	defer tsImage.Close()

	ts := httptest.NewServer(fn)
	url := ts.URL + "?width=200&height=200&url=" + tsImage.URL
	defer ts.Close()

	res, err := http.Get(url)
	if err != nil {
		t.Fatal("Request failed")
	}
	if res.StatusCode != 400 {
		t.Fatalf("Invalid response status: %d", res.StatusCode)
	}
}

func TestMountDirectory(t *testing.T) {
	cfg := Config{
		Mount:            "../../testdata",
		MaxAllowedPixels: 18.0,
		PathPrefix:       "/",
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(&source.SourceConfig{Type: bodysource.ImageSourceTypeBody}),
			objectsource.NewObjectImageSource(&source.SourceConfig{Type: objectsource.ImageSourceTypeObject}),
			fssource.NewFileSystemImageSource(&source.SourceConfig{
				Type:      fssource.ImageSourceTypeFileSystem,
				MountPath: "../../testdata",
			}),
			httpsource.NewHTTPImageSource(&source.SourceConfig{Type: httpsource.ImageSourceTypeHTTP}),
		),
	}
	fn := ImageMiddleware(cfg, cfg.Resolver)(image.Crop)

	ts := httptest.NewServer(fn)
	url := ts.URL + "?width=200&height=200&file=large.jpg"
	defer ts.Close()

	res, err := http.Get(url)
	if err != nil {
		t.Fatal("Cannot perform the request")
	}
	if res.StatusCode != 200 {
		t.Fatalf("Invalid response status: %d", res.StatusCode)
	}

	img, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(img) == 0 {
		t.Fatalf("Empty response body")
	}

	err = assertSize(img, 200, 200)
	if err != nil {
		t.Error(err)
	}

	if bimg.DetermineImageTypeName(img) != "jpeg" {
		t.Fatalf("Invalid image type")
	}
}

func TestMountInvalidDirectory(t *testing.T) {
	cfg := Config{
		Mount:            "_invalid_",
		MaxAllowedPixels: 18.0,
		PathPrefix:       "/",
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(&source.SourceConfig{Type: bodysource.ImageSourceTypeBody}),
			objectsource.NewObjectImageSource(&source.SourceConfig{Type: objectsource.ImageSourceTypeObject}),
			fssource.NewFileSystemImageSource(&source.SourceConfig{
				Type:      fssource.ImageSourceTypeFileSystem,
				MountPath: "_invalid_",
			}),
			httpsource.NewHTTPImageSource(&source.SourceConfig{Type: httpsource.ImageSourceTypeHTTP}),
		),
	}
	fn := ImageMiddleware(cfg, cfg.Resolver)(image.Crop)
	ts := httptest.NewServer(fn)
	url := ts.URL + "?top=100&left=100&areawidth=200&areaheight=120&file=large.jpg"
	defer ts.Close()

	res, err := http.Get(url)
	if err != nil {
		t.Fatal("Cannot perform the request")
	}

	if res.StatusCode != 400 {
		t.Fatalf("Invalid response status: %d", res.StatusCode)
	}
}

func TestMountInvalidPath(t *testing.T) {
	cfg := Config{
		Mount:      "_invalid_",
		PathPrefix: "/",
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(&source.SourceConfig{Type: bodysource.ImageSourceTypeBody}),
			objectsource.NewObjectImageSource(&source.SourceConfig{Type: objectsource.ImageSourceTypeObject}),
			fssource.NewFileSystemImageSource(&source.SourceConfig{
				Type:      fssource.ImageSourceTypeFileSystem,
				MountPath: "_invalid_",
			}),
			httpsource.NewHTTPImageSource(&source.SourceConfig{Type: httpsource.ImageSourceTypeHTTP}),
		),
	}
	fn := ImageMiddleware(cfg, cfg.Resolver)(image.Crop)
	ts := httptest.NewServer(fn)
	url := ts.URL + "?top=100&left=100&areawidth=200&areaheight=120&file=../../large.jpg"
	defer ts.Close()

	res, err := http.Get(url)
	if err != nil {
		t.Fatal("Cannot perform the request")
	}

	if res.StatusCode != 400 {
		t.Fatalf("Invalid response status: %s", res.Status)
	}
}

func TestPathThumbnailObjectStorage(t *testing.T) {
	buf, _ := os.ReadFile(path.Join("../../testdata", "large.jpg"))
	cfg := testConfigWithObjectStorage(buf)

	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/thumbnail/300x200q85/uploads/2026/05/image.jpg")
	if err != nil {
		t.Fatal("Cannot perform the request")
	}
	if res.StatusCode != 200 {
		t.Fatalf("Invalid response status: %d", res.StatusCode)
	}

	img, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(img) == 0 {
		t.Fatalf("Empty response body")
	}

	err = assertSize(img, 300, 200)
	if err != nil {
		t.Error(err)
	}
}

func TestPathThumbnailObjectStorageWithType(t *testing.T) {
	buf, _ := os.ReadFile(path.Join("../../testdata", "large.jpg"))
	cfg := testConfigWithObjectStorage(buf)

	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/thumbnail/300x200q85.webp/uploads/2026/05/image.jpg?embed=true")
	if err != nil {
		t.Fatal("Cannot perform the request")
	}
	if res.StatusCode != 200 {
		t.Fatalf("Invalid response status: %d", res.StatusCode)
	}

	img, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(img) == 0 {
		t.Fatalf("Empty response body")
	}

	err = assertSize(img, 300, 200)
	if err != nil {
		t.Error(err)
	}

	if bimg.DetermineImageTypeName(img) != "webp" {
		t.Fatalf("Invalid image type")
	}
}

func TestPathThumbnailInvalidSpec(t *testing.T) {
	cfg := testConfigWithObjectStorage([]byte("image"))

	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/thumbnail/300x0q85/uploads/2026/05/image.jpg")
	if err != nil {
		t.Fatal("Cannot perform the request")
	}
	if res.StatusCode != 400 {
		t.Fatalf("Invalid response status: %d", res.StatusCode)
	}
}

func TestPathThumbnailDisabledEndpoint(t *testing.T) {
	buf, _ := os.ReadFile(path.Join("../../testdata", "large.jpg"))
	cfg := testConfigWithObjectStorage(buf)
	cfg.Endpoints = EndpointSet{"thumbnail"}

	ts := httptest.NewServer(NewServerMux(cfg))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/thumbnail/300x200/uploads/2026/05/image.jpg")
	if err != nil {
		t.Fatal("Cannot perform the request")
	}
	if res.StatusCode != 501 {
		t.Fatalf("Invalid response status: %d", res.StatusCode)
	}
}

func TestObjectStorageSource(t *testing.T) {
	buf, _ := os.ReadFile(path.Join("../../testdata", "large.jpg"))
	cfg := Config{
		PathPrefix:       "/",
		MaxAllowedPixels: 18.0,
		ObjectStorage:    fakeObjectStorage{body: buf},
		Resolver: source.NewResolver(
			bodysource.NewBodyImageSource(&source.SourceConfig{Type: bodysource.ImageSourceTypeBody}),
			objectsource.NewObjectImageSource(&source.SourceConfig{
				Type:          objectsource.ImageSourceTypeObject,
				ObjectStorage: fakeObjectStorage{body: buf},
			}),
			fssource.NewFileSystemImageSource(&source.SourceConfig{Type: fssource.ImageSourceTypeFileSystem}),
			httpsource.NewHTTPImageSource(&source.SourceConfig{Type: httpsource.ImageSourceTypeHTTP}),
		),
	}
	fn := ImageMiddleware(cfg, cfg.Resolver)(image.Thumbnail)

	ts := httptest.NewServer(fn)
	defer ts.Close()

	res, err := http.Get(ts.URL + "?width=300&height=200&type=webp&object=uploads/2026/05/image.jpg")
	if err != nil {
		t.Fatal("Cannot perform the request")
	}
	if res.StatusCode != 200 {
		t.Fatalf("Invalid response status: %d", res.StatusCode)
	}

	img, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(img) == 0 {
		t.Fatalf("Empty response body")
	}

	err = assertSize(img, 300, 200)
	if err != nil {
		t.Error(err)
	}

	if bimg.DetermineImageTypeName(img) != "webp" {
		t.Fatalf("Invalid image type")
	}
}

func controller(op image.Operation) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		cfg := Config{MaxAllowedPixels: 18.0, PathPrefix: "/"}
		imageHandler(w, r, buf, op, cfg)
	}
}

func testServer(fn func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(fn))
}

func readFile(file string) io.Reader {
	buf, _ := os.Open(path.Join("../../testdata", file))
	return buf
}

func assertSize(buf []byte, width, height int) error {
	size, err := bimg.NewImage(buf).Size()
	if err != nil {
		return err
	}
	if size.Width != width || size.Height != height {
		return fmt.Errorf("invalid image size: %dx%d, expected: %dx%d", size.Width, size.Height, width, height)
	}
	return nil
}

// testConfigWithObjectStorage creates a Config suitable for testing path thumbnail
// and object storage endpoints. It wires a fake object storage into both the
// Config.ObjectStorage field and the Resolver's ObjectImageSource.
func testConfigWithObjectStorage(body []byte) Config {
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
