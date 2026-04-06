package template_test

import (
	"testing"

	"github.com/marcusPrado02/go-commons/ports/template"
	"github.com/stretchr/testify/assert"
)

func TestHTMLResult(t *testing.T) {
	r := template.HTMLResult("welcome", "<h1>Hi</h1>")
	assert.Equal(t, "welcome", r.TemplateName)
	assert.Equal(t, "<h1>Hi</h1>", r.Content)
	assert.Equal(t, template.ContentTypeHTML, r.ContentType)
	assert.Equal(t, "UTF-8", r.Charset)
}

func TestTextResult(t *testing.T) {
	r := template.TextResult("plain", "hello")
	assert.Equal(t, template.ContentTypeText, r.ContentType)
}

func TestXMLResult(t *testing.T) {
	r := template.XMLResult("feed", "<feed/>")
	assert.Equal(t, template.ContentTypeXML, r.ContentType)
}

func TestTemplateResult_Bytes(t *testing.T) {
	r := template.HTMLResult("t", "abc")
	assert.Equal(t, []byte("abc"), r.Bytes())
}

func TestTemplateResult_IsEmpty(t *testing.T) {
	empty := template.HTMLResult("t", "")
	assert.True(t, empty.IsEmpty())
	nonempty := template.HTMLResult("t", "x")
	assert.False(t, nonempty.IsEmpty())
}
