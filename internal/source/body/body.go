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

type BodyImageSource struct{}

func NewBodyImageSource() source.ImageSource {
	return &BodyImageSource{}
}

func (s *BodyImageSource) Matches(r *http.Request) bool {
	return r.Method == http.MethodPost || r.Method == http.MethodPut
}

func (s *BodyImageSource) GetImage(r *http.Request) ([]byte, error) {
	if isFormBody(r) {
		return readFormBody(r)
	}
	return readRawBody(r)
}

func isFormBody(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/")
}

func readFormBody(r *http.Request) ([]byte, error) {
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		return nil, image.Wrap(image.KindInvalidParam, "failed to parse multipart form", err)
	}

	file, _, err := r.FormFile(formFieldName)
	if err != nil {
		return nil, image.Wrap(image.KindInvalidParam, "missing form file field", err)
	}
	defer func() { _ = file.Close() }()

	buf, err := io.ReadAll(file)
	if len(buf) == 0 {
		err = image.ErrEmptyBody
	}

	return buf, err
}

func readRawBody(r *http.Request) ([]byte, error) {
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, image.Wrap(image.KindInvalidParam, "error reading request body", err)
	}
	return buf, nil
}
