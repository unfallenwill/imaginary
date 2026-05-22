package config

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Port != 8088 {
		t.Fatalf("expected default port 8088, got %d", cfg.Port)
	}
	if cfg.PathPrefix != "/" {
		t.Fatalf("expected default path prefix /, got %s", cfg.PathPrefix)
	}
	if cfg.MaxAllowedPixels != 18.0 {
		t.Fatalf("expected default max pixels 18.0, got %f", cfg.MaxAllowedPixels)
	}
	if cfg.HTTPCacheTTL != -1 {
		t.Fatalf("expected default http-cache-ttl -1, got %d", cfg.HTTPCacheTTL)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected default log level info, got %s", cfg.LogLevel)
	}
}

func TestParseBasicFlags(t *testing.T) {
	cfg, err := Parse([]string{"-p", "9090", "-cors", "-log-level", "warning"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Port)
	}
	if !cfg.CORS {
		t.Fatal("expected CORS to be true")
	}
	if cfg.LogLevel != "warning" {
		t.Fatalf("expected log level warning, got %s", cfg.LogLevel)
	}
}

func TestParseEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "7070")
	t.Setenv("URL_SIGNATURE_KEY", "a-really-long-signature-key-that-is-32-chars")
	t.Setenv("GOLANG_LOG", "error")

	cfg, err := Parse([]string{"-enable-url-signature"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 7070 {
		t.Fatalf("expected port 7070 from env, got %d", cfg.Port)
	}
	if cfg.URLSignatureKey != "a-really-long-signature-key-that-is-32-chars" {
		t.Fatalf("expected URL signature key from env, got %s", cfg.URLSignatureKey)
	}
	if cfg.LogLevel != "error" {
		t.Fatalf("expected log level error from env, got %s", cfg.LogLevel)
	}
}

func TestValidateMountSuccess(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mount = t.TempDir()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error for valid mount, got %v", err)
	}
}

func TestValidateMountNotDirectory(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "notadir")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	cfg := DefaultConfig()
	cfg.Mount = f.Name()
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for file mount path")
	}
}

func TestValidateMountNonexistent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mount = "/nonexistent/path/12345"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for nonexistent mount path")
	}
}

func TestValidateMountRoot(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mount = "/"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for root mount path")
	}
}

func TestValidateHTTPCacheTTL(t *testing.T) {
	cfg := DefaultConfig()
	cfg.HTTPCacheTTL = 100
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error for valid TTL, got %v", err)
	}

	cfg.HTTPCacheTTL = 999999999
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for TTL too large")
	}
}

func TestValidateURLSignature(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnableURLSignature = true

	// Missing key
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing signature key")
	}

	// Key too short
	cfg.URLSignatureKey = "short"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for short signature key")
	}

	// Valid key
	cfg.URLSignatureKey = "a-really-long-signature-key-that-is-32-chars"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error for valid key, got %v", err)
	}
}

