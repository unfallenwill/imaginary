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

	img "github.com/h2non/imaginary/internal/image"
)

const maxHTTPCacheTTLSecs = 31_556_926 // ~1 tropical year in seconds (365.2425 days)

// CLIConfig holds all configuration parsed from command-line flags,
// environment variables, and optional YAML config file.
type CLIConfig struct {
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
	Burst       int

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

	// Deprecated
	Gzip bool

	// Resolved from config file
	Storage StorageOptions

	// internal: raw flag values parsed before conversion
	forwardHeadersRaw string
	allowedOriginsRaw string
}

// DefaultConfig returns a CLIConfig with sensible defaults.
func DefaultConfig() CLIConfig {
	return CLIConfig{
		Port:             8088,
		PathPrefix:       "/",
		MaxAllowedPixels: 18.0,
		HTTPReadTimeout:  60,
		HTTPWriteTimeout: 60,
		LogLevel:         "info",
		Burst:            100,
		MRelease:         30,
		CPUs:             runtime.GOMAXPROCS(-1),
		HTTPCacheTTL:     -1,
	}
}

// Parse parses command-line flags, config file, and environment variables into a CLIConfig.
// Priority: code defaults < config file < CLI flags < env vars.
func Parse(args []string) (CLIConfig, error) {
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
	fs.BoolVar(&cfg.Gzip, "gzip", cfg.Gzip, "Enable gzip compression (deprecated)")
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
	fs.IntVar(&cfg.Burst, "burst", cfg.Burst, "Deprecated: no longer used")
	fs.IntVar(&cfg.MRelease, "mrelease", cfg.MRelease, "Memory release interval in seconds")
	fs.IntVar(&cfg.CPUs, "cpus", cfg.CPUs, "Number of CPU cores to use")
	fs.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Log level: info, warning, error")
	fs.BoolVar(&cfg.ReturnSize, "return-size", cfg.ReturnSize, "Return image size in HTTP headers")
	fs.StringVar(&cfg.ConfigFile, "config", cfg.ConfigFile, "YAML config file path")

	if err := fs.Parse(args); err != nil {
		return CLIConfig{}, err
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
		fileCfg, err := LoadFile(configPath)
		if err != nil {
			return CLIConfig{}, fmt.Errorf("cannot load config file: %w", err)
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

// applyFileOverrides applies FileConfig values to CLIConfig for fields not
// explicitly set via CLI flags.
func applyFileOverrides(cfg *CLIConfig, f FileConfig, explicit map[string]struct{}) {
	setString := func(flagName string, val *string, dst *string) {
		if val != nil {
			if _, ok := explicit[flagName]; !ok {
				*dst = *val
			}
		}
	}
	setInt := func(flagName string, val *int, dst *int) {
		if val != nil {
			if _, ok := explicit[flagName]; !ok {
				*dst = *val
			}
		}
	}
	setBool := func(flagName string, val *bool, dst *bool) {
		if val != nil {
			if _, ok := explicit[flagName]; !ok {
				*dst = *val
			}
		}
	}
	setFloat64 := func(flagName string, val *float64, dst *float64) {
		if val != nil {
			if _, ok := explicit[flagName]; !ok {
				*dst = *val
			}
		}
	}

	// Network
	setString("a", f.Addr, &cfg.Addr)
	setInt("p", f.Port, &cfg.Port)
	setString("certfile", f.CertFile, &cfg.CertFile)
	setString("keyfile", f.KeyFile, &cfg.KeyFile)

	// Timeouts
	setInt("http-read-timeout", f.HTTPReadTimeout, &cfg.HTTPReadTimeout)
	setInt("http-write-timeout", f.HTTPWriteTimeout, &cfg.HTTPWriteTimeout)

	// Logging
	setString("log-level", f.LogLevel, &cfg.LogLevel)

	// Routing
	setString("path-prefix", f.PathPrefix, &cfg.PathPrefix)

	// Middleware
	setBool("cors", f.CORS, &cfg.CORS)
	setString("key", f.APIKey, &cfg.APIKey)
	setInt("concurrency", f.Concurrency, &cfg.Concurrency)
	setInt("burst", f.Burst, &cfg.Burst)

	// Caching
	setInt("http-cache-ttl", f.HTTPCacheTTL, &cfg.HTTPCacheTTL)

	// URL source
	setBool("enable-url-source", f.EnableURLSource, &cfg.EnableURLSource)
	setBool("enable-auth-forwarding", f.AuthForwarding, &cfg.AuthForwarding)
	setString("authorization", f.Authorization, &cfg.Authorization)
	setString("forward-headers", f.ForwardHeaders, &cfg.forwardHeadersRaw)
	setString("allowed-origins", f.AllowedOrigins, &cfg.allowedOriginsRaw)
	setInt("max-allowed-size", f.MaxAllowedSize, &cfg.MaxAllowedSize)
	setFloat64("max-allowed-resolution", f.MaxAllowedPixels, &cfg.MaxAllowedPixels)
	setBool("enable-url-signature", f.EnableURLSignature, &cfg.EnableURLSignature)
	setString("url-signature-key", f.URLSignatureKey, &cfg.URLSignatureKey)

	// Filesystem
	setString("mount", f.Mount, &cfg.Mount)

	// Endpoints
	setString("disable-endpoints", f.DisableEndpoints, &cfg.DisableEndpoints)

	// Placeholder
	setBool("enable-placeholder", f.EnablePlaceholder, &cfg.EnablePlaceholder)
	setString("placeholder", f.Placeholder, &cfg.Placeholder)
	setInt("placeholder-status", f.PlaceholderStatus, &cfg.PlaceholderStatus)

	// Resource management
	setInt("cpus", f.CPUs, &cfg.CPUs)
	setInt("mrelease", f.MRelease, &cfg.MRelease)

	// Response
	setBool("return-size", f.ReturnSize, &cfg.ReturnSize)

	// Storage
	if f.Storage != nil {
		cfg.Storage = *f.Storage
	}
}

// applyEnvOverrides applies environment variable overrides for select fields.
func (c *CLIConfig) applyEnvOverrides() {
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
func (c CLIConfig) Validate() error {
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

	if c.Gzip {
		fmt.Println("warning: -gzip flag is deprecated and will not have effect")
	}

	return nil
}

// ResolvePlaceholder loads the placeholder image bytes if configured.
func (c CLIConfig) ResolvePlaceholder() ([]byte, bool, error) {
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
func (c CLIConfig) ParseEndpoints() []string {
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
