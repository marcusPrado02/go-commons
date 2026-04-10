package s3_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	s3adapter "github.com/marcusPrado02/go-commons/adapters/files/s3"
	filesport "github.com/marcusPrado02/go-commons/ports/files"
)

// compile-time interface check
var _ filesport.FileStorePort = (*s3adapter.Client)(nil)

const (
	localstackDefaultURL = "http://localhost:4566"
	testBucket           = "go-commons-test"
	testRegion           = "us-east-1"
)

// localstackURL returns the LocalStack endpoint from the environment, with a fallback.
func localstackURL() string {
	if u := os.Getenv("LOCALSTACK_URL"); u != "" {
		return u
	}
	return localstackDefaultURL
}

// skipIfNoLocalstack pings the LocalStack endpoint and skips if unavailable.
func skipIfNoLocalstack(t *testing.T) {
	t.Helper()
	hc := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := hc.Get(localstackURL() + "/_localstack/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Skipf("LocalStack not available at %s — skipping S3 integration test", localstackURL())
	}
	resp.Body.Close()
}

// newLocalstackClient builds an S3 client pointed at LocalStack.
func newLocalstackClient(t *testing.T) *s3adapter.Client {
	t.Helper()
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(testRegion),
		awsconfig.WithCredentialsProvider(awscreds.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	endpoint := localstackURL()
	return s3adapter.NewWithOptions(cfg, func(o *awss3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true // LocalStack requires path-style addressing
	})
}

// ensureBucket creates the test bucket if it does not exist.
func ensureBucket(t *testing.T) {
	t.Helper()
	cfg, _ := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(testRegion),
		awsconfig.WithCredentialsProvider(awscreds.NewStaticCredentialsProvider("test", "test", "")),
	)
	endpoint := localstackURL()
	rawClient := awss3.NewFromConfig(cfg, func(o *awss3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	_, _ = rawClient.CreateBucket(context.Background(), &awss3.CreateBucketInput{
		Bucket: aws.String(testBucket),
	})
}

func TestS3_Upload_Download(t *testing.T) {
	skipIfNoLocalstack(t)
	client := newLocalstackClient(t)
	ensureBucket(t)

	ctx := context.Background()
	id := filesport.FileID{Bucket: testBucket, Key: "test/upload-download.txt"}
	content := "hello localstack"

	_, err := client.Upload(ctx, id, strings.NewReader(content),
		filesport.WithContentType("text/plain"),
	)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	obj, err := client.Download(ctx, id)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer obj.Content.Close()

	got, err := io.ReadAll(obj.Content)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != content {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestS3_Exists(t *testing.T) {
	skipIfNoLocalstack(t)
	client := newLocalstackClient(t)
	ensureBucket(t)

	ctx := context.Background()
	id := filesport.FileID{Bucket: testBucket, Key: "test/exists.txt"}

	exists, err := client.Exists(ctx, id)
	if err != nil {
		t.Fatalf("Exists (before upload): %v", err)
	}
	if exists {
		t.Error("expected file to not exist before upload")
	}

	if _, err := client.Upload(ctx, id, strings.NewReader("x")); err != nil {
		t.Fatalf("Upload: %v", err)
	}

	exists, err = client.Exists(ctx, id)
	if err != nil {
		t.Fatalf("Exists (after upload): %v", err)
	}
	if !exists {
		t.Error("expected file to exist after upload")
	}
}

func TestS3_Delete(t *testing.T) {
	skipIfNoLocalstack(t)
	client := newLocalstackClient(t)
	ensureBucket(t)

	ctx := context.Background()
	id := filesport.FileID{Bucket: testBucket, Key: "test/delete-me.txt"}

	if _, err := client.Upload(ctx, id, strings.NewReader("bye")); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if err := client.Delete(ctx, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	exists, err := client.Exists(ctx, id)
	if err != nil {
		t.Fatalf("Exists after delete: %v", err)
	}
	if exists {
		t.Error("expected file to not exist after delete")
	}
}

func TestS3_List(t *testing.T) {
	skipIfNoLocalstack(t)
	client := newLocalstackClient(t)
	ensureBucket(t)

	ctx := context.Background()
	prefix := "test/list/"

	keys := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, k := range keys {
		id := filesport.FileID{Bucket: testBucket, Key: prefix + k}
		if _, err := client.Upload(ctx, id, strings.NewReader("data")); err != nil {
			t.Fatalf("Upload %s: %v", k, err)
		}
	}

	result, err := client.List(ctx, testBucket, prefix)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Objects) < len(keys) {
		t.Errorf("expected at least %d objects, got %d", len(keys), len(result.Objects))
	}
}

func TestS3_Copy(t *testing.T) {
	skipIfNoLocalstack(t)
	client := newLocalstackClient(t)
	ensureBucket(t)

	ctx := context.Background()
	src := filesport.FileID{Bucket: testBucket, Key: "test/copy-src.txt"}
	dst := filesport.FileID{Bucket: testBucket, Key: "test/copy-dst.txt"}
	content := "copy me"

	if _, err := client.Upload(ctx, src, strings.NewReader(content)); err != nil {
		t.Fatalf("Upload src: %v", err)
	}
	if err := client.Copy(ctx, src, dst); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	obj, err := client.Download(ctx, dst)
	if err != nil {
		t.Fatalf("Download dst: %v", err)
	}
	defer obj.Content.Close()
	got, _ := io.ReadAll(obj.Content)
	if string(got) != content {
		t.Errorf("copied content: got %q, want %q", got, content)
	}
}

func TestS3_GetMetadata(t *testing.T) {
	skipIfNoLocalstack(t)
	client := newLocalstackClient(t)
	ensureBucket(t)

	ctx := context.Background()
	id := filesport.FileID{Bucket: testBucket, Key: "test/metadata.txt"}
	body := []byte("metadata test")

	if _, err := client.Upload(ctx, id, bytes.NewReader(body),
		filesport.WithContentType("text/plain"),
	); err != nil {
		t.Fatalf("Upload: %v", err)
	}

	meta, err := client.GetMetadata(ctx, id)
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if meta.ContentLength != int64(len(body)) {
		t.Errorf("ContentLength: got %d, want %d", meta.ContentLength, len(body))
	}
}

func TestS3_GeneratePresignedURL(t *testing.T) {
	skipIfNoLocalstack(t)
	client := newLocalstackClient(t)
	ensureBucket(t)

	ctx := context.Background()
	id := filesport.FileID{Bucket: testBucket, Key: "test/presign.txt"}
	if _, err := client.Upload(ctx, id, strings.NewReader("presign data")); err != nil {
		t.Fatalf("Upload: %v", err)
	}

	u, err := client.GeneratePresignedURL(ctx, id, filesport.PresignGet, time.Hour)
	if err != nil {
		t.Fatalf("GeneratePresignedURL: %v", err)
	}
	if u == nil || u.String() == "" {
		t.Fatal("expected non-empty presigned URL")
	}
}
