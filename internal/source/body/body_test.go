package body

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/h2non/imaginary/internal/source"
)

func TestSourceBodyMatch(t *testing.T) {
	u, _ := url.Parse("http://foo")
	req := &http.Request{Method: http.MethodPost, URL: u}
	src := NewBodyImageSource(&source.SourceConfig{})

	if !src.Matches(req) {
		t.Error("Cannot match the request")
	}
}

func TestBodyImageSource(t *testing.T) {
	var body []byte
	var err error

	src := NewBodyImageSource(&source.SourceConfig{})
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

	file, _ := os.Open("../../../testdata/large.jpg")
	r, _ := http.NewRequest(http.MethodPost, "http://foo/bar", file)
	w := httptest.NewRecorder()
	fakeHandler(w, r)

	buf, _ := ioutil.ReadFile("../../../testdata/large.jpg")
	if len(body) != len(buf) {
		t.Error("Invalid response body")
	}
}

func testReadBody(t *testing.T) {
	var body []byte
	var err error

	src := NewBodyImageSource(&source.SourceConfig{})
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

	file, _ := os.Open("../../../testdata/large.jpg")
	r, _ := http.NewRequest(http.MethodPost, "http://foo/bar", file)
	w := httptest.NewRecorder()
	fakeHandler(w, r)

	buf, _ := ioutil.ReadFile("../../../testdata/large.jpg")
	if len(body) != len(buf) {
		t.Error("Invalid response body")
	}
}
