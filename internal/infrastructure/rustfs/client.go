package rustfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// StorageClient defines the interface for object storage operations
type StorageClient interface {
	PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
	PresignGetURLWithOptions(ctx context.Context, bucket, key string, ttl time.Duration, opts PresignOptions) (string, error)
	PresignPutURL(ctx context.Context, bucket, key string, ttl time.Duration, contentType string) (string, error)
	HeadObject(ctx context.Context, bucket, key string) (ObjectInfo, error)
	DeleteObject(ctx context.Context, bucket, key string) error
}

// ObjectInfo describes an existing object. Returned by HeadObject so callers
// can confirm a presigned PUT actually landed and learn its size / content-type.
type ObjectInfo struct {
	Size        int64
	ContentType string
}

// CompletedPart is one entry in the parts list sent to CompleteMultipartUpload.
// PartNumber is 1-based; ETags come from the UploadPart responses and must be
// replayed verbatim (including quotes — S3 returns them with quotes embedded).
type CompletedPart struct {
	PartNumber int
	ETag       string
}

// PresignOptions tunes a presigned GET request so the storage layer can serve
// the object with browser-friendly headers (cache, content-type fallback, range).
type PresignOptions struct {
	// ResponseContentType overrides the stored Content-Type when the URL is
	// fetched. Useful when the upload path stored application/octet-stream
	// but we know the file is video/mp4.
	ResponseContentType string
	// ResponseCacheControl is sent back as Cache-Control on the response. The
	// default for video is a long max-age with immutable so browsers can keep
	// the URL response in disk cache for the full TTL of the presign.
	ResponseCacheControl string
	// ResponseContentDisposition forces the browser to handle the file inline
	// (play rather than download). Defaults to inline.
	ResponseContentDisposition string
}

// Client implements StorageClient for RustFS (S3-compatible)
type Client struct {
	s3Client *s3.S3
}

// NewClient creates a new RustFS client
func NewClient(endpoint, accessKey, secretKey, region string) (*Client, error) {
	endpointConfig := resolveEndpointConfig(endpoint, region)
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(endpointConfig.endpoint),
		Region:           aws.String(endpointConfig.region),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		S3ForcePathStyle: aws.Bool(endpointConfig.forcePathStyle),
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

var digitalOceanRegionHostPattern = regexp.MustCompile(`^[a-z]{3}[0-9]\.digitaloceanspaces\.com$`)

type endpointConfig struct {
	endpoint       string
	region         string
	forcePathStyle bool
}

func resolveEndpointConfig(endpoint, region string) endpointConfig {
	endpoint = normalizeEndpoint(endpoint)
	if isDigitalOceanSpacesEndpoint(endpoint) {
		return endpointConfig{
			endpoint:       endpoint,
			region:         "us-east-1",
			forcePathStyle: false,
		}
	}
	return endpointConfig{
		endpoint:       endpoint,
		region:         region,
		forcePathStyle: true,
	}
}

func normalizeEndpoint(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return endpoint
	}

	parts := strings.Split(parsed.Hostname(), ".")
	if len(parts) < 4 {
		return endpoint
	}

	regionHost := strings.Join(parts[1:], ".")
	if !digitalOceanRegionHostPattern.MatchString(regionHost) {
		return endpoint
	}

	host := regionHost
	if port := parsed.Port(); port != "" {
		host += ":" + port
	}
	parsed.Host = host
	return parsed.String()
}

func isDigitalOceanSpacesEndpoint(endpoint string) bool {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return false
	}
	return digitalOceanRegionHostPattern.MatchString(parsed.Hostname())
}

// PresignGetURL generates a presigned URL for downloading an object.
func (c *Client) PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	return c.PresignGetURLWithOptions(ctx, bucket, key, ttl, PresignOptions{})
}

// PresignGetURLWithOptions is PresignGetURL with cache and content-type overrides.
// Browsers use these headers to decide whether to re-fetch on seek/scrub and to
// dispatch byte-range requests cleanly across origins.
func (c *Client) PresignGetURLWithOptions(ctx context.Context, bucket, key string, ttl time.Duration, opts PresignOptions) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if opts.ResponseContentType != "" {
		input.ResponseContentType = aws.String(opts.ResponseContentType)
	}
	if opts.ResponseCacheControl != "" {
		input.ResponseCacheControl = aws.String(opts.ResponseCacheControl)
	}
	if opts.ResponseContentDisposition != "" {
		input.ResponseContentDisposition = aws.String(opts.ResponseContentDisposition)
	}
	req, _ := c.s3Client.GetObjectRequest(input)

	url, err := req.Presign(ttl)
	if err != nil {
		return "", fmt.Errorf("failed to presign URL: %w", err)
	}

	return url, nil
}

