package contracts

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/marcusPrado02/go-commons/ports/files"
	"github.com/stretchr/testify/suite"
)

// FileStoreContract is a reusable test suite for FileStorePort implementations.
// Provide a Bucket name and the implementation under test. Each test case uses
// a unique key derived from the test name to avoid collisions.
//
// Example:
//
//	func TestS3Client(t *testing.T) {
//	    suite.Run(t, &contracts.FileStoreContract{
//	        Store:  s3.New(cfg),
//	        Bucket: "my-test-bucket",
//	    })
//	}
type FileStoreContract struct {
	suite.Suite
	// Store is the FileStorePort implementation under test.
	Store files.FileStorePort
	// Bucket is the bucket/container used for all test operations.
	Bucket string
}

func (s *FileStoreContract) key(suffix string) files.FileID {
	return files.FileID{
		Bucket: s.Bucket,
		Key:    fmt.Sprintf("contract-test/%s-%d", suffix, time.Now().UnixNano()),
	}
}

func (s *FileStoreContract) TestUpload_Download_RoundTrip() {
	ctx := context.Background()
	id := s.key("upload-download")
	content := "hello from the contract suite"

	_, err := s.Store.Upload(ctx, id, strings.NewReader(content))
	s.Require().NoError(err)

	obj, err := s.Store.Download(ctx, id)
	s.Require().NoError(err)
	defer obj.Content.Close()

	got, err := io.ReadAll(obj.Content)
	s.Require().NoError(err)
	s.Equal(content, string(got))
}

func (s *FileStoreContract) TestUpload_ContentType_StoredInMetadata() {
	ctx := context.Background()
	id := s.key("content-type")

	_, err := s.Store.Upload(ctx, id, strings.NewReader("data"),
		files.WithContentType("text/plain; charset=utf-8"),
	)
	s.Require().NoError(err)

	meta, err := s.Store.GetMetadata(ctx, id)
	s.Require().NoError(err)
	s.Contains(meta.ContentType, "text/plain")
}

func (s *FileStoreContract) TestExists_TrueAfterUpload() {
	ctx := context.Background()
	id := s.key("exists")

	_, err := s.Store.Upload(ctx, id, bytes.NewReader([]byte("x")))
	s.Require().NoError(err)

	ok, err := s.Store.Exists(ctx, id)
	s.Require().NoError(err)
	s.True(ok)
}

func (s *FileStoreContract) TestExists_FalseForMissingKey() {
	ctx := context.Background()
	id := files.FileID{Bucket: s.Bucket, Key: "contract-test/does-not-exist-xyz"}

	ok, err := s.Store.Exists(ctx, id)
	s.Require().NoError(err)
	s.False(ok)
}

func (s *FileStoreContract) TestDelete_RemovesFile() {
	ctx := context.Background()
	id := s.key("delete")

	_, err := s.Store.Upload(ctx, id, strings.NewReader("to be deleted"))
	s.Require().NoError(err)

	s.Require().NoError(s.Store.Delete(ctx, id))

	ok, err := s.Store.Exists(ctx, id)
	s.Require().NoError(err)
	s.False(ok)
}

func (s *FileStoreContract) TestList_ReturnsUploadedObjects() {
	ctx := context.Background()
	prefix := fmt.Sprintf("contract-test/list-%d/", time.Now().UnixNano())

	for i := 0; i < 3; i++ {
		id := files.FileID{Bucket: s.Bucket, Key: fmt.Sprintf("%sfile-%d", prefix, i)}
		_, err := s.Store.Upload(ctx, id, strings.NewReader("data"))
		s.Require().NoError(err)
	}

	result, err := s.Store.List(ctx, s.Bucket, prefix)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(result.Objects), 3, "expected at least 3 objects under prefix")
}

func (s *FileStoreContract) TestDownload_ClosingBodyTwiceIsSafe() {
	ctx := context.Background()
	id := s.key("double-close")

	_, err := s.Store.Upload(ctx, id, strings.NewReader("body"))
	s.Require().NoError(err)

	obj, err := s.Store.Download(ctx, id)
	s.Require().NoError(err)

	// First close
	s.Require().NoError(obj.Content.Close())
	// Second close must not panic (io.NopCloser etc. are safe; S3 may return EOF error — that's OK).
	_ = obj.Content.Close()
}
