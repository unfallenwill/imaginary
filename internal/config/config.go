package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/h2non/bimg"
	"gopkg.in/yaml.v3"

	img "github.com/h2non/imaginary/internal/image"
)

const maxHTTPCacheTTLSecs = 31_556_926 // ~1 tropical year in seconds (365.2425 days)

// Config holds all configuration parsed from command-line flags,
// environment variables, and optional YAML config file.
type Config struct {
	// Meta
	ShowHelp    bool
	ShowVersion bool

	// Network
	Addr     string
	Port     int
	CertFile string
	KeyFile  string

	// Timeouts
	HTTPReadTimeout  int
	HTTPWriteTimeout int

	// Logging
	LogLevel string

	// Routing
	PathPrefix string

	// Middleware
	CORS        bool
	APIKey      string
	Concurrency int

	// Caching
	HTTPCacheTTL int

	// URL source
	EnableURLSource    bool
	AuthForwarding     bool
	Authorization      string
	ForwardHeaders     []string
	AllowedOrigins     []*url.URL
	MaxAllowedSize     int
	MaxAllowedPixels   float64
	EnableURLSignature bool
	URLSignatureKey    string

	// Filesystem
	Mount string

	// Endpoints
	DisableEndpoints string

	// Placeholder
	EnablePlaceholder bool
	Placeholder       string
	PlaceholderStatus int

	// Resource management
	CPUs     int
	MRelease int

	// Response
	ReturnSize bool

	// Config file
	ConfigFile string

	// Resolved from config file
	Storage StorageOptions

	// internal: raw flag values parsed before conversion
	forwardHeadersRaw string
	allowedOriginsRaw string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Port:             8088,
		PathPrefix:       "/",
		MaxAllowedPixels: 18.0,
		HTTPReadTimeout:  60,
		HTTPWriteTimeout: 60,
		LogLevel:         "info",
		MRelease:         30,
		CPUs:             runtime.GOMAXPROCS(-1),
		HTTPCacheTTL:     -1,
	}
}

