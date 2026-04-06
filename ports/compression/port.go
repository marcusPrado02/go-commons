// Package compression defines the port interface for data compression.
package compression

import (
	"context"
	"io"
)

// Format identifies the compression algorithm.
type Format string

const (
	FormatGzip   Format = "gzip"
	FormatZstd   Format = "zstd"
	FormatSnappy Format = "snappy"
)

// CompressionPort compresses and decompresses data streams.
type CompressionPort interface {
	// Compress reads from src and returns a compressed stream in the given format.
	Compress(ctx context.Context, src io.Reader, format Format) (io.Reader, error)
	// Decompress reads a compressed stream and returns the decompressed data.
	Decompress(ctx context.Context, src io.Reader, format Format) (io.Reader, error)
}
