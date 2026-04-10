// Package s3 provides an AWS S3 implementation of ports/files.FileStorePort.
package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	filesport "github.com/marcusPrado02/go-commons/ports/files"
)

// Client is an AWS S3 implementation of FileStorePort.
type Client struct {
	s3       *awss3.Client
	signer   *awss3.PresignClient
	uploader *manager.Uploader
	opts     clientOptions
}

type clientOptions struct {
	defaultStorageClass filesport.StorageClass
	sseAlgorithm        string
}

// Option configures an S3 Client.
type Option func(*clientOptions)

// WithDefaultStorageClass sets the default storage class for uploads.
func WithDefaultStorageClass(sc filesport.StorageClass) Option {
	return func(o *clientOptions) { o.defaultStorageClass = sc }
}

// WithSSEAlgorithm enables server-side encryption with the given algorithm (e.g. "AES256").
func WithSSEAlgorithm(alg string) Option {
	return func(o *clientOptions) { o.sseAlgorithm = alg }
}

// New creates an S3 Client from an aws.Config.
func New(cfg aws.Config, opts ...Option) *Client {
	return NewWithOptions(cfg, nil, opts...)
}

// NewWithOptions creates an S3 Client with additional AWS S3 options (e.g. a custom endpoint for LocalStack).
// awsOpts are applied to the underlying *awss3.Client. Pass nil if not needed.
func NewWithOptions(cfg aws.Config, awsOpts func(*awss3.Options), opts ...Option) *Client {
	svc := awss3.NewFromConfig(cfg, func(o *awss3.Options) {
		if awsOpts != nil {
			awsOpts(o)
		}
	})
	o := clientOptions{defaultStorageClass: filesport.StorageClassStandard}
	for _, opt := range opts {
		opt(&o)
	}
	return &Client{
		s3:       svc,
		signer:   awss3.NewPresignClient(svc),
		uploader: manager.NewUploader(svc),
		opts:     o,
	}
}

// Upload stores content under the given FileID.
func (c *Client) Upload(ctx context.Context, id filesport.FileID, content io.Reader, optFns ...filesport.UploadOption) (filesport.UploadResult, error) {
	opts := filesport.UploadOptions{StorageClass: c.opts.defaultStorageClass}
	for _, fn := range optFns {
		fn(&opts)
	}

	input := &awss3.PutObjectInput{
		Bucket:       aws.String(id.Bucket),
		Key:          aws.String(id.Key),
		Body:         content,
		StorageClass: types.StorageClass(string(opts.StorageClass)),
	}
	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}
	if c.opts.sseAlgorithm != "" {
		input.ServerSideEncryption = types.ServerSideEncryption(c.opts.sseAlgorithm)
	}

	result, err := c.uploader.Upload(ctx, input)
	if err != nil {
		return filesport.UploadResult{}, fmt.Errorf("s3: upload failed: %w", err)
	}
	return filesport.UploadResult{Location: result.Location}, nil
}

// Download retrieves file content and metadata. Caller must close FileObject.Content.
func (c *Client) Download(ctx context.Context, id filesport.FileID) (filesport.FileObject, error) {
	out, err := c.s3.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(id.Bucket),
		Key:    aws.String(id.Key),
	})
	if err != nil {
		return filesport.FileObject{}, fmt.Errorf("s3: download failed: %w", err)
	}
	meta := filesport.FileMetadata{ETag: aws.ToString(out.ETag)}
	if out.ContentType != nil {
		meta.ContentType = *out.ContentType
	}
	if out.ContentLength != nil {
		meta.ContentLength = *out.ContentLength
	}
	return filesport.FileObject{Content: out.Body, Metadata: meta}, nil
}

// Delete removes a single file.
func (c *Client) Delete(ctx context.Context, id filesport.FileID) error {
	_, err := c.s3.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(id.Bucket),
		Key:    aws.String(id.Key),
	})
	if err != nil {
		return fmt.Errorf("s3: delete failed: %w", err)
	}
	return nil
}

// DeleteAll removes multiple files. Reports per-file errors in DeleteResult.Failed.
func (c *Client) DeleteAll(ctx context.Context, ids []filesport.FileID) (filesport.DeleteResult, error) {
	var result filesport.DeleteResult
	for _, id := range ids {
		if err := c.Delete(ctx, id); err != nil {
			result.Failed = append(result.Failed, filesport.DeleteError{ID: id, Cause: err})
		} else {
			result.Deleted = append(result.Deleted, id)
		}
	}
	return result, nil
}

