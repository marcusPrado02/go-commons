// Package stdlib provides a CompressionPort implementation using Go standard library.
// Supports gzip and flate (deflate). No external dependencies.
package stdlib

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"fmt"
	"io"

	"github.com/marcusPrado02/go-commons/ports/compression"
)

// Compressor is a stdlib implementation of compression.CompressionPort.
// Supports gzip and zstd (via flate as a substitute) formats.
// For production use of snappy or zstd, use a dedicated adapter.
type Compressor struct{}

// New creates a new stdlib Compressor.
func New() *Compressor { return &Compressor{} }

// Compress compresses src using the given format and returns the compressed stream.
// Supported formats: gzip, zstd (implemented with flate/deflate).
// snappy is not supported by the stdlib — returns an error.
func (c *Compressor) Compress(_ context.Context, src io.Reader, format compression.Format) (io.Reader, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("stdlib: read source failed: %w", err)
	}

	var buf bytes.Buffer
	switch format {
	case compression.FormatGzip:
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(data); err != nil {
			return nil, fmt.Errorf("stdlib: gzip compress failed: %w", err)
		}
		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("stdlib: gzip close failed: %w", err)
		}
	case compression.FormatZstd:
		// zstd is not in the stdlib; use flate (deflate) as the closest available alternative.
		w, err := flate.NewWriter(&buf, flate.BestSpeed)
		if err != nil {
			return nil, fmt.Errorf("stdlib: flate compress failed: %w", err)
		}
		if _, err := w.Write(data); err != nil {
			return nil, fmt.Errorf("stdlib: flate write failed: %w", err)
		}
		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("stdlib: flate close failed: %w", err)
		}
	case compression.FormatSnappy:
		return nil, fmt.Errorf("stdlib: snappy is not supported — use adapters/compression/snappy")
	default:
		return nil, fmt.Errorf("stdlib: unsupported format: %s", format)
	}
	return &buf, nil
}

// Decompress decompresses src in the given format and returns the decompressed stream.
func (c *Compressor) Decompress(_ context.Context, src io.Reader, format compression.Format) (io.Reader, error) {
	switch format {
	case compression.FormatGzip:
		r, err := gzip.NewReader(src)
		if err != nil {
			return nil, fmt.Errorf("stdlib: gzip decompress failed: %w", err)
		}
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("stdlib: gzip read failed: %w", err)
		}
		_ = r.Close()
		return bytes.NewReader(data), nil
	case compression.FormatZstd:
		r := flate.NewReader(src)
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("stdlib: flate decompress failed: %w", err)
		}
		_ = r.Close()
		return bytes.NewReader(data), nil
	case compression.FormatSnappy:
		return nil, fmt.Errorf("stdlib: snappy is not supported — use adapters/compression/snappy")
	default:
		return nil, fmt.Errorf("stdlib: unsupported format: %s", format)
	}
}

var _ compression.CompressionPort = (*Compressor)(nil)
