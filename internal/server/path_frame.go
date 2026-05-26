package server

import (
	"net/http"
	"path"
	"strings"

	img "github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
)

func pathFrameController(cfg Config, resolver *source.Resolver) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			ErrorReply(w, r, errMethodNotAllowed, cfg.Error)
			return
		}
		if cfg.ObjectStorage == nil {
			ErrorReply(w, r, img.ErrNotImplemented, cfg.Error)
			return
		}
		if cfg.Endpoints.Contains("frame") {
			ErrorReply(w, r, img.ErrNotImplemented, cfg.Error)
			return
		}

		key, err := parsePathFrameKey(r.URL.Path, cfg.PathPrefix)
		if err != nil {
			ErrorReply(w, r, err, cfg.Error)
			return
		}

		req := withPathFrameQuery(r, key)
		frameController(cfg, resolver)(w, req)
	}
}

func withPathFrameQuery(r *http.Request, key string) *http.Request {
	req := new(http.Request)
	*req = *r

	u := *r.URL
	q := cloneQuery(r.URL.Query())
	q.Set("object", key)
	u.RawQuery = q.Encode()
	req.URL = &u

	return req
}

func parsePathFrameKey(requestPath, prefix string) (string, error) {
	pattern := framePathPattern(prefix)
	key := strings.TrimPrefix(requestPath, pattern)
	if key == "" {
		return "", img.NewInvalidParamError("invalid frame path: missing object key")
	}
	if err := source.ValidateObjectKey(key); err != nil {
		return "", img.NewInvalidParamError("invalid object key")
	}
	return key, nil
}

func framePathPattern(prefix string) string {
	pattern := path.Join(prefix, "/frame/")
	if !strings.HasSuffix(pattern, "/") {
		pattern += "/"
	}
	return pattern
}
