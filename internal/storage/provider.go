package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/h2non/imaginary/internal/config"
	"github.com/h2non/imaginary/internal/source"
)

// NewProvider creates an object storage provider from config.
func NewProvider(ctx context.Context, o config.StorageOptions) (source.ObjectStorage, error) {
	switch strings.ToLower(strings.TrimSpace(o.Type)) {
	case "":
		return nil, nil
	case "s3-compatible":
		return NewS3Provider(ctx, o)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", o.Type)
	}
}
