package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileStorageOptions(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "imaginary.json")

	t.Setenv("TEST_STORAGE_SECRET", "secret")
	err := os.WriteFile(configFile, []byte(`{
		"storage": {
			"type": "s3",
			"bucket": "images",
			"region": "ap-shanghai",
			"endpoint": "https://cos.ap-shanghai.myqcloud.com",
			"access_key": "access",
			"secret_key": "${TEST_STORAGE_SECRET}",
			"force_path_style": true,
			"key_prefix": "origin"
		}
	}`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	options, err := LoadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}

	if options.Storage.Type != "s3" {
		t.Fatalf("Invalid storage type: %s", options.Storage.Type)
	}
	if options.Storage.SecretKey != "secret" {
		t.Fatalf("Invalid storage secret: %s", options.Storage.SecretKey)
	}
	if !options.Storage.ForcePathStyle {
		t.Fatal("Invalid storage force_path_style")
	}
}
