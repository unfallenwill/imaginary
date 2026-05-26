package source

import (
	"context"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/h2non/imaginary/internal/image"
)

// HTTPClient is the interface for HTTP clients used by source implementations.
// This allows dependency injection for testing.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ObjectStorage represents a configured object storage backend.
// Defined here at the consumer, following Go's "define interfaces where they're used" principle.
type ObjectStorage interface {
	Open(ctx context.Context, key string) (io.ReadCloser, int64, error)
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

// ObjectStorageTimeout is the default timeout for object storage operations.
const ObjectStorageTimeout = 30 * time.Second

// ValidateObjectKey checks that an object storage key is safe to use.
func ValidateObjectKey(key string) error {
	if key == "" || strings.Contains(key, "\\") {
		return image.ErrInvalidFilePath
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == "." || segment == ".." {
			return image.ErrInvalidFilePath
		}
	}

	cleaned := path.Clean("/" + key)
	if cleaned == "/" {
		return image.ErrInvalidFilePath
	}

	return nil
}

// ReadObjectBody reads the full body from an object storage reader,
// enforcing an optional maximum size limit.
func ReadObjectBody(body io.Reader, maxAllowedSize int) ([]byte, error) {
	if maxAllowedSize <= 0 {
		buf, err := io.ReadAll(body)
		if err != nil {
			return nil, image.Wrap(image.KindUpstream, "error reading object body", err)
		}
		return buf, nil
	}

	limited := io.LimitReader(body, int64(maxAllowedSize)+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return nil, image.Wrap(image.KindUpstream, "error reading object body", err)
	}
	if len(buf) > maxAllowedSize {
		return nil, image.ErrContentTooLarge
	}

	return buf, nil
}