// PresignPutURL generates a presigned URL the browser can PUT to directly. This
// keeps the Go API process out of the upload path so a 2 GB lesson upload no
// longer streams through the backend server.
func (c *Client) PresignPutURL(ctx context.Context, bucket, key string, ttl time.Duration, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}
	req, _ := c.s3Client.PutObjectRequest(input)

	url, err := req.Presign(ttl)
	if err != nil {
		return "", fmt.Errorf("failed to presign PUT URL: %w", err)
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

// HeadObject returns the size and content-type of an object. Used to confirm
// that a presigned PUT completed and to surface the stored file size back to
// the API caller.
func (c *Client) HeadObject(ctx context.Context, bucket, key string) (ObjectInfo, error) {
	out, err := c.s3Client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("failed to head object: %w", err)
	}
	info := ObjectInfo{}
	if out.ContentLength != nil {
		info.Size = *out.ContentLength
	}
	if out.ContentType != nil {
		info.ContentType = *out.ContentType
	}
	return info, nil
}

// GetObject streams an object's bytes back. The caller is responsible for
// closing the returned reader. Used by the thumbnail pipeline which needs to
// re-read the just-uploaded video file to extract a poster frame.
func (c *Client) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	out, err := c.s3Client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	return out.Body, nil
}

// MultipartUploadState tracks the server-side state of a S3 multipart upload.
// The client receives an upload id on CreateMultipartUpload and reuses it for
// every UploadPart call; the parts list is sent to CompleteMultipartUpload to
// assemble the final object.
type MultipartUploadState struct {
	UploadID string
}

// CreateMultipartUpload begins a multipart upload and returns an upload id.
// Use this for files larger than ~10 MB so the browser can resume across
// reloads: each part is independently retryable and its byte range is
// recorded in IndexedDB.
func (c *Client) CreateMultipartUpload(ctx context.Context, bucket, key, contentType string) (MultipartUploadState, error) {
	out, err := c.s3Client.CreateMultipartUploadWithContext(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return MultipartUploadState{}, fmt.Errorf("create multipart upload: %w", err)
	}
	return MultipartUploadState{UploadID: *out.UploadId}, nil
}

// UploadPart uploads a single chunk of a multipart upload. The returned ETag
// must be saved client-side and replayed to CompleteMultipartUpload in order.
// The aws-sdk-go v1 UploadPart input requires an io.ReadSeeker; we type-assert
// and gracefully fall back to a bytes.Buffer when given a non-seekable stream
// (e.g. an HTTP body the transcoder is piping through). The S3 minimum part
// size is 5 MB except for the last part, so callers should buffer smaller
// chunks before calling.
func (c *Client) UploadPart(ctx context.Context, bucket, key, uploadID string, partNumber int, body io.Reader) (string, error) {
	var seeker io.ReadSeeker
	if rs, ok := body.(io.ReadSeeker); ok {
		seeker = rs
	} else {
		// Fallback: buffer the body into memory. Acceptable because the
		// chunked uploader on the frontend always sends a Blob, and the
		// transcoder always sends a pre-sized *bytes.Reader.
		buf, err := io.ReadAll(body)
		if err != nil {
			return "", fmt.Errorf("buffer part %d: %w", partNumber, err)
		}
		seeker = bytes.NewReader(buf)
	}
	out, err := c.s3Client.UploadPartWithContext(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int64(int64(partNumber)),
		Body:       seeker,
	})
	if err != nil {
		return "", fmt.Errorf("upload part %d: %w", partNumber, err)
	}
	if out.ETag == nil {
		return "", fmt.Errorf("upload part %d returned no ETag", partNumber)
	}
	return *out.ETag, nil
}

// CompleteMultipartUpload assembles the final object from the uploaded parts.
// Parts must be sorted by part number; mismatches are an error from S3.
func (c *Client) CompleteMultipartUpload(ctx context.Context, bucket, key, uploadID string, parts []CompletedPart) error {
	completed := make([]*s3.CompletedPart, 0, len(parts))
	for _, p := range parts {
		completed = append(completed, &s3.CompletedPart{
			ETag:       aws.String(p.ETag),
			PartNumber: aws.Int64(int64(p.PartNumber)),
		})
	}
	_, err := c.s3Client.CompleteMultipartUploadWithContext(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completed,
		},
	})
	if err != nil {
		return fmt.Errorf("complete multipart upload: %w", err)
	}
	return nil
}

// AbortMultipartUpload cancels an in-progress multipart upload. Safe to call
// on a non-existent upload (S3 returns a no-op error we ignore).
func (c *Client) AbortMultipartUpload(ctx context.Context, bucket, key, uploadID string) error {
	_, err := c.s3Client.AbortMultipartUploadWithContext(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		return fmt.Errorf("abort multipart upload: %w", err)
	}
	return nil
}

// PresignUploadPart returns a presigned URL the browser can PUT a single
// chunk to. Lets the multipart upload bypass the Go API for the actual byte
// transfer. The upload id and part number are baked into the URL.
func (c *Client) PresignUploadPart(ctx context.Context, bucket, key, uploadID string, partNumber int, ttl time.Duration) (string, error) {
	req, _ := c.s3Client.UploadPartRequest(&s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int64(int64(partNumber)),
	})
	return req.Presign(ttl)
}

// PresignGetURLRange returns a presigned URL that fetches only a byte range.
// Used by the transcoder to verify chunk integrity and by debug tools.
func (c *Client) PresignGetURLRange(ctx context.Context, bucket, key string, start, end int64, ttl time.Duration) (string, error) {
	req, _ := c.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  aws.String(fmt.Sprintf("bytes=%d-%d", start, end)),
	})
	return req.Presign(ttl)
}
