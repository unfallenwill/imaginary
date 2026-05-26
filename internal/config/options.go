package config

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
