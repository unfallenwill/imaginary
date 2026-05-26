package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/h2non/bimg"
	"github.com/rs/cors"

	img "github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
	"github.com/h2non/imaginary/internal/version"
)

func Middleware(fn func(http.ResponseWriter, *http.Request), cfg Config) http.Handler {
	next := http.Handler(http.HandlerFunc(fn))

	if len(cfg.Endpoints) > 0 {
		next = filterEndpoint(next, cfg)
	}
	if cfg.Concurrency > 0 {
		next = throttle(next, cfg)
	}
	if cfg.CORS {
		next = cors.Default().Handler(next)
	}
	if cfg.APIKey != "" {
		next = authorizeClient(next, cfg)
	}
	if cfg.HTTPCacheTTL >= 0 {
		next = setCacheHeaders(next, cfg.HTTPCacheTTL)
	}

	return validate(defaultHeaders(next), cfg)
}

func ImageMiddleware(cfg Config, resolver *source.Resolver) func(img.Operation) http.Handler {
	return func(fn img.Operation) http.Handler {
		handler := validateImage(Middleware(imageController(cfg, resolver, fn), cfg), cfg)

		if cfg.EnableURLSignature {
			return validateURLSignature(handler, cfg)
		}

		return handler
	}
}

func filterEndpoint(next http.Handler, cfg Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.Endpoints.IsAllowed(r) {
			next.ServeHTTP(w, r)
			return
		}
		ErrorReply(w, r, img.ErrNotImplemented, cfg.Error)
	})
}

func throttle(next http.Handler, cfg Config) http.Handler {
	sem := cfg.ConcurrencySem
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"Too many requests","status":503}`))
		}
	})
}

func validate(next http.Handler, cfg Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			ErrorReply(w, r, errMethodNotAllowed, cfg.Error)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func validateImage(next http.Handler, cfg Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := r.URL.Path
		if r.Method == http.MethodGet && isPublicPath(reqPath) {
			next.ServeHTTP(w, r)
			return
		}

		if r.Method == http.MethodGet && cfg.Mount == "" && !cfg.EnableURLSource {
			ErrorReply(w, r, errGetMethodNotAllowed, cfg.Error)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authorizeClient(next http.Handler, cfg Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("API-Key")
		if key == "" {
			key = r.URL.Query().Get("key")
		}

		if key != cfg.APIKey {
			ErrorReply(w, r, errInvalidAPIKey, cfg.Error)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func defaultHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", fmt.Sprintf("imaginary %s (bimg %s)", version.Version, bimg.Version))
		next.ServeHTTP(w, r)
	})
}

func setCacheHeaders(next http.Handler, ttl int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer next.ServeHTTP(w, r)

		if r.Method != http.MethodGet || isPublicPath(r.URL.Path) {
			return
		}

		ttlDiff := time.Duration(ttl) * time.Second
		expires := time.Now().Add(ttlDiff)

		w.Header().Add("Expires", strings.ReplaceAll(expires.Format(time.RFC1123), "UTC", "GMT"))
		w.Header().Add("Cache-Control", getCacheControl(ttl))
	})
}

func getCacheControl(ttl int) string {
	if ttl == 0 {
		return "private, no-cache, no-store, must-revalidate"
	}
	return fmt.Sprintf("public, s-maxage=%d, max-age=%d, no-transform", ttl, ttl)
}

func isPublicPath(requestPath string) bool {
	return requestPath == "/" || requestPath == "/health" || requestPath == "/form"
}

func validateURLSignature(next http.Handler, cfg Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		sign := query.Get("sign")
		query.Del("sign")

		h := hmac.New(sha256.New, []byte(cfg.URLSignatureKey))
		_, _ = h.Write([]byte(r.URL.Path))
		_, _ = h.Write([]byte(query.Encode()))
		expectedSign := h.Sum(nil)

		urlSign, err := base64.RawURLEncoding.DecodeString(sign)
		if err != nil {
			ErrorReply(w, r, errInvalidURLSignature, cfg.Error)
			return
		}

		if !hmac.Equal(urlSign, expectedSign) {
			ErrorReply(w, r, errURLSignatureMismatch, cfg.Error)
			return
		}

		next.ServeHTTP(w, r)
	})
}
