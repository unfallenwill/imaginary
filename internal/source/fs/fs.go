package fs

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
)

type FileSystemImageSource struct {
	mountPath string
}

func NewFileSystemImageSource(mountPath string) source.ImageSource {
	return &FileSystemImageSource{mountPath: mountPath}
}

func (s *FileSystemImageSource) Matches(r *http.Request) bool {
	file, err := s.getFileParam(r)
	if err != nil {
		return false
	}
	return r.Method == http.MethodGet && file != ""
}

func (s *FileSystemImageSource) GetImage(ctx context.Context, r *http.Request) ([]byte, error) {
	file, err := s.getFileParam(r)
	if err != nil {
		return nil, err
	}

	if file == "" {
		return nil, image.ErrMissingParamFile
	}

	file, err = s.buildPath(file)
	if err != nil {
		return nil, err
	}

	return s.read(file)
}

func (s *FileSystemImageSource) buildPath(file string) (string, error) {
	file = path.Clean(path.Join(s.mountPath, file))
	if !strings.HasPrefix(file, s.mountPath) {
		return "", image.ErrInvalidFilePath
	}
	return file, nil
}

func (s *FileSystemImageSource) read(file string) ([]byte, error) {
	buf, err := os.ReadFile(file)
	if err != nil {
		return nil, image.ErrInvalidFilePath
	}
	return buf, nil
}

func (s *FileSystemImageSource) getFileParam(r *http.Request) (string, error) {
	unescaped, err := url.QueryUnescape(r.URL.Query().Get("file"))
	if err != nil {
		return "", image.Wrap(image.KindInvalidParam, "failed to unescape file param", err)
	}

	return unescaped, nil
}
