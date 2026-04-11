// Package template defines the port interface for server-side template rendering.
package template

import "context"

// Port renders named templates with provided data.
type Port interface {
	// Render executes the named template with the given data map.
	Render(ctx context.Context, name string, data map[string]any) (Result, error)
	// Exists reports whether a template with the given name is registered.
	Exists(ctx context.Context, name string) (bool, error)
}

// Content type constants for use in Result.
const (
	ContentTypeHTML = "text/html"
	ContentTypeText = "text/plain"
	ContentTypeXML  = "application/xml"
)

// Result holds the output of a rendered template.
type Result struct {
	TemplateName string
	Content      string
	// ContentType should be one of the ContentType* constants.
	ContentType string
	Charset     string
}

// HTMLResult constructs a Result with HTML content type.
func HTMLResult(name, content string) Result {
	return Result{TemplateName: name, Content: content, ContentType: ContentTypeHTML, Charset: "UTF-8"}
}

// TextResult constructs a Result with plain-text content type.
func TextResult(name, content string) Result {
	return Result{TemplateName: name, Content: content, ContentType: ContentTypeText, Charset: "UTF-8"}
}

// XMLResult constructs a Result with XML content type.
func XMLResult(name, content string) Result {
	return Result{TemplateName: name, Content: content, ContentType: ContentTypeXML, Charset: "UTF-8"}
}

// Bytes returns the Content as a UTF-8 byte slice.
func (t Result) Bytes() []byte { return []byte(t.Content) }

// IsEmpty returns true if the Content is empty.
func (t Result) IsEmpty() bool { return t.Content == "" }
