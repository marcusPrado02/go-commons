package files_test

import (
	"testing"

	"github.com/marcusPrado02/go-commons/ports/files"
	"github.com/stretchr/testify/assert"
)

func TestWithContentType(t *testing.T) {
	opts := &files.UploadOptions{}
	files.WithContentType("image/png")(opts)
	assert.Equal(t, "image/png", opts.ContentType)
}

func TestWithStorageClass(t *testing.T) {
	opts := &files.UploadOptions{}
	files.WithStorageClass(files.StorageClassGlacier)(opts)
	assert.Equal(t, files.StorageClassGlacier, opts.StorageClass)
}

func TestWithMetadata(t *testing.T) {
	opts := &files.UploadOptions{}
	files.WithMetadata(map[string]string{"owner": "alice"})(opts)
	assert.Equal(t, "alice", opts.Metadata["owner"])
}

func TestWithMaxKeys(t *testing.T) {
	opts := &files.ListOptions{}
	files.WithMaxKeys(50)(opts)
	assert.Equal(t, 50, opts.MaxKeys)
}

func TestWithContentDisposition(t *testing.T) {
	opts := &files.PresignOptions{}
	files.WithContentDisposition("attachment")(opts)
	assert.Equal(t, "attachment", opts.ResponseContentDisposition)
}
