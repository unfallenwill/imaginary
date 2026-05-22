package config

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
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
	Storage            StorageOptions
	ObjectStorage      ObjectStorage
}

// ObjectStorage represents a configured object storage backend.
type ObjectStorage interface {
	Open(context.Context, string) (io.ReadCloser, int64, error)
}

// StorageOptions carries object storage configuration loaded from a config file.
type StorageOptions struct {
	Type           string `json:"type"`
	Bucket         string `json:"bucket"`
	Region         string `json:"region"`
	Endpoint       string `json:"endpoint"`
	AccessKey      string `json:"access_key"`
	SecretKey      string `json:"secret_key"`
	ForcePathStyle bool   `json:"force_path_style"`
	KeyPrefix      string `json:"key_prefix"`
}

// FileOptions carries options loaded from a JSON config file.
type FileOptions struct {
	Storage StorageOptions `json:"storage"`
}

// LoadFile reads and parses a JSON config file, expanding environment variables
// in string values before decoding.
func LoadFile(filename string) (FileOptions, error) {
	var options FileOptions

	buf, err := os.ReadFile(filename) // #nosec G304 -- config path is explicitly provided by the server operator.
	if err != nil {
		return options, err
	}

	err = json.Unmarshal([]byte(os.ExpandEnv(string(buf))), &options)
	return options, err
}

// Endpoints represents a list of endpoint names to disable.
type Endpoints []string

// IsValid validates if a given HTTP request endpoint is valid or not.
func (e Endpoints) IsValid(r *http.Request) bool {
	parts := strings.Split(r.URL.Path, "/")
	endpoint := parts[len(parts)-1]
	return !e.Contains(endpoint)
}

// Contains validates if a given endpoint name is present.
func (e Endpoints) Contains(endpoint string) bool {
	for _, name := range e {
		if endpoint == name {
			return true
		}
	}
	return false
}