// Parse parses command-line flags, config file, and environment variables into a Config.
// Priority: code defaults < config file < CLI flags < env vars.
func Parse(args []string) (Config, error) {
	cfg := DefaultConfig()

	fs := flag.NewFlagSet("imaginary", flag.ContinueOnError)

	fs.BoolVar(&cfg.ShowHelp, "h", false, "Show help")
	fs.BoolVar(&cfg.ShowHelp, "help", false, "Show help")
	fs.BoolVar(&cfg.ShowVersion, "v", false, "Show version")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "Show version")

	fs.StringVar(&cfg.Addr, "a", cfg.Addr, "Bind address")
	fs.IntVar(&cfg.Port, "p", cfg.Port, "Port to listen")
	fs.StringVar(&cfg.PathPrefix, "path-prefix", cfg.PathPrefix, "URL path prefix to listen to")
	fs.BoolVar(&cfg.CORS, "cors", cfg.CORS, "Enable CORS support")
	fs.BoolVar(&cfg.AuthForwarding, "enable-auth-forwarding", cfg.AuthForwarding, "Forward Authorization header to image source server")
	fs.BoolVar(&cfg.EnableURLSource, "enable-url-source", cfg.EnableURLSource, "Enable remote HTTP URL image source processing")
	fs.BoolVar(&cfg.EnablePlaceholder, "enable-placeholder", cfg.EnablePlaceholder, "Enable image placeholder on error")
	fs.BoolVar(&cfg.EnableURLSignature, "enable-url-signature", cfg.EnableURLSignature, "Enable URL signature")
	fs.StringVar(&cfg.URLSignatureKey, "url-signature-key", cfg.URLSignatureKey, "URL signature key (32 characters minimum)")
	fs.Var(&stringValue{dst: &cfg.allowedOriginsRaw}, "allowed-origins", "Restrict remote image source to certain origins (comma-separated)")
	fs.IntVar(&cfg.MaxAllowedSize, "max-allowed-size", cfg.MaxAllowedSize, "Max size of HTTP image source in bytes")
	fs.Float64Var(&cfg.MaxAllowedPixels, "max-allowed-resolution", cfg.MaxAllowedPixels, "Max image resolution in megapixels")
	fs.StringVar(&cfg.APIKey, "key", cfg.APIKey, "API key for authorization")
	fs.StringVar(&cfg.Mount, "mount", cfg.Mount, "Mount local directory")
	fs.StringVar(&cfg.CertFile, "certfile", cfg.CertFile, "TLS certificate file path")
	fs.StringVar(&cfg.KeyFile, "keyfile", cfg.KeyFile, "TLS private key file path")
	fs.StringVar(&cfg.Authorization, "authorization", cfg.Authorization, "Constant Authorization header for image source servers")
	fs.Var(&stringValue{dst: &cfg.forwardHeadersRaw}, "forward-headers", "Custom headers to forward to image source (comma-separated)")
	fs.StringVar(&cfg.Placeholder, "placeholder", cfg.Placeholder, "Custom placeholder image path")
	fs.IntVar(&cfg.PlaceholderStatus, "placeholder-status", cfg.PlaceholderStatus, "HTTP status for placeholder response")
	fs.StringVar(&cfg.DisableEndpoints, "disable-endpoints", cfg.DisableEndpoints, "Comma-separated endpoints to disable")
	fs.IntVar(&cfg.HTTPCacheTTL, "http-cache-ttl", cfg.HTTPCacheTTL, "HTTP cache TTL in seconds")
	fs.IntVar(&cfg.HTTPReadTimeout, "http-read-timeout", cfg.HTTPReadTimeout, "HTTP read timeout in seconds")
	fs.IntVar(&cfg.HTTPWriteTimeout, "http-write-timeout", cfg.HTTPWriteTimeout, "HTTP write timeout in seconds")
	fs.IntVar(&cfg.Concurrency, "concurrency", cfg.Concurrency, "Max concurrent image processing requests")
	fs.IntVar(&cfg.MRelease, "mrelease", cfg.MRelease, "Memory release interval in seconds")
	fs.IntVar(&cfg.CPUs, "cpus", cfg.CPUs, "Number of CPU cores to use")
	fs.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Log level: info, warning, error")
	fs.BoolVar(&cfg.ReturnSize, "return-size", cfg.ReturnSize, "Return image size in HTTP headers")
	fs.StringVar(&cfg.ConfigFile, "config", cfg.ConfigFile, "YAML config file path")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	// Collect explicitly set CLI flags.
	explicit := make(map[string]struct{})
	fs.Visit(func(f *flag.Flag) {
		explicit[f.Name] = struct{}{}
	})

	// Determine config file path: -config flag > default path.
	configPath := cfg.ConfigFile
	if configPath == "" {
		configPath = defaultConfigPath()
	}

	if configPath != "" {
		fileCfg, err := loadFile(configPath)
		if err != nil {
			return Config{}, fmt.Errorf("cannot load config file: %w", err)
		}
		applyFileOverrides(&cfg, fileCfg, explicit)
	}

	cfg.applyEnvOverrides()
	cfg.ForwardHeaders = parseForwardHeaders(cfg.forwardHeadersRaw)
	cfg.AllowedOrigins = parseOrigins(cfg.allowedOriginsRaw)

	return cfg, nil
}

// defaultConfigPath returns the default config file path if it exists.
func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	p := filepath.Join(home, ".config", "imaginary", "config.yaml")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// setFileOverride sets dst to *val if val is non-nil and flagName was not
// explicitly set via CLI flags.
func setFileOverride[T any](flagName string, val *T, dst *T, explicit map[string]struct{}) {
	if val != nil {
		if _, ok := explicit[flagName]; !ok {
			*dst = *val
		}
	}
}

