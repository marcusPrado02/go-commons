// Package gcs provides a Google Cloud Storage implementation of ports/files.FileStorePort.
package gcs

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"cloud.google.com/go/storage"
	filesport "github.com/marcusPrado02/go-commons/ports/files"
	"google.golang.org/api/iterator"
)

// Client is a GCS implementation of FileStorePort.
type Client struct {
	gcs *storage.Client
}

// New creates a new GCS client. The provided context is used only for client creation.
func New(ctx context.Context) (*Client, error) {
	c, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs: failed to create client: %w", err)
	}
	return &Client{gcs: c}, nil
}

// Upload writes content to GCS.
func (c *Client) Upload(ctx context.Context, id filesport.FileID, content io.Reader, optFns ...filesport.UploadOption) (filesport.UploadResult, error) {
	opts := filesport.UploadOptions{}
	for _, fn := range optFns {
		fn(&opts)
	}
	wc := c.gcs.Bucket(id.Bucket).Object(id.Key).NewWriter(ctx)
	if opts.ContentType != "" {
		wc.ContentType = opts.ContentType
	}
	if _, err := io.Copy(wc, content); err != nil {
		_ = wc.Close()
		return filesport.UploadResult{}, fmt.Errorf("gcs: upload failed: %w", err)
	}
	if err := wc.Close(); err != nil {
		return filesport.UploadResult{}, fmt.Errorf("gcs: upload close failed: %w", err)
	}
	return filesport.UploadResult{Location: fmt.Sprintf("gs://%s/%s", id.Bucket, id.Key)}, nil
}

// Download retrieves file content. Caller must close FileObject.Content.
func (c *Client) Download(ctx context.Context, id filesport.FileID) (filesport.FileObject, error) {
	rc, err := c.gcs.Bucket(id.Bucket).Object(id.Key).NewReader(ctx)
	if err != nil {
		return filesport.FileObject{}, fmt.Errorf("gcs: download failed: %w", err)
	}
	return filesport.FileObject{Content: rc}, nil
}

// Delete removes a file from GCS.
func (c *Client) Delete(ctx context.Context, id filesport.FileID) error {
	if err := c.gcs.Bucket(id.Bucket).Object(id.Key).Delete(ctx); err != nil {
		return fmt.Errorf("gcs: delete failed: %w", err)
	}
	return nil
}

// DeleteAll removes multiple files.
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
	_, err := c.gcs.Bucket(id.Bucket).Object(id.Key).Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("gcs: exists check failed: %w", err)
	}
	return true, nil
}

// GetMetadata returns file attributes.
func (c *Client) GetMetadata(ctx context.Context, id filesport.FileID) (filesport.FileMetadata, error) {
	attrs, err := c.gcs.Bucket(id.Bucket).Object(id.Key).Attrs(ctx)
	if err != nil {
		return filesport.FileMetadata{}, fmt.Errorf("gcs: get metadata failed: %w", err)
	}
	return filesport.FileMetadata{
		ContentType:   attrs.ContentType,
		ContentLength: attrs.Size,
		ETag:          attrs.Etag,
		LastModified:  attrs.Updated,
	}, nil
}

// List returns objects in a bucket matching the prefix.
func (c *Client) List(ctx context.Context, bucket, prefix string, optFns ...filesport.ListOption) (filesport.ListResult, error) {
	opts := filesport.ListOptions{MaxKeys: 1000}
	for _, fn := range optFns {
		fn(&opts)
	}

	query := &storage.Query{Prefix: prefix}
	it := c.gcs.Bucket(bucket).Objects(ctx, query)

	var objects []filesport.FileMetadata
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return filesport.ListResult{}, fmt.Errorf("gcs: list failed: %w", err)
		}
		objects = append(objects, filesport.FileMetadata{
			ContentType:   attrs.ContentType,
			ContentLength: attrs.Size,
			ETag:          attrs.Etag,
		})
		if opts.MaxKeys > 0 && len(objects) >= opts.MaxKeys {
			break
		}
	}
	return filesport.ListResult{Objects: objects}, nil
}

// GeneratePresignedURL creates a signed URL for direct GCS access.
func (c *Client) GeneratePresignedURL(ctx context.Context, id filesport.FileID, op filesport.PresignedOperation, ttl time.Duration, _ ...filesport.PresignOption) (*url.URL, error) {
	method := string(op)
	signed, err := c.gcs.Bucket(id.Bucket).SignedURL(id.Key, &storage.SignedURLOptions{
		Method:  method,
		Expires: time.Now().Add(ttl),
	})
	if err != nil {
		return nil, fmt.Errorf("gcs: presign failed: %w", err)
	}
	return url.Parse(signed)
}

// Copy duplicates a GCS object.
func (c *Client) Copy(ctx context.Context, src, dst filesport.FileID) error {
	srcObj := c.gcs.Bucket(src.Bucket).Object(src.Key)
	dstObj := c.gcs.Bucket(dst.Bucket).Object(dst.Key)
	if _, err := dstObj.CopierFrom(srcObj).Run(ctx); err != nil {
		return fmt.Errorf("gcs: copy failed: %w", err)
	}
	return nil
}

var _ filesport.FileStorePort = (*Client)(nil)
