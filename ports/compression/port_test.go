package compression_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/marcusPrado02/go-commons/ports/compression"
)

// Compile-time check that CompressionPort can be implemented.
var _ compression.CompressionPort = (*nilCompression)(nil)

type nilCompression struct{}

func (n *nilCompression) Compress(_ context.Context, src io.Reader, _ compression.Format) (io.Reader, error) {
	return src, nil
}
func (n *nilCompression) Decompress(_ context.Context, src io.Reader, _ compression.Format) (io.Reader, error) {
	return src, nil
}

func TestFormat_Constants(t *testing.T) {
	cases := []struct {
		name     string
		format   compression.Format
		expected string
	}{
		{"gzip", compression.FormatGzip, "gzip"},
		{"zstd", compression.FormatZstd, "zstd"},
		{"snappy", compression.FormatSnappy, "snappy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.format) != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.format)
			}
		})
	}
}

func TestCompressionPort_InterfaceSignature(t *testing.T) {
	var cp compression.CompressionPort = &nilCompression{}
	ctx := context.Background()
	src := strings.NewReader("hello")

	r, err := cp.Compress(ctx, src, compression.FormatGzip)
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}
	if _, err := cp.Decompress(ctx, r, compression.FormatGzip); err != nil {
		t.Fatalf("Decompress: %v", err)
	}
}
