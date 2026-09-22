package corporate

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/daqing/airway/lib/ssg"
)

func render(t *testing.T, comp templ.Component) string {
	t.Helper()

	var buf bytes.Buffer
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func testContext(slug string, title string, data any) *ssg.Context {
	return &ssg.Context{
		Meta: ssg.Meta{Title: "Acme", Description: "Test site"},
		Page: ssg.Page{Slug: slug, Title: title, Data: data},
		Pages: []ssg.Page{
			{Slug: "/", Title: "Home"},
			{Slug: "/about", Title: "About"},
		},
	}
}

func TestThemeImplementsSiteTheme(t *testing.T) {
	if Theme.Name() != "corporate" {
		t.Fatalf("theme name = %q, want %q", Theme.Name(), "corporate")
	}
}

func TestRenderPageDataTypes(t *testing.T) {
	custom := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<p>custom body</p>")
		return err
	})

	tests := []struct {
		name string
		slug string
		data any
		want []string
	}{
		{
			name: "home",
			slug: "/",
			data: Home{
				Heading: "We build ships",
				Tagline: "Since 1951",
				Features: []Feature{
					{Title: "Design", Body: "Hull-first thinking."},
					{Title: "Service", Body: "Worldwide harbors."},
				},
			},
			want: []string{
				"<title>Acme</title>",
				"We build ships",
				"Since 1951",
				"Design",
				"Worldwide harbors.",
			},
		},
		{
			name: "content",
			slug: "/about",
			data: Content{
				Lead:       "Family-owned since 1951.",
				Paragraphs: []string{"We operate out of Hamburg.", "Our fleet counts 12 vessels."},
			},
			want: []string{
				"<title>About · Acme</title>",
				"Family-owned since 1951.",
				"Our fleet counts 12 vessels.",
			},
		},
		{
			name: "custom component",
			slug: "/about",
			data: custom,
			want: []string{"<p>custom body</p>"},
		},
		{
			name: "nil data",
			slug: "/about",
			data: nil,
			want: []string{`<a class="brand" href="/">Acme</a>`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			html := render(t, Theme.Render(testContext(tc.slug, "About", tc.data)))
			for _, want := range tc.want {
				if !strings.Contains(html, want) {
					t.Fatalf("rendered page missing %q:\n%s", want, html)
				}
			}
		})
	}
}

func TestRenderMarksActiveNavEntry(t *testing.T) {
	html := render(t, Theme.Render(testContext("/about", "About", nil)))

	if !strings.Contains(html, `href="/about" class="active"`) {
		t.Fatalf("expected /about marked active:\n%s", html)
	}
	if strings.Contains(html, `href="/" class="active"`) {
		t.Fatalf("did not expect / marked active:\n%s", html)
	}
}

func TestAssetsShipStylesheet(t *testing.T) {
	assets := Theme.Assets()
	if assets == nil {
		t.Fatalf("expected theme assets")
	}

	data, err := fs.ReadFile(assets, "corporate.css")
	if err != nil {
		t.Fatalf("read corporate.css: %v", err)
	}
	if !strings.Contains(string(data), ".site-header") {
		t.Fatalf("corporate.css missing expected rules")
	}
}

func TestPageTitle(t *testing.T) {
	tests := []struct {
		slug  string
		title string
		want  string
	}{
		{"/", "Home", "Acme"},
		{"/about", "About", "About · Acme"},
		{"/about", "", "Acme"},
	}

	for _, tc := range tests {
		ctx := testContext(tc.slug, tc.title, nil)
		if got := pageTitle(ctx); got != tc.want {
			t.Fatalf("pageTitle(%q, %q) = %q, want %q", tc.slug, tc.title, got, tc.want)
		}
	}
}
