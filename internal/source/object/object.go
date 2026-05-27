package objectsource

import (
	"context"
	"net/http"

	"github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
)

const ObjectQueryKey = "object"

type ObjectImageSource struct {
	storage        source.ObjectStorage
	maxAllowedSize int
}

func NewObjectImageSource(storage source.ObjectStorage, maxAllowedSize int) source.ImageSource {
	return &ObjectImageSource{storage: storage, maxAllowedSize: maxAllowedSize}
}

func (s *ObjectImageSource) Matches(r *http.Request) bool {
	return r.Method == http.MethodGet && r.URL.Query().Get(ObjectQueryKey) != ""
}

func (s *ObjectImageSource) GetImage(ctx context.Context, r *http.Request) ([]byte, error) {
	if s.storage == nil {
		return nil, image.ErrMissingImageSource
	}

	key := r.URL.Query().Get(ObjectQueryKey)
	if err := source.ValidateObjectKey(key); err != nil {
		return nil, image.ErrInvalidFilePath
	}

	body, contentLength, err := s.storage.Open(ctx, key)
	if err != nil {
		return nil, image.Wrap(image.KindUpstream, "error fetching remote object", err)
	}
	defer func() { _ = body.Close() }()

	if s.maxAllowedSize > 0 && contentLength > int64(s.maxAllowedSize) {
		return nil, image.ErrContentTooLarge
	}

	buf, err := source.ReadObjectBody(body, s.maxAllowedSize)
	if err != nil {
		return nil, err
	}
	if len(buf) == 0 {
		return nil, image.ErrEmptyBody
	}

	return buf, nil
}
