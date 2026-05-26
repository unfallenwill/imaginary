package source_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/h2non/imaginary/internal/source"
	bodysource "github.com/h2non/imaginary/internal/source/body"
	httpsource "github.com/h2non/imaginary/internal/source/http"
)

func TestResolverMatch(t *testing.T) {
	resolver := source.NewResolver(
		bodysource.NewBodyImageSource(0),
		httpsource.NewHTTPImageSource(httpsource.Config{}),
	)

	u, _ := url.Parse("http://foo?url=http://bar/image.jpg")
	req := &http.Request{Method: http.MethodGet, URL: u}

	src := resolver.Match(req)
	if src == nil {
		t.Error("Cannot match image source")
	}
}