func TestParseForwardHeaders(t *testing.T) {
	cfg, err := Parse([]string{"-forward-headers", "X-Custom, X-Token"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.ForwardHeaders) != 2 {
		t.Fatalf("expected 2 forward headers, got %d", len(cfg.ForwardHeaders))
	}
	if cfg.ForwardHeaders[0] != "X-Custom" {
		t.Fatalf("expected X-Custom, got %s", cfg.ForwardHeaders[0])
	}
	if cfg.ForwardHeaders[1] != "X-Token" {
		t.Fatalf("expected X-Token, got %s", cfg.ForwardHeaders[1])
	}
}

func TestParseAllowedOrigins(t *testing.T) {
	cfg, err := Parse([]string{"-allowed-origins", "http://localhost,http://example.com/path"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0].Host != "localhost" {
		t.Fatalf("expected localhost, got %s", cfg.AllowedOrigins[0].Host)
	}
	// Paths should have trailing slash added
	if cfg.AllowedOrigins[1].Path != "/path/" {
		t.Fatalf("expected /path/, got %s", cfg.AllowedOrigins[1].Path)
	}
}

func TestParseAllowedOriginsWildcard(t *testing.T) {
	cfg, err := Parse([]string{"-allowed-origins", "http://example.com/*"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.AllowedOrigins) != 1 {
		t.Fatalf("expected 1 origin, got %d", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0].Path != "/" {
		t.Fatalf("expected / after wildcard strip, got %s", cfg.AllowedOrigins[0].Path)
	}
}

func TestParseEndpoints(t *testing.T) {
	cfg, err := Parse([]string{"-disable-endpoints", "form, crop, HEALTH"})
	if err != nil {
		t.Fatal(err)
	}
	endpoints := cfg.ParseEndpoints()
	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d", len(endpoints))
	}
	if endpoints[0] != "form" {
		t.Fatalf("expected form, got %s", endpoints[0])
	}
	if endpoints[1] != "crop" {
		t.Fatalf("expected crop, got %s", endpoints[1])
	}
	if endpoints[2] != "health" {
		t.Fatalf("expected health, got %s", endpoints[2])
	}
}

func TestLoadStorage(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.json")
	t.Setenv("TEST_STORAGE_SECRET", "secret")
	if err := os.WriteFile(configFile, []byte(`{
		"storage": {
			"type": "s3-compatible",
			"bucket": "images",
			"region": "us-east-1"
		}
	}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Parse([]string{"-config", configFile})
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.LoadStorage(); err != nil {
		t.Fatal(err)
	}
	if cfg.Storage.Type != "s3-compatible" {
		t.Fatalf("expected s3-compatible, got %s", cfg.Storage.Type)
	}
	if cfg.Storage.Bucket != "images" {
		t.Fatalf("expected images, got %s", cfg.Storage.Bucket)
	}
}

func TestResolvePlaceholderNone(t *testing.T) {
	cfg := DefaultConfig()
	data, enabled, err := cfg.ResolvePlaceholder()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected placeholder to be disabled")
	}
	if data != nil {
		t.Fatal("expected nil placeholder data")
	}
}

func TestResolvePlaceholderBuiltIn(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnablePlaceholder = true
	data, enabled, err := cfg.ResolvePlaceholder()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("expected placeholder to be enabled")
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty placeholder data")
	}
}

func TestToServerConfig(t *testing.T) {
	cfg, err := Parse([]string{"-p", "9090", "-cors", "-key", "mykey"})
	if err != nil {
		t.Fatal(err)
	}
	serverCfg := cfg.ToServerConfig(nil, false)
	if serverCfg.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", serverCfg.Port)
	}
	if !serverCfg.CORS {
		t.Fatal("expected CORS to be true")
	}
	if serverCfg.APIKey != "mykey" {
		t.Fatalf("expected API key mykey, got %s", serverCfg.APIKey)
	}
}

func TestParseEmptyForwardHeaders(t *testing.T) {
	cfg, err := Parse([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ForwardHeaders != nil {
		t.Fatalf("expected nil forward headers, got %v", cfg.ForwardHeaders)
	}
}

func TestParseEmptyOrigins(t *testing.T) {
	cfg, err := Parse([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AllowedOrigins != nil {
		t.Fatalf("expected nil origins, got %v", cfg.AllowedOrigins)
	}
}

func TestParseAllFlags(t *testing.T) {
	cfg, err := Parse([]string{
		"-a", "0.0.0.0",
		"-p", "3000",
		"-path-prefix", "/api",
		"-cors",
		"-enable-url-source",
		"-enable-url-signature",
		"-url-signature-key", "01234567890123456789012345678901",
		"-max-allowed-size", "10485760",
		"-max-allowed-resolution", "50.0",
		"-key", "test-api-key",
		"-mount", t.TempDir(),
		"-http-cache-ttl", "3600",
		"-http-read-timeout", "30",
		"-http-write-timeout", "30",
		"-concurrency", "5",
		"-burst", "50",
		"-mrelease", "60",
		"-log-level", "debug",
		"-return-size",
		"-enable-auth-forwarding",
		"-authorization", "Bearer token",
		"-forward-headers", "X-Request-Id",
		"-placeholder-status", "404",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "0.0.0.0" {
		t.Fatalf("expected addr 0.0.0.0, got %s", cfg.Addr)
	}
	if cfg.Port != 3000 {
		t.Fatalf("expected port 3000, got %d", cfg.Port)
	}
	if cfg.PathPrefix != "/api" {
		t.Fatalf("expected path prefix /api, got %s", cfg.PathPrefix)
	}
	if cfg.MaxAllowedSize != 10485760 {
		t.Fatalf("expected max allowed size 10485760, got %d", cfg.MaxAllowedSize)
	}
	if cfg.MaxAllowedPixels != 50.0 {
		t.Fatalf("expected max pixels 50.0, got %f", cfg.MaxAllowedPixels)
	}
	if !cfg.ReturnSize {
		t.Fatal("expected return-size to be true")
	}
}

func TestURLParsing(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"http://localhost", "localhost"},
		{"http://example.com/path", "example.com"},
	}
	for _, tt := range tests {
		u, err := url.Parse(tt.input)
		if err != nil {
			t.Fatal(err)
		}
		if u.Host != tt.expect {
			t.Fatalf("expected host %s, got %s", tt.expect, u.Host)
		}
	}
}
