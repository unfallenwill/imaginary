package config

import (
	"net/http"
	"net/url"
	"strings"
)

// ServerOptions carries the full runtime configuration for the imaginary HTTP server.
type ServerOptions struct {
	Port               int
	Burst              int
	Concurrency        int
	HTTPCacheTTL       int
	HTTPReadTimeout    int
	HTTPWriteTimeout   int
	MaxAllowedSize     int
	MaxAllowedPixels   float64
	CORS               bool
	Gzip               bool // deprecated
	AuthForwarding     bool
	EnableURLSource    bool
	EnablePlaceholder  bool
	EnableURLSignature bool
	URLSignatureKey    string
	Address            string
	PathPrefix         string
	APIKey             string
	Mount              string
	CertFile           string
	KeyFile            string
	Authorization      string
	Placeholder        string
	PlaceholderStatus  int
	ForwardHeaders     []string
	PlaceholderImage   []byte
	Endpoints          Endpoints
	AllowedOrigins     []*url.URL
	LogLevel           string
	ReturnSize         bool
}

// Endpoints represents a list of endpoint names to disable.
type Endpoints []string

// IsValid validates if a given HTTP request endpoint is valid or not.
func (e Endpoints) IsValid(r *http.Request) bool {
	parts := strings.Split(r.URL.Path, "/")
	endpoint := parts[len(parts)-1]
	for _, name := range e {
		if endpoint == name {
			return false
		}
	}
	return true
}