// applyFileOverrides applies FileConfig values to Config for fields not
// explicitly set via CLI flags.
func applyFileOverrides(cfg *Config, f fileConfig, explicit map[string]struct{}) {
	// Network
	setFileOverride("a", f.Addr, &cfg.Addr, explicit)
	setFileOverride("p", f.Port, &cfg.Port, explicit)
	setFileOverride("certfile", f.CertFile, &cfg.CertFile, explicit)
	setFileOverride("keyfile", f.KeyFile, &cfg.KeyFile, explicit)

	// Timeouts
	setFileOverride("http-read-timeout", f.HTTPReadTimeout, &cfg.HTTPReadTimeout, explicit)
	setFileOverride("http-write-timeout", f.HTTPWriteTimeout, &cfg.HTTPWriteTimeout, explicit)

	// Logging
	setFileOverride("log-level", f.LogLevel, &cfg.LogLevel, explicit)

	// Routing
	setFileOverride("path-prefix", f.PathPrefix, &cfg.PathPrefix, explicit)

	// Middleware
	setFileOverride("cors", f.CORS, &cfg.CORS, explicit)
	setFileOverride("key", f.APIKey, &cfg.APIKey, explicit)
	setFileOverride("concurrency", f.Concurrency, &cfg.Concurrency, explicit)

	// Caching
	setFileOverride("http-cache-ttl", f.HTTPCacheTTL, &cfg.HTTPCacheTTL, explicit)

	// URL source
	setFileOverride("enable-url-source", f.EnableURLSource, &cfg.EnableURLSource, explicit)
	setFileOverride("enable-auth-forwarding", f.AuthForwarding, &cfg.AuthForwarding, explicit)
	setFileOverride("authorization", f.Authorization, &cfg.Authorization, explicit)
	setFileOverride("forward-headers", f.ForwardHeaders, &cfg.forwardHeadersRaw, explicit)
	setFileOverride("allowed-origins", f.AllowedOrigins, &cfg.allowedOriginsRaw, explicit)
	setFileOverride("max-allowed-size", f.MaxAllowedSize, &cfg.MaxAllowedSize, explicit)
	setFileOverride("max-allowed-resolution", f.MaxAllowedPixels, &cfg.MaxAllowedPixels, explicit)
	setFileOverride("enable-url-signature", f.EnableURLSignature, &cfg.EnableURLSignature, explicit)
	setFileOverride("url-signature-key", f.URLSignatureKey, &cfg.URLSignatureKey, explicit)

	// Filesystem
	setFileOverride("mount", f.Mount, &cfg.Mount, explicit)

	// Endpoints
	setFileOverride("disable-endpoints", f.DisableEndpoints, &cfg.DisableEndpoints, explicit)

	// Placeholder
	setFileOverride("enable-placeholder", f.EnablePlaceholder, &cfg.EnablePlaceholder, explicit)
	setFileOverride("placeholder", f.Placeholder, &cfg.Placeholder, explicit)
	setFileOverride("placeholder-status", f.PlaceholderStatus, &cfg.PlaceholderStatus, explicit)

	// Resource management
	setFileOverride("cpus", f.CPUs, &cfg.CPUs, explicit)
	setFileOverride("mrelease", f.MRelease, &cfg.MRelease, explicit)

	// Response
	setFileOverride("return-size", f.ReturnSize, &cfg.ReturnSize, explicit)

	// Storage
	if f.Storage != nil {
		cfg.Storage = *f.Storage
	}
}

// applyEnvOverrides applies environment variable overrides for select fields.
func (c *Config) applyEnvOverrides() {
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		if p, err := strconv.Atoi(portEnv); err == nil && p > 0 {
			c.Port = p
		}
	}
	if keyEnv := os.Getenv("URL_SIGNATURE_KEY"); keyEnv != "" {
		c.URLSignatureKey = keyEnv
	}
	if logEnv := os.Getenv("GOLANG_LOG"); logEnv != "" {
		c.LogLevel = logEnv
	}
}

