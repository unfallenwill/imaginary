package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	img "github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
)

type pathThumbnailParams struct {
	Width   int
	Height  int
	Quality int
	Type    string
	Key     string
}

func pathThumbnailController(cfg Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			ErrorReply(w, r, img.ErrMethodNotAllowed, cfg.Error)
			return
		}
		if cfg.ObjectStorage == nil {
			ErrorReply(w, r, img.ErrNotImplemented, cfg.Error)
			return
		}
		if cfg.Endpoints.Contains("thumbnail") {
			ErrorReply(w, r, img.ErrNotImplemented, cfg.Error)
			return
		}

		params, err := parsePathThumbnailParams(r.URL.Path, cfg.PathPrefix)
		if err != nil {
			replyError(w, r, err, img.KindInvalidParam, cfg)
			return
		}
		if cfg.MaxAllowedPixels > 0 && (float64(params.Width)*float64(params.Height))/1000000 > cfg.MaxAllowedPixels {
			ErrorReply(w, r, img.ErrResolutionTooBig, cfg.Error)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), source.ObjectStorageTimeout)
		defer cancel()

		body, contentLength, err := cfg.ObjectStorage.Open(ctx, params.Key)
		if err != nil {
			ErrorReply(w, r, img.WrapError("Error while fetching object", img.KindUpstream, err), cfg.Error)
			return
		}
		defer func() { _ = body.Close() }()

		if cfg.MaxAllowedSize > 0 && contentLength > int64(cfg.MaxAllowedSize) {
			ErrorReply(w, r, img.NewError(fmt.Sprintf("Object size %d exceeds maximum allowed %d bytes", contentLength, cfg.MaxAllowedSize), img.KindInvalidParam), cfg.Error)
			return
		}

		buf, err := source.ReadObjectBody(body, cfg.MaxAllowedSize)
		if err != nil {
			replyError(w, r, err, img.KindUpstream, cfg)
			return
		}
		if len(buf) == 0 {
			ErrorReply(w, r, img.ErrEmptyBody, cfg.Error)
			return
		}

		req := withPathThumbnailQuery(r, params)
		imageHandler(w, req, buf, img.Thumbnail, cfg)
	}
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

func parsePathThumbnailParams(requestPath, prefix string) (pathThumbnailParams, error) {
	var params pathThumbnailParams

	pattern := thumbnailPathPattern(prefix)
	if !strings.HasPrefix(requestPath, pattern) {
		return params, fmt.Errorf("invalid thumbnail path")
	}

	parts := strings.SplitN(strings.TrimPrefix(requestPath, pattern), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return params, fmt.Errorf("invalid thumbnail path")
	}

	width, height, quality, imageType, err := parseThumbnailSpec(parts[0])
	if err != nil {
		return params, err
	}
	if err := source.ValidateObjectKey(parts[1]); err != nil {
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

func thumbnailPathPattern(prefix string) string {
	pattern := path.Join(prefix, "/thumbnail/")
	if !strings.HasSuffix(pattern, "/") {
		pattern += "/"
	}
	return pattern
}
