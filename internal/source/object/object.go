package objectsource

import (
	"context"
	"net/http"

	"github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
)

const ImageSourceTypeObject source.ImageSourceType = "object"
const ObjectQueryKey = "object"

type ObjectImageSource struct {
	Config *source.SourceConfig
}

func NewObjectImageSource(config *source.SourceConfig) source.ImageSource {
	return &ObjectImageSource{config}
}

func (s *ObjectImageSource) Matches(r *http.Request) bool {
	return r.Method == http.MethodGet && r.URL.Query().Get(ObjectQueryKey) != ""
}

func (s *ObjectImageSource) GetImage(r *http.Request) ([]byte, error) {
	if s.Config.ObjectStorage == nil {
		return nil, image.ErrMissingImageSource
	}

	key := r.URL.Query().Get(ObjectQueryKey)
	if err := source.ValidateObjectKey(key); err != nil {
		return nil, image.ErrInvalidFilePath
	}

	ctx, cancel := context.WithTimeout(r.Context(), source.ObjectStorageTimeout)
	defer cancel()

	body, contentLength, err := s.Config.ObjectStorage.Open(ctx, key)
	if err != nil {
		return nil, image.WrapUpstreamError("error fetching remote object", err)
	}
	defer func() { _ = body.Close() }()

	if s.Config.MaxAllowedSize > 0 && contentLength > int64(s.Config.MaxAllowedSize) {
		return nil, image.ErrContentTooLarge
	}

	buf, err := source.ReadObjectBody(body, s.Config.MaxAllowedSize)
	if err != nil {
		return nil, err
	}
	if len(buf) == 0 {
		return nil, image.ErrEmptyBody
	}

	return buf, nil
}

// ReadObjectBody delegates to source.ReadObjectBody for backward compatibility.
// Deprecated: use source.ReadObjectBody directly.
var ReadObjectBody = source.ReadObjectBody

// ValidateObjectKey delegates to source.ValidateObjectKey for backward compatibility.
// Deprecated: use source.ValidateObjectKey directly.
var ValidateObjectKey = source.ValidateObjectKey

// ObjectStorageTimeout delegates to source.ObjectStorageTimeout for backward compatibility.
// Deprecated: use source.ObjectStorageTimeout directly.
const ObjectStorageTimeout = source.ObjectStorageTimeout
