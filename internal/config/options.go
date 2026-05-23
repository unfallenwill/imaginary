package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// StorageOptions carries object storage configuration loaded from a config file.
type StorageOptions struct {
	Type           string `yaml:"type"`
	Bucket         string `yaml:"bucket"`
	Region         string `yaml:"region"`
	Endpoint       string `yaml:"endpoint"`
	AccessKey      string `yaml:"access_key"`
	SecretKey      string `yaml:"secret_key"`
	ForcePathStyle bool   `yaml:"force_path_style"`
	KeyPrefix      string `yaml:"key_prefix"`
}

// FileConfig represents the full YAML config file structure.
// Pointer fields distinguish between "not set" (nil) and a zero value.
type FileConfig struct {
	Addr               *string         `yaml:"addr"`
	Port               *int            `yaml:"port"`
	CertFile           *string         `yaml:"cert_file"`
	KeyFile            *string         `yaml:"key_file"`
	HTTPReadTimeout    *int            `yaml:"http_read_timeout"`
	HTTPWriteTimeout   *int            `yaml:"http_write_timeout"`
	LogLevel           *string         `yaml:"log_level"`
	PathPrefix         *string         `yaml:"path_prefix"`
	CORS               *bool           `yaml:"cors"`
	APIKey             *string         `yaml:"api_key"`
	Concurrency        *int            `yaml:"concurrency"`
	Burst              *int            `yaml:"burst"`
	HTTPCacheTTL       *int            `yaml:"http_cache_ttl"`
	EnableURLSource    *bool           `yaml:"enable_url_source"`
	AuthForwarding     *bool           `yaml:"auth_forwarding"`
	Authorization      *string         `yaml:"authorization"`
	ForwardHeaders     *string         `yaml:"forward_headers"`
	AllowedOrigins     *string         `yaml:"allowed_origins"`
	MaxAllowedSize     *int            `yaml:"max_allowed_size"`
	MaxAllowedPixels   *float64        `yaml:"max_allowed_pixels"`
	EnableURLSignature *bool           `yaml:"enable_url_signature"`
	URLSignatureKey    *string         `yaml:"url_signature_key"`
	Mount              *string         `yaml:"mount"`
	DisableEndpoints   *string         `yaml:"disable_endpoints"`
	EnablePlaceholder  *bool           `yaml:"enable_placeholder"`
	Placeholder        *string         `yaml:"placeholder"`
	PlaceholderStatus  *int            `yaml:"placeholder_status"`
	CPUs               *int            `yaml:"cpus"`
	MRelease           *int            `yaml:"mrelease"`
	ReturnSize         *bool           `yaml:"return_size"`
	Storage            *StorageOptions `yaml:"storage"`
}

// LoadFile reads and parses a YAML config file, expanding environment variables
// in string values before decoding.
func LoadFile(filename string) (FileConfig, error) {
	var cfg FileConfig

	buf, err := os.ReadFile(filename) // #nosec G304 -- config path is explicitly provided by the server operator.
	if err != nil {
		return cfg, err
	}

	err = yaml.Unmarshal([]byte(os.ExpandEnv(string(buf))), &cfg)
	return cfg, err
}
