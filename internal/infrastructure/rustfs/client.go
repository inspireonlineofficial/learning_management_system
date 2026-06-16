package rustfs

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// StorageClient defines the interface for object storage operations
type StorageClient interface {
	PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error
	PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
	DeleteObject(ctx context.Context, bucket, key string) error
}

// Client implements StorageClient for RustFS (S3-compatible)
type Client struct {
	s3Client *s3.S3
}

// NewClient creates a new RustFS client
func NewClient(endpoint, accessKey, secretKey, region string) (*Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		S3ForcePathStyle: aws.Bool(true), // Required for S3-compatible services
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &Client{
		s3Client: s3.New(sess),
	}, nil
}

// PutObject uploads an object to RustFS
func (c *Client) PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error {
	_, err := c.s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          aws.ReadSeekCloser(r),
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

// PresignGetURL generates a presigned URL for downloading an object
func (c *Client) PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	req, _ := c.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	url, err := req.Presign(ttl)
	if err != nil {
		return "", fmt.Errorf("failed to presign URL: %w", err)
	}

	return url, nil
}

// DeleteObject deletes an object from RustFS
func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := c.s3Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}