// Exists returns true if the object exists.
func (c *Client) Exists(ctx context.Context, id filesport.FileID) (bool, error) {
	_, err := c.s3.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(id.Bucket),
		Key:    aws.String(id.Key),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("s3: exists check failed: %w", err)
	}
	return true, nil
}

// GetMetadata returns file metadata without downloading content.
func (c *Client) GetMetadata(ctx context.Context, id filesport.FileID) (filesport.FileMetadata, error) {
	out, err := c.s3.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(id.Bucket),
		Key:    aws.String(id.Key),
	})
	if err != nil {
		return filesport.FileMetadata{}, fmt.Errorf("s3: get metadata failed: %w", err)
	}
	meta := filesport.FileMetadata{ETag: aws.ToString(out.ETag)}
	if out.ContentType != nil {
		meta.ContentType = *out.ContentType
	}
	if out.ContentLength != nil {
		meta.ContentLength = *out.ContentLength
	}
	return meta, nil
}

// List returns objects in a bucket matching the prefix.
func (c *Client) List(ctx context.Context, bucket, prefix string, optFns ...filesport.ListOption) (filesport.ListResult, error) {
	opts := filesport.ListOptions{MaxKeys: 1000}
	for _, fn := range optFns {
		fn(&opts)
	}

	input := &awss3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}
	if opts.MaxKeys > 0 {
		mk := int32(opts.MaxKeys)
		input.MaxKeys = &mk
	}
	if opts.ContinuationToken != "" {
		input.ContinuationToken = aws.String(opts.ContinuationToken)
	}

	out, err := c.s3.ListObjectsV2(ctx, input)
	if err != nil {
		return filesport.ListResult{}, fmt.Errorf("s3: list failed: %w", err)
	}

	var objects []filesport.FileMetadata
	for _, obj := range out.Contents {
		objects = append(objects, filesport.FileMetadata{
			ETag:          aws.ToString(obj.ETag),
			ContentLength: aws.ToInt64(obj.Size),
		})
	}

	var nextToken string
	if out.NextContinuationToken != nil {
		nextToken = *out.NextContinuationToken
	}

	return filesport.ListResult{
		Objects:           objects,
		ContinuationToken: nextToken,
		IsTruncated:       aws.ToBool(out.IsTruncated),
	}, nil
}

// GeneratePresignedURL creates a time-limited URL for direct client access.
func (c *Client) GeneratePresignedURL(ctx context.Context, id filesport.FileID, op filesport.PresignedOperation, ttl time.Duration, optFns ...filesport.PresignOption) (*url.URL, error) {
	var rawURL string

	switch op {
	case filesport.PresignGet:
		req, err := c.signer.PresignGetObject(ctx, &awss3.GetObjectInput{
			Bucket: aws.String(id.Bucket),
			Key:    aws.String(id.Key),
		}, awss3.WithPresignExpires(ttl))
		if err != nil {
			return nil, err
		}
		rawURL = req.URL
	case filesport.PresignPut:
		req, err := c.signer.PresignPutObject(ctx, &awss3.PutObjectInput{
			Bucket: aws.String(id.Bucket),
			Key:    aws.String(id.Key),
		}, awss3.WithPresignExpires(ttl))
		if err != nil {
			return nil, err
		}
		rawURL = req.URL
	case filesport.PresignDelete:
		req, err := c.signer.PresignDeleteObject(ctx, &awss3.DeleteObjectInput{
			Bucket: aws.String(id.Bucket),
			Key:    aws.String(id.Key),
		}, awss3.WithPresignExpires(ttl))
		if err != nil {
			return nil, err
		}
		rawURL = req.URL
	default:
		return nil, fmt.Errorf("s3: unsupported presigned operation: %s", op)
	}

	return url.Parse(rawURL)
}

// Copy duplicates a file within S3.
func (c *Client) Copy(ctx context.Context, src, dst filesport.FileID) error {
	copySource := src.Bucket + "/" + src.Key
	_, err := c.s3.CopyObject(ctx, &awss3.CopyObjectInput{
		Bucket:     aws.String(dst.Bucket),
		Key:        aws.String(dst.Key),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		return fmt.Errorf("s3: copy failed: %w", err)
	}
	return nil
}

var _ filesport.FileStorePort = (*Client)(nil)
