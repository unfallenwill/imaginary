package source

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

// ObjectStorage represents a configured object storage backend.
// Defined here at the consumer, following Go's "define interfaces where they're used" principle.
type ObjectStorage interface {
	Open(ctx context.Context, key string) (io.ReadCloser, int64, error)
}

type ImageSourceType string
type ImageSourceFactoryFunction func(*SourceConfig) ImageSource

type SourceConfig struct {
	AuthForwarding bool
	Authorization  string
	MountPath      string
	Type           ImageSourceType
	ForwardHeaders []string
	AllowedOrigins []*url.URL
	MaxAllowedSize int
	ObjectStorage  ObjectStorage
}

type ImageSource interface {
	Matches(*http.Request) bool
	GetImage(*http.Request) ([]byte, error)
}

// Resolver matches incoming requests to configured image sources.
type Resolver struct {
	sources []ImageSource
}

// NewResolver creates a request resolver with a deterministic source order.
func NewResolver(sources ...ImageSource) *Resolver {
	return &Resolver{sources: append([]ImageSource(nil), sources...)}
}

// Match returns the first configured source that can handle the request.
func (r *Resolver) Match(req *http.Request) ImageSource {
	if r == nil {
		return nil
	}
	for _, source := range r.sources {
		if source.Matches(req) {
			return source
		}
	}
	return nil
}
