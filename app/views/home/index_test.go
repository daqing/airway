package home

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

func TestIndexRendersHomePage(t *testing.T) {
	html := renderToString(t, Index())

	for _, want := range []string{
		"<!doctype html>",
		`<html lang="en">`,
		"<title>Airway — The full-stack Go web framework</title>",
		`name="description"`,
		"Less setup.",
		"THE FULL-STACK GO WEB FRAMEWORK",
		`id="features"`,
		`id="get-started"`,
		`id="main"`,
		"PostgreSQL",
		"MySQL 8",
		"SQLite",
		"Find us on GitHub",
		"中文文档",
		`href="` + repositoryURL + `"`,
		`href="` + repositoryURL + `#readme"`,
		`class="home-footer`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected output to contain %q", want)
		}
	}
}

func TestIndexSupportsLightAndDarkThemes(t *testing.T) {
	html := renderToString(t, Index())

	for _, want := range []string{
		`data-theme-toggle`,
		`class="theme-toggle"`,
		`prefers-color-scheme: light`,
		`.home-page[data-theme="light"]`,
		`.home-page:not([data-theme="dark"])`,
		`color-scheme: dark`,
		`color-scheme: light`,
		`airway-theme`,
		`icon-sun`,
		`icon-moon`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected output to contain %q", want)
		}
	}
}

func TestIndexRendersAllFeatureCards(t *testing.T) {
	html := renderToString(t, Index())

	features := []struct {
		number string
		title  string
	}{
		{"01", "A data layer that speaks Go"},
		{"02", "HTML, with a Go mindset"},
		{"03", "A head start, built in"},
		{"04", "Files at home. Or in the cloud."},
		{"05", "Ready for real time"},
		{"06", "A straightforward way to ship"},
	}

	if got := strings.Count(html, `class="feature-card"`); got != len(features) {
		t.Errorf("expected %d feature cards, got %d", len(features), got)
	}

	for _, feature := range features {
		if !strings.Contains(html, feature.title) {
			t.Errorf("expected feature %q in output", feature.title)
		}
		if !strings.Contains(html, `class="feature-number">`+feature.number+`<`) {
			t.Errorf("expected feature number %q in output", feature.number)
		}
	}
}

func TestFeatureRendersCardContent(t *testing.T) {
	html := renderToString(t, feature("07", "A test title", "A test description.", "#test-anchor", "Read more", "terminal"))

	for _, want := range []string{
		`class="feature-number">07<`,
		"<h3>A test title</h3>",
		"<p>A test description.</p>",
		"Read more",
		`href="` + repositoryURL + `#test-anchor"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, html)
		}
	}
}

func TestFeatureRendersAnIconForEveryKnownKind(t *testing.T) {
	for _, icon := range []string{"data", "code", "terminal", "storage", "realtime", "deploy"} {
		html := renderToString(t, feature("01", "Title", "Description.", "#anchor", "Label", icon))

		if !strings.Contains(html, "<svg") {
			t.Errorf("expected icon kind %q to render an svg", icon)
		}
	}
}

func TestArrowRendersDirectionSpecificPath(t *testing.T) {
	up := renderToString(t, arrow("up"))
	right := renderToString(t, arrow("right"))

	if !strings.Contains(up, `d="M6 18 18 6M6 6h12v12"`) {
		t.Errorf("expected up arrow path, got:\n%s", up)
	}

	if !strings.Contains(right, `d="M4 12h16m-6-6 6 6-6 6"`) {
		t.Errorf("expected right arrow path, got:\n%s", right)
	}

	if up == right {
		t.Error("expected different directions to render different arrows")
	}
}
