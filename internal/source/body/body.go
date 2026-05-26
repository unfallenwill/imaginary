package body

import (
	"io"
	"net/http"
	"strings"

	"github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
)

const formFieldName = "file"
const maxMemory int64 = 1024 * 1024 * 64

// defaultMaxBodySize is the default maximum request body size (100 MB)
// when MaxAllowedSize is not explicitly configured.
const defaultMaxBodySize = 100 * 1024 * 1024

type BodyImageSource struct {
	maxAllowedSize int
}

func NewBodyImageSource(maxAllowedSize int) source.ImageSource {
	return &BodyImageSource{maxAllowedSize: maxAllowedSize}
}

func (s *BodyImageSource) Matches(r *http.Request) bool {
	return r.Method == http.MethodPost || r.Method == http.MethodPut
}

func (s *BodyImageSource) GetImage(r *http.Request) ([]byte, error) {
	if isFormBody(r) {
		return readFormBody(r, s.maxSize())
	}
	return readRawBody(r, s.maxSize())
}

func (s *BodyImageSource) maxSize() int {
	if s.maxAllowedSize > 0 {
		return s.maxAllowedSize
	}
	return defaultMaxBodySize
}

func isFormBody(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/")
}

func readFormBody(r *http.Request, maxSize int) ([]byte, error) {
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		return nil, image.Wrap(image.KindInvalidParam, "failed to parse multipart form", err)
	}

	file, _, err := r.FormFile(formFieldName)
	if err != nil {
		return nil, image.Wrap(image.KindInvalidParam, "missing form file field", err)
	}
	defer func() { _ = file.Close() }()

	limited := io.LimitReader(file, int64(maxSize)+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return nil, image.Wrap(image.KindInvalidParam, "error reading form file", err)
	}
	if len(buf) > maxSize {
		return nil, image.ErrContentTooLarge
	}
	if len(buf) == 0 {
		return nil, image.ErrEmptyBody
	}

	return buf, nil
}

func readRawBody(r *http.Request, maxSize int) ([]byte, error) {
	limited := io.LimitReader(r.Body, int64(maxSize)+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return nil, image.Wrap(image.KindInvalidParam, "error reading request body", err)
	}
	if len(buf) > maxSize {
		return nil, image.ErrContentTooLarge
	}
	return buf, nil
}
