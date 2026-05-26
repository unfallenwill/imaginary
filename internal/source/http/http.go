package httpsource

import (
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

const ImageSourceTypeHTTP source.ImageSourceType = "http"
const URLQueryKey = "url"

type HTTPImageSource struct {
	Config *source.SourceConfig
}

func NewHTTPImageSource(config *source.SourceConfig) source.ImageSource {
	return &HTTPImageSource{config}
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

func (s *HTTPImageSource) GetImage(req *http.Request) ([]byte, error) {
	u, err := parseURL(req)
	if err != nil {
		return nil, image.ErrInvalidImageURL
	}
	if shouldRestrictOrigin(u, s.Config.AllowedOrigins) {
		return nil, image.ErrOriginNotAllowed
	}
	return s.fetchImage(u, req)
}

func (s *HTTPImageSource) fetchImage(url *url.URL, ireq *http.Request) ([]byte, error) {
	if s.Config.MaxAllowedSize > 0 {
		req, err := newHTTPRequest(s, ireq, http.MethodHead, url)
		if err != nil {
			return nil, err
		}
		res, err := s.httpClient().Do(req)
		if err != nil {
			return nil, image.WrapUpstreamError("error fetching remote http image headers", err)
		}
		_ = res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode > 206 {
			return nil, image.NewUpstreamError(fmt.Sprintf("error fetching remote http image headers: (status=%d) (url=%s)", res.StatusCode, req.URL.String()))
		}

		contentLength, _ := strconv.Atoi(res.Header.Get("Content-Length"))
		if contentLength > s.Config.MaxAllowedSize {
			return nil, image.ErrContentTooLarge
		}
	}

	req, err := newHTTPRequest(s, ireq, http.MethodGet, url)
	if err != nil {
		return nil, err
	}
	res, err := s.httpClient().Do(req)
	if err != nil {
		return nil, image.WrapUpstreamError("error fetching remote http image", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != 200 {
		return nil, image.NewUpstreamError(fmt.Sprintf("error fetching remote http image: (status=%d) (url=%s)", res.StatusCode, req.URL.String()))
	}

	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, image.WrapUpstreamError(fmt.Sprintf("unable to read image from response body (url=%s)", req.URL.String()), err)
	}
	return buf, nil
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

func newHTTPRequest(s *HTTPImageSource, ireq *http.Request, method string, url *url.URL) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ireq.Context(), method, url.String(), nil)
	if err != nil {
		return nil, image.WrapInvalidParamError("invalid request URL", err)
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

func shouldRestrictOrigin(url *url.URL, origins []*url.URL) bool {
	if len(origins) == 0 {
		return false
	}

	for _, origin := range origins {
		if origin.Host == url.Host {
			if strings.HasPrefix(url.Path, origin.Path) {
				return false
			}
		}

		if len(origin.Host) >= 2 && origin.Host[0:2] == "*." {
			if url.Host == origin.Host[2:] {
				if strings.HasPrefix(url.Path, origin.Path) {
					return false
				}
			}

			if strings.HasSuffix(url.Host, origin.Host[1:]) {
				if strings.HasPrefix(url.Path, origin.Path) {
					return false
				}
			}
		}
	}

	return true
}
