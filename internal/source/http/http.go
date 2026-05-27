package httpsource

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/h2non/imaginary/internal/image"
	"github.com/h2non/imaginary/internal/source"
	"github.com/h2non/imaginary/internal/version"
)

const URLQueryKey = "url"

// defaultMaxBodySize is the default maximum response body size (100 MB)
// when MaxAllowedSize is not explicitly configured.
const defaultMaxBodySize = 100 * 1024 * 1024

// Config holds configuration for the HTTP image source.
type Config struct {
	AuthForwarding bool
	Authorization  string
	ForwardHeaders []string
	AllowedOrigins []*url.URL
	MaxAllowedSize int
	HTTPClient     source.HTTPClient
}

type HTTPImageSource struct {
	Config *Config
}

func NewHTTPImageSource(cfg Config) source.ImageSource {
	return &HTTPImageSource{Config: &cfg}
}

func (s *HTTPImageSource) Matches(r *http.Request) bool {
	return r.Method == http.MethodGet && r.URL.Query().Get(URLQueryKey) != ""
}

func (s *HTTPImageSource) httpClient() source.HTTPClient {
	if s.Config.HTTPClient != nil {
		return s.Config.HTTPClient
	}
	return defaultHTTPClient
}

var defaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

func (s *HTTPImageSource) GetImage(ctx context.Context, req *http.Request) ([]byte, error) {
	u, err := parseURL(req)
	if err != nil {
		return nil, image.ErrInvalidImageURL
	}
	if shouldRestrictOrigin(u, s.Config.AllowedOrigins) {
		return nil, image.ErrOriginNotAllowed
	}
	return s.fetchImage(ctx, u, req)
}

func (s *HTTPImageSource) fetchImage(ctx context.Context, url *url.URL, ireq *http.Request) ([]byte, error) {
	if s.Config.MaxAllowedSize > 0 {
		req, err := newHTTPRequest(ctx, s, ireq, http.MethodHead, url)
		if err != nil {
			return nil, err
		}
		res, err := s.httpClient().Do(req)
		if err != nil {
			return nil, image.Wrap(image.KindUpstream, "error fetching remote http image headers", err)
		}
		_ = res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode > 206 {
			return nil, image.New(image.KindUpstream, fmt.Sprintf("error fetching remote http image headers: (status=%d) (url=%s)", res.StatusCode, req.URL.String()))
		}

		contentLength, _ := strconv.Atoi(res.Header.Get("Content-Length"))
		if contentLength > s.Config.MaxAllowedSize {
			return nil, image.ErrContentTooLarge
		}
	}

	req, err := newHTTPRequest(ctx, s, ireq, http.MethodGet, url)
	if err != nil {
		return nil, err
	}
	res, err := s.httpClient().Do(req)
	if err != nil {
		return nil, image.Wrap(image.KindUpstream, "error fetching remote http image", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != 200 {
		return nil, image.New(image.KindUpstream, fmt.Sprintf("error fetching remote http image: (status=%d) (url=%s)", res.StatusCode, req.URL.String()))
	}

	var reader io.Reader = res.Body
	limit := s.maxAllowedSize()
	reader = io.LimitReader(reader, int64(limit)+1)

	buf, err := io.ReadAll(reader)
	if err != nil {
		return nil, image.Wrap(image.KindUpstream, fmt.Sprintf("unable to read image from response body (url=%s)", req.URL.String()), err)
	}
	if len(buf) > limit {
		return nil, image.ErrContentTooLarge
	}
	return buf, nil
}

func (s *HTTPImageSource) maxAllowedSize() int {
	if s.Config.MaxAllowedSize > 0 {
		return s.Config.MaxAllowedSize
	}
	return defaultMaxBodySize
}

func (s *HTTPImageSource) setAuthorizationHeader(req *http.Request, ireq *http.Request) {
	auth := s.Config.Authorization
	if auth == "" {
		auth = ireq.Header.Get("X-Forward-Authorization")
	}
	if auth == "" {
		auth = ireq.Header.Get("Authorization")
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
}

func (s *HTTPImageSource) setForwardHeaders(req *http.Request, ireq *http.Request) {
	headers := s.Config.ForwardHeaders
	for _, header := range headers {
		if _, ok := ireq.Header[header]; ok {
			req.Header.Set(header, ireq.Header.Get(header))
		}
	}
}

func parseURL(request *http.Request) (*url.URL, error) {
	return url.Parse(request.URL.Query().Get(URLQueryKey))
}

func newHTTPRequest(ctx context.Context, s *HTTPImageSource, ireq *http.Request, method string, url *url.URL) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url.String(), nil)
	if err != nil {
		return nil, image.Wrap(image.KindInvalidParam, "invalid request URL", err)
	}
	req.Header.Set("User-Agent", "imaginary/"+version.Version)
	req.URL = url

	if len(s.Config.ForwardHeaders) != 0 {
		s.setForwardHeaders(req, ireq)
	}

	if s.Config.AuthForwarding || s.Config.Authorization != "" {
		s.setAuthorizationHeader(req, ireq)
	}

	return req, nil
}

// originAllowed checks if a URL matches a specific allowed origin.
func originAllowed(u *url.URL, origin *url.URL) bool {
	if origin.Host == u.Host {
		return strings.HasPrefix(u.Path, origin.Path)
	}

	if strings.HasPrefix(origin.Host, "*.") {
		suffix := origin.Host[1:] // e.g. ".example.com"
		prefix := origin.Host[2:] // e.g. "example.com"
		if u.Host == prefix || strings.HasSuffix(u.Host, suffix) {
			return strings.HasPrefix(u.Path, origin.Path)
		}
	}

	return false
}

func shouldRestrictOrigin(url *url.URL, origins []*url.URL) bool {
	if len(origins) == 0 {
		return false
	}

	for _, origin := range origins {
		if originAllowed(url, origin) {
			return false
		}
	}

	return true
}
