package layouts

import (
	"context"
	"io"
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

func renderWithChildren(t *testing.T, component templ.Component, children string) string {
	t.Helper()

	body := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, children)
		return err
	})

	var buf strings.Builder
	ctx := templ.WithChildren(context.Background(), body)
	if err := component.Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}

	return buf.String()
}

func TestBaseRendersDocumentShell(t *testing.T) {
	html := renderWithChildren(t, Base("My Page", "My description"), "<p>Hello</p>")

	for _, want := range []string{
		"<!doctype html>",
		`<html lang="en">`,
		"<title>My Page</title>",
		`<meta name="description" content="My description">`,
		`<meta property="og:title" content="My Page">`,
		`<meta property="og:description" content="My description">`,
		`<meta property="og:type" content="website">`,
		`<meta name="theme-color" content="#101319">`,
		"<p>Hello</p>",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, html)
		}
	}
}

func TestBaseOmitsMetaDescriptionWhenNotProvided(t *testing.T) {
	html := renderWithChildren(t, Base("My Page"), "<p>Hello</p>")

	if !strings.Contains(html, "<title>My Page</title>") {
		t.Errorf("expected output to contain the title, got:\n%s", html)
	}

	for _, unwanted := range []string{
		`name="description"`,
		`property="og:title"`,
		`property="og:description"`,
		`name="theme-color"`,
	} {
		if strings.Contains(html, unwanted) {
			t.Errorf("expected output not to contain %q, got:\n%s", unwanted, html)
		}
	}
}

func TestBaseEscapesTitleAndDescription(t *testing.T) {
	html := renderToString(t, Base(`<script>alert("x")</script>`, `a "quoted" description`))

	if strings.Contains(html, "<script>") {
		t.Errorf("expected title to be escaped, got:\n%s", html)
	}

	for _, want := range []string{
		"&lt;script&gt;",
		`content="a &#34;quoted&#34; description"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, html)
		}
	}
}
