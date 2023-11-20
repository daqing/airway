package utils

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
)

func RenderMarkdown(content string) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(content), &buf); err != nil {
		return template.HTML(content)
	}

	return template.HTML(buf.String())
}
