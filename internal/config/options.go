package config

import (
	"encoding/json"
	"os"
)

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
