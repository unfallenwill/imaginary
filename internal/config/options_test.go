package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileStorageOptions(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "imaginary.yaml")

	t.Setenv("TEST_STORAGE_SECRET", "secret")
	err := os.WriteFile(configFile, []byte(`
storage:
  type: s3-compatible
  bucket: images
  region: ap-shanghai
  endpoint: https://cos.ap-shanghai.myqcloud.com
  access_key: access
  secret_key: ${TEST_STORAGE_SECRET}
  force_path_style: true
  key_prefix: origin
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Storage == nil {
		t.Fatal("expected storage to be non-nil")
	}
	if cfg.Storage.Type != "s3-compatible" {
		t.Fatalf("Invalid storage type: %s", cfg.Storage.Type)
	}
	if cfg.Storage.SecretKey != "secret" {
		t.Fatalf("Invalid storage secret: %s", cfg.Storage.SecretKey)
	}
	if !cfg.Storage.ForcePathStyle {
		t.Fatal("Invalid storage force_path_style")
	}
}

func TestLoadFileAllFields(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "imaginary.yaml")

	err := os.WriteFile(configFile, []byte(`
port: 9090
addr: "0.0.0.0"
cors: true
log_level: warning
concurrency: 10
burst: 50
enable_url_source: true
api_key: test-key
mount: /data
http_cache_ttl: 3600
return_size: true
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Port == nil || *cfg.Port != 9090 {
		t.Fatal("expected port 9090")
	}
	if cfg.CORS == nil || !*cfg.CORS {
		t.Fatal("expected cors true")
	}
	if cfg.LogLevel == nil || *cfg.LogLevel != "warning" {
		t.Fatal("expected log_level warning")
	}
	if cfg.ReturnSize == nil || !*cfg.ReturnSize {
		t.Fatal("expected return_size true")
	}
}
