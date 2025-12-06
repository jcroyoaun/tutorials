// Package storage provides S3 storage capabilities for the sentinel application.
package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client handles uploading discovery state to S3.
type S3Client struct {
	client   *s3.Client
	bucket   string
	fileName string
}

// NewS3Client creates a new S3 client configured for the specified bucket.
// It uses the default AWS SDK credential chain which supports:
// - Environment variables
// - Shared credentials file
// - IAM roles for service accounts (IRSA) when running in EKS
func NewS3Client(ctx context.Context, bucket, fileName string) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	return &S3Client{
		client:   client,
		bucket:   bucket,
		fileName: fileName,
	}, nil
}

// Upload serializes the provided data to JSON and uploads it to S3.
func (c *S3Client) Upload(ctx context.Context, data any) error {
	// Marshal data to pretty-printed JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling json: %w", err)
	}

	// Upload to S3
	_, err = c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(c.fileName),
		Body:        bytes.NewReader(jsonData),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("uploading to s3: %w", err)
	}

	return nil
}
