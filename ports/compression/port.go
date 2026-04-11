// Package compression defines the port interface for data compression.
package compression

import (
	"context"
	"io"
)

// Format identifies the compression algorithm.
type Format string

const (
	// FormatGzip uses the gzip compression format (RFC 1952). Supported by all adapters.
	FormatGzip Format = "gzip"
	// FormatZstd uses the Zstandard compression format. Requires an adapter with zstd support.
	FormatZstd Format = "zstd"
	// FormatSnappy uses Google's Snappy compression format. Requires an adapter with snappy support.
	FormatSnappy Format = "snappy"
)

// Port compresses and decompresses data streams.
type Port interface {
	// Compress reads from src and returns a compressed stream in the given format.
	Compress(ctx context.Context, src io.Reader, format Format) (io.Reader, error)
	// Decompress reads a compressed stream and returns the decompressed data.
	Decompress(ctx context.Context, src io.Reader, format Format) (io.Reader, error)
}
