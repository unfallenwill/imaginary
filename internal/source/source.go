package source

import (
	"net/http"
	"net/url"

	"github.com/h2non/imaginary/internal/config"
)

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
	ObjectStorage  config.ObjectStorage
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

// NewSourceConfig creates a source config from server options.
func NewSourceConfig(o config.ServerOptions, sourceType ImageSourceType) *SourceConfig {
	return &SourceConfig{
		Type:           sourceType,
		MountPath:      o.Mount,
		AuthForwarding: o.AuthForwarding,
		Authorization:  o.Authorization,
		AllowedOrigins: o.AllowedOrigins,
		MaxAllowedSize: o.MaxAllowedSize,
		ForwardHeaders: o.ForwardHeaders,
		ObjectStorage:  o.ObjectStorage,
	}
}
