package source_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/h2non/imaginary/internal/config"
	"github.com/h2non/imaginary/internal/source"

	// Register source providers via init()
	_ "github.com/h2non/imaginary/internal/source/body"
	_ "github.com/h2non/imaginary/internal/source/fs"
	_ "github.com/h2non/imaginary/internal/source/http"
)

func TestMatchSource(t *testing.T) {
	opts := config.ServerOptions{EnableURLSource: true}
	source.LoadSources(opts)

	u, _ := url.Parse("http://foo?url=http://bar/image.jpg")
	req := &http.Request{Method: http.MethodGet, URL: u}

	src := source.MatchSource(req)
	if src == nil {
		t.Error("Cannot match image source")
	}
}
