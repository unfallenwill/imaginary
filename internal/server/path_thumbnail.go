package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/h2non/imaginary/internal/config"
	img "github.com/h2non/imaginary/internal/image"
)

const objectStorageTimeout = 30 * time.Second

type pathThumbnailParams struct {
	Width   int
	Height  int
	Quality int
	Type    string
	Key     string
}

func pathThumbnailController(o config.ServerOptions) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			img.ErrorReply(r, w, img.ErrMethodNotAllowed, o)
			return
		}
		if o.ObjectStorage == nil {
			img.ErrorReply(r, w, img.ErrNotImplemented, o)
			return
		}
		if o.Endpoints.Contains("thumbnail") {
			img.ErrorReply(r, w, img.ErrNotImplemented, o)
			return
		}

		params, err := parsePathThumbnailParams(r.URL.Path, o)
		if err != nil {
			img.ErrorReply(r, w, img.NewError(err.Error(), http.StatusBadRequest), o)
			return
		}
		if o.MaxAllowedPixels > 0 && (float64(params.Width)*float64(params.Height))/1000000 > o.MaxAllowedPixels {
			img.ErrorReply(r, w, img.ErrResolutionTooBig, o)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), objectStorageTimeout)
		defer cancel()

		body, contentLength, err := o.ObjectStorage.Open(ctx, params.Key)
		if err != nil {
			img.ErrorReply(r, w, img.NewError(fmt.Sprintf("Error while fetching object: %s", err.Error()), http.StatusBadRequest), o)
			return
		}
		defer func() { _ = body.Close() }()

		if o.MaxAllowedSize > 0 && contentLength > int64(o.MaxAllowedSize) {
			img.ErrorReply(r, w, img.NewError(fmt.Sprintf("Object size %d exceeds maximum allowed %d bytes", contentLength, o.MaxAllowedSize), http.StatusRequestEntityTooLarge), o)
			return
		}

		buf, err := readObjectBody(body, o.MaxAllowedSize)
		if err != nil {
			img.ErrorReply(r, w, img.NewError(err.Error(), http.StatusBadRequest), o)
			return
		}
		if len(buf) == 0 {
			img.ErrorReply(r, w, img.ErrEmptyBody, o)
			return
		}

		req := withPathThumbnailQuery(r, params)
		imageHandler(w, req, buf, img.Thumbnail, o)
	}
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

func withPathThumbnailQuery(r *http.Request, params pathThumbnailParams) *http.Request {
	req := new(http.Request)
	*req = *r

	u := *r.URL
	q := cloneQuery(r.URL.Query())
	q.Set("width", strconv.Itoa(params.Width))
	q.Set("height", strconv.Itoa(params.Height))
	if params.Quality > 0 {
		q.Set("quality", strconv.Itoa(params.Quality))
	}
	if params.Type != "" {
		q.Set("type", params.Type)
	}
	u.RawQuery = q.Encode()
	req.URL = &u

	return req
}

func cloneQuery(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for key, value := range values {
		cloned[key] = append([]string(nil), value...)
	}
	return cloned
}

func parsePathThumbnailParams(requestPath string, o config.ServerOptions) (pathThumbnailParams, error) {
	var params pathThumbnailParams

	prefix := thumbnailPathPattern(o)
	if !strings.HasPrefix(requestPath, prefix) {
		return params, fmt.Errorf("invalid thumbnail path")
	}

	parts := strings.SplitN(strings.TrimPrefix(requestPath, prefix), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return params, fmt.Errorf("invalid thumbnail path")
	}

	width, height, quality, imageType, err := parseThumbnailSpec(parts[0])
	if err != nil {
		return params, err
	}
	if err := validateObjectKey(parts[1]); err != nil {
		return params, err
	}

	params.Width = width
	params.Height = height
	params.Quality = quality
	params.Type = imageType
	params.Key = parts[1]

	return params, nil
}

func parseThumbnailSpec(spec string) (int, int, int, string, error) {
	spec, imageType, err := splitThumbnailSpecType(spec)
	if err != nil {
		return 0, 0, 0, "", err
	}

	dimensions := spec
	quality := 0

	if idx := strings.LastIndex(spec, "q"); idx > -1 {
		dimensions = spec[:idx]
		if dimensions == "" || spec[idx+1:] == "" {
			return 0, 0, 0, "", fmt.Errorf("invalid thumbnail spec")
		}
		var err error
		quality, err = strconv.Atoi(spec[idx+1:])
		if err != nil || quality < 1 || quality > 100 {
			return 0, 0, 0, "", fmt.Errorf("invalid thumbnail quality")
		}
	}

	parts := strings.Split(dimensions, "x")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, 0, 0, "", fmt.Errorf("invalid thumbnail dimensions")
	}

	width, err := strconv.Atoi(parts[0])
	if err != nil || width <= 0 {
		return 0, 0, 0, "", fmt.Errorf("invalid thumbnail width")
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil || height <= 0 {
		return 0, 0, 0, "", fmt.Errorf("invalid thumbnail height")
	}

	return width, height, quality, imageType, nil
}

func splitThumbnailSpecType(spec string) (string, string, error) {
	parts := strings.Split(spec, ".")
	if len(parts) == 1 {
		return spec, "", nil
	}
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid thumbnail type")
	}
	imageType := strings.ToLower(parts[1])
	if img.ImageType(imageType) == 0 {
		return "", "", fmt.Errorf("invalid thumbnail type")
	}

	return parts[0], imageType, nil
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

func thumbnailPathPattern(o config.ServerOptions) string {
	pattern := path.Join(o.PathPrefix, "/thumbnail/")
	if !strings.HasSuffix(pattern, "/") {
		pattern += "/"
	}
	return pattern
}