// Validate checks the configuration for errors.
func (c Config) Validate() error {
	if c.Mount != "" {
		src, err := os.Stat(c.Mount)
		if err != nil {
			return fmt.Errorf("error while mounting directory: %w", err)
		}
		if !src.IsDir() {
			return fmt.Errorf("mount path is not a directory: %s", c.Mount)
		}
		if c.Mount == "/" {
			return fmt.Errorf("cannot mount root directory for security reasons")
		}
	}

	if c.HTTPCacheTTL != -1 {
		if c.HTTPCacheTTL < 0 || c.HTTPCacheTTL > maxHTTPCacheTTLSecs {
			return fmt.Errorf("the -http-cache-ttl flag only accepts a value from 0 to %d", maxHTTPCacheTTLSecs)
		}
	}

	if c.EnableURLSignature {
		if c.URLSignatureKey == "" {
			return fmt.Errorf("URL signature key is required")
		}
		if len(c.URLSignatureKey) < 32 {
			return fmt.Errorf("URL signature key must be a minimum of 32 characters")
		}
	}

	return nil
}

// ResolvePlaceholder loads the placeholder image bytes if configured.
func (c Config) ResolvePlaceholder() ([]byte, bool, error) {
	if c.Placeholder != "" {
		buf, err := os.ReadFile(c.Placeholder) // #nosec G304 -- path provided by server operator
		if err != nil {
			return nil, false, fmt.Errorf("cannot start the server: %w", err)
		}
		imageType := bimg.DetermineImageType(buf)
		if !bimg.IsImageTypeSupportedByVips(imageType).Load {
			return nil, false, fmt.Errorf("placeholder image type is not supported. Only JPEG, PNG or WEBP are supported")
		}
		return buf, true, nil
	}

	if c.EnablePlaceholder {
		return img.Placeholder, true, nil
	}

	return nil, false, nil
}

// ParseEndpoints parses the disable-endpoints string into a normalized list.
func (c Config) ParseEndpoints() []string {
	if c.DisableEndpoints == "" {
		return nil
	}
	var endpoints []string
	for _, ep := range strings.Split(c.DisableEndpoints, ",") {
		if norm := strings.ToLower(strings.TrimSpace(ep)); norm != "" {
			endpoints = append(endpoints, norm)
		}
	}
	return endpoints
}

// stringValue implements flag.Value for a simple string destination.
type stringValue struct {
	dst *string
}

func (s *stringValue) String() string {
	if s.dst == nil {
		return ""
	}
	return *s.dst
}

func (s *stringValue) Set(v string) error {
	*s.dst = v
	return nil
}

// fileConfig is the YAML-parsed representation of the config file.
// All fields are pointers to distinguish "not set" from zero values.
type fileConfig struct {
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

// loadFile reads and parses a YAML config file, expanding environment
// variables in string values before decoding.
func loadFile(filename string) (fileConfig, error) {
	var cfg fileConfig

	buf, err := os.ReadFile(filename) // #nosec G304 -- config path is explicitly provided by the server operator.
	if err != nil {
		return cfg, err
	}

	err = yaml.Unmarshal([]byte(os.ExpandEnv(string(buf))), &cfg)
	return cfg, err
}

// --- internal helpers ---

func parseForwardHeaders(raw string) []string {
	if raw == "" {
		return nil
	}
	var headers []string
	for _, h := range strings.Split(raw, ",") {
		if norm := strings.TrimSpace(h); norm != "" {
			headers = append(headers, norm)
		}
	}
	return headers
}

func parseOrigins(raw string) []*url.URL {
	if raw == "" {
		return nil
	}
	var urls []*url.URL
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		u, err := url.Parse(origin)
		if err != nil {
			continue
		}
		if u.Path != "" {
			last := u.Path[len(u.Path)-1:]
			if last == "*" {
				u.Path = strings.TrimSuffix(u.Path, "*")
			} else if last != "/" {
				u.Path += "/"
			}
		}
		urls = append(urls, u)
	}
	return urls
}
