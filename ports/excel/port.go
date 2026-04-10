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
	// Name is the tab label for the sheet. Must be unique within the workbook.
	Name string
	// Headers is the first row of the sheet, used as column labels.
	Headers []string
	// Rows contains the data rows. Each element must be JSON-serializable or a primitive type.
	Rows [][]any
}

// ExcelRequest describes the workbook to generate.
type ExcelRequest struct {
	// Filename is the suggested file name for the generated workbook (e.g. "report.xlsx").
	Filename string
	// Sheets is the ordered list of worksheets to include in the workbook.
	Sheets []Sheet
}
