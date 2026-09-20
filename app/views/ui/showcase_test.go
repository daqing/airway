package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func renderToString(t *testing.T, component templ.Component) string {
	t.Helper()

	var buf strings.Builder
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}

	return buf.String()
}

func TestShowcaseRendersComponentPage(t *testing.T) {
	html := renderToString(t, Showcase())

	for _, want := range []string{
		"<!doctype html>",
		"<title>airway-ui — component showcase</title>",
		`name="description"`,
		"The Airway component library",
		"airway-ui",
		`data-island="ui-showcase"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected output to contain %q", want)
		}
	}
}
