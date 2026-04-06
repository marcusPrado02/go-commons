// Package excel defines the port interface for Excel spreadsheet generation.
package excel

import (
	"context"
	"io"
)

// ExcelPort generates Excel (.xlsx) files from structured data.
type ExcelPort interface {
	// Generate produces an Excel file from the given request.
	// The returned io.Reader contains the .xlsx content.
	Generate(ctx context.Context, req ExcelRequest) (io.Reader, error)
}

// Sheet defines a single worksheet within an Excel workbook.
type Sheet struct {
	Name    string
	Headers []string
	Rows    [][]any
}

// ExcelRequest describes the workbook to generate.
type ExcelRequest struct {
	Filename string
	Sheets   []Sheet
}
