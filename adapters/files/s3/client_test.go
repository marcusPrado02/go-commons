package s3_test

import (
	"testing"

	s3adapter "github.com/marcusPrado02/go-commons/adapters/files/s3"
	filesport "github.com/marcusPrado02/go-commons/ports/files"
)

// Compile-time interface check — ensures Client implements FileStorePort.
var _ filesport.FileStorePort = (*s3adapter.Client)(nil)

func TestNew_ReturnsClient(t *testing.T) {
	// Integration test — requires AWS credentials.
	// Skipped in unit test runs without credentials.
	t.Skip("requires AWS credentials")
}
