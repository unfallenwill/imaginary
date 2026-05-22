package fs

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/h2non/imaginary/internal/source"
)

func TestFileSystemImageSource(t *testing.T) {
	var body []byte
	var err error
	const fixtureFile = "../../../testdata/large.jpg"

	src := NewFileSystemImageSource(&source.SourceConfig{MountPath: "../../../testdata"})
	fakeHandler := func(w http.ResponseWriter, r *http.Request) {
		if !src.Matches(r) {
			t.Fatal("Cannot match the request")
		}

		body, err = src.GetImage(r)
		if err != nil {
			t.Fatalf("Error while reading the body: %s", err)
		}
		_, _ = w.Write(body)
	}

	file, _ := os.Open(fixtureFile)
	r, _ := http.NewRequest(http.MethodGet, "http://foo/bar?file=large.jpg", file)
	w := httptest.NewRecorder()
	fakeHandler(w, r)

	buf, _ := ioutil.ReadFile(fixtureFile)
	if len(body) != len(buf) {
		t.Error("Invalid response body")
	}
}
