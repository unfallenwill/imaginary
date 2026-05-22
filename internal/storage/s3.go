package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/h2non/imaginary/internal/config"
)

// S3Provider reads objects from an S3-compatible bucket.
type S3Provider struct {
	client    *s3.Client
	bucket    string
	keyPrefix string
}

// NewS3Provider creates an S3-compatible object storage provider.
func NewS3Provider(ctx context.Context, o config.StorageOptions) (*S3Provider, error) {
	if o.Bucket == "" {
		return nil, fmt.Errorf("missing storage bucket")
	}
	if o.Region == "" {
		return nil, fmt.Errorf("missing storage region")
	}

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(o.Region),
	}
	if o.AccessKey != "" || o.SecretKey != "" {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(o.AccessKey, o.SecretKey, ""),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.UsePathStyle = o.ForcePathStyle
		if o.Endpoint != "" {
			options.BaseEndpoint = aws.String(o.Endpoint)
		}
	})

	return &S3Provider{
		client:    client,
		bucket:    o.Bucket,
		keyPrefix: strings.Trim(o.KeyPrefix, "/"),
	}, nil
}

// Open returns an object body and its content length, if known.
func (p *S3Provider) Open(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	objectKey := cleanObjectKey(key)
	if p.keyPrefix != "" {
		objectKey = path.Join(p.keyPrefix, objectKey)
	}

	out, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, 0, err
	}

	return out.Body, aws.ToInt64(out.ContentLength), nil
}

func cleanObjectKey(key string) string {
	return strings.TrimLeft(path.Clean("/"+key), "/")
}
