package stdlib_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/marcusPrado02/go-commons/adapters/compression/stdlib"
	"github.com/marcusPrado02/go-commons/ports/compression"
)

func TestCompressDecompress_Gzip_RoundTrip(t *testing.T) {
	c := stdlib.New()
	ctx := context.Background()
	original := "hello, world"

	compressed, err := c.Compress(ctx, strings.NewReader(original), compression.FormatGzip)
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}

	decompressed, err := c.Decompress(ctx, compressed, compression.FormatGzip)
	if err != nil {
		t.Fatalf("Decompress: %v", err)
	}

	got, _ := io.ReadAll(decompressed)
	if string(got) != original {
		t.Errorf("expected %q, got %q", original, got)
	}
}

func TestCompressDecompress_Zstd_RoundTrip(t *testing.T) {
	c := stdlib.New()
	ctx := context.Background()
	original := "zstd round-trip via flate"

	compressed, err := c.Compress(ctx, strings.NewReader(original), compression.FormatZstd)
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}

	decompressed, err := c.Decompress(ctx, compressed, compression.FormatZstd)
	if err != nil {
		t.Fatalf("Decompress: %v", err)
	}

	got, _ := io.ReadAll(decompressed)
	if string(got) != original {
		t.Errorf("expected %q, got %q", original, got)
	}
}

func TestCompress_Snappy_ReturnsError(t *testing.T) {
	c := stdlib.New()
	_, err := c.Compress(context.Background(), strings.NewReader("x"), compression.FormatSnappy)
	if err == nil {
		t.Fatal("expected error for snappy")
	}
}
