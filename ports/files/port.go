// Package files defines the port interface for object/file storage.
package files

import (
	"context"
	"io"
	"net/url"
	"time"
)

// FileStorePort is the primary port for cloud object storage operations.
type FileStorePort interface {
	// Upload stores content under the given FileID.
	Upload(ctx context.Context, id FileID, content io.Reader, opts ...UploadOption) (UploadResult, error)
	// Download retrieves the content and metadata for a file.
	// The caller is responsible for closing FileObject.Content.
	Download(ctx context.Context, id FileID) (FileObject, error)
	// Delete removes a single file.
	Delete(ctx context.Context, id FileID) error
	// DeleteAll removes multiple files and reports which succeeded.
	DeleteAll(ctx context.Context, ids []FileID) (DeleteResult, error)
	// Exists returns true if the file exists, false if not found.
	Exists(ctx context.Context, id FileID) (bool, error)
	// GetMetadata returns file metadata without downloading content.
	GetMetadata(ctx context.Context, id FileID) (FileMetadata, error)
	// List returns files in a bucket under the given prefix.
	// prefix is path-like, e.g. "uploads/2026/" — no leading slash.
	List(ctx context.Context, bucket, prefix string, opts ...ListOption) (ListResult, error)
	// GeneratePresignedURL creates a time-limited URL for direct client access.
	GeneratePresignedURL(ctx context.Context, id FileID, op PresignedOperation, ttl time.Duration, opts ...PresignOption) (*url.URL, error)
	// Copy duplicates a file from src to dst within the same or different bucket.
	Copy(ctx context.Context, src, dst FileID) error
}

// FileID identifies a file by its bucket and key.
type FileID struct {
	Bucket string
	Key    string
}

// FileObject contains the content stream and metadata of a downloaded file.
// The caller must close Content after reading.
type FileObject struct {
	Content  io.ReadCloser
	Metadata FileMetadata
}

// FileMetadata holds descriptive information about a stored file.
type FileMetadata struct {
	ContentType   string
	ContentLength int64
	ETag          string
	LastModified  time.Time
	UserMetadata  map[string]string
}

// UploadResult is returned after a successful upload.
type UploadResult struct {
	ETag     string
	Location string
}

// DeleteResult reports the outcome of a bulk delete operation.
type DeleteResult struct {
	Deleted []FileID
	Failed  []DeleteError
}

// DeleteError pairs a FileID with the reason it could not be deleted.
type DeleteError struct {
	ID    FileID
	Cause error
}

// ListResult holds a page of listed files.
type ListResult struct {
	Objects           []FileMetadata
	ContinuationToken string
	IsTruncated       bool
}

// PresignedOperation is the HTTP method for a presigned URL.
type PresignedOperation string

const (
	PresignGet    PresignedOperation = "GET"
	PresignPut    PresignedOperation = "PUT"
	PresignDelete PresignedOperation = "DELETE"
)

// StorageClass controls durability/cost trade-offs in the storage backend.
type StorageClass string

const (
	StorageClassStandard StorageClass = "STANDARD"
	StorageClassGlacier  StorageClass = "GLACIER"
	StorageClassIA       StorageClass = "STANDARD_IA"
)

// UploadOption configures an upload operation.
type UploadOption func(*UploadOptions)

// UploadOptions holds resolved upload configuration.
type UploadOptions struct {
	ContentType  string
	StorageClass StorageClass
	Metadata     map[string]string
}

// WithContentType sets the MIME type for the uploaded file.
func WithContentType(ct string) UploadOption {
	return func(o *UploadOptions) { o.ContentType = ct }
}

// WithStorageClass sets the storage class for the uploaded file.
func WithStorageClass(sc StorageClass) UploadOption {
	return func(o *UploadOptions) { o.StorageClass = sc }
}

// WithMetadata attaches user-defined key-value metadata to the upload.
func WithMetadata(m map[string]string) UploadOption {
	return func(o *UploadOptions) { o.Metadata = m }
}

// ListOption configures a list operation.
type ListOption func(*ListOptions)

// ListOptions holds resolved list configuration.
type ListOptions struct {
	MaxKeys           int
	ContinuationToken string
}

// WithMaxKeys limits the number of objects returned in a list.
func WithMaxKeys(n int) ListOption {
	return func(o *ListOptions) { o.MaxKeys = n }
}

// PresignOption configures presigned URL generation.
type PresignOption func(*PresignOptions)

// PresignOptions holds resolved presign configuration.
type PresignOptions struct {
	ResponseContentDisposition string
}

// WithContentDisposition sets the Content-Disposition response header on the presigned URL.
func WithContentDisposition(cd string) PresignOption {
	return func(o *PresignOptions) { o.ResponseContentDisposition = cd }
}
