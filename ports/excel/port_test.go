package excel_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/marcusPrado02/go-commons/ports/excel"
)

// Compile-time check that Port can be implemented.
var _ excel.Port = (*nilExcel)(nil)

type nilExcel struct{}

func (n *nilExcel) Generate(_ context.Context, _ excel.Request) (io.Reader, error) {
	return strings.NewReader(""), nil
}

func TestSheet_Fields(t *testing.T) {
	s := excel.Sheet{
		Name:    "Sales",
		Headers: []string{"Date", "Amount"},
		Rows:    [][]any{{"2026-01-01", 100}},
	}
	if s.Name != "Sales" {
		t.Errorf("unexpected Name: %q", s.Name)
	}
	if len(s.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(s.Headers))
	}
	if len(s.Rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(s.Rows))
	}
}

func TestExcelRequest_Fields(t *testing.T) {
	req := excel.Request{
		Filename: "report.xlsx",
		Sheets:   []excel.Sheet{{Name: "Sheet1"}},
	}
	if req.Filename != "report.xlsx" {
		t.Errorf("unexpected Filename: %q", req.Filename)
	}
	if len(req.Sheets) != 1 {
		t.Errorf("expected 1 sheet, got %d", len(req.Sheets))
	}
}

func TestSheet_ZeroValue(t *testing.T) {
	var s excel.Sheet
	if s.Name != "" || s.Headers != nil || s.Rows != nil {
		t.Fatal("expected zero-value Sheet fields")
	}
}

func TestExcelRequest_ZeroValue(t *testing.T) {
	var req excel.Request
	if req.Filename != "" || req.Sheets != nil {
		t.Fatal("expected zero-value Request fields")
	}
}
