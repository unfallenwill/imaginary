package fs

import (
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
)

const ImageSourceTypeFileSystem source.ImageSourceType = "fs"

type FileSystemImageSource struct {
	Config *source.SourceConfig
}

func NewFileSystemImageSource(config *source.SourceConfig) source.ImageSource {
	return &FileSystemImageSource{config}
}

func (s *FileSystemImageSource) Matches(r *http.Request) bool {
	file, err := s.getFileParam(r)
	if err != nil {
		return false
	}
	return r.Method == http.MethodGet && file != ""
}

func (s *FileSystemImageSource) GetImage(r *http.Request) ([]byte, error) {
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
	file = path.Clean(path.Join(s.Config.MountPath, file))
	if !strings.HasPrefix(file, s.Config.MountPath) {
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
		return "", image.WrapError("failed to unescape file param", image.KindInvalidParam, err)
	}

	return unescaped, nil
}
