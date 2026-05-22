package objectsource

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
)

const ImageSourceTypeObject source.ImageSourceType = "object"
const ObjectQueryKey = "object"
const objectStorageTimeout = 30 * time.Second

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
	if err := validateObjectKey(key); err != nil {
		return nil, image.ErrInvalidFilePath
	}

	ctx, cancel := context.WithTimeout(r.Context(), objectStorageTimeout)
	defer cancel()

	body, contentLength, err := s.Config.ObjectStorage.Open(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("error fetching remote object: %v", err)
	}
	defer func() { _ = body.Close() }()

	if s.Config.MaxAllowedSize > 0 && contentLength > int64(s.Config.MaxAllowedSize) {
		return nil, fmt.Errorf("object size %d exceeds maximum allowed %d bytes", contentLength, s.Config.MaxAllowedSize)
	}

	buf, err := readObjectBody(body, s.Config.MaxAllowedSize)
	if err != nil {
		return nil, err
	}
	if len(buf) == 0 {
		return nil, image.ErrEmptyBody
	}

	return buf, nil
}

func readObjectBody(body io.Reader, maxAllowedSize int) ([]byte, error) {
	if maxAllowedSize <= 0 {
		return io.ReadAll(body)
	}

	limited := io.LimitReader(body, int64(maxAllowedSize)+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(buf) > maxAllowedSize {
		return nil, fmt.Errorf("object body exceeds maximum allowed %d bytes", maxAllowedSize)
	}

	return buf, nil
}

func validateObjectKey(key string) error {
	if key == "" || strings.Contains(key, "\\") {
		return fmt.Errorf("invalid object key")
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == "." || segment == ".." {
			return fmt.Errorf("invalid object key")
		}
	}

	cleaned := path.Clean("/" + key)
	if cleaned == "/" {
		return fmt.Errorf("invalid object key")
	}

	return nil
}
