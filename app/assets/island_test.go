package assets

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf strings.Builder
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestIslandRendersMountPointAndProps(t *testing.T) {
	html := render(t, Island("OrderList", map[string]any{"start": 3, "label": "orders"}))

	mount := regexp.MustCompile(`<div data-island="OrderList" data-island-id="(\d+)"></div>`)
	m := mount.FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("mount point missing in:\n%s", html)
	}

	if !strings.Contains(html, `<script id="island-data-`+m[1]+`" type="application/json">`) {
		t.Fatalf("props script for id %s missing in:\n%s", m[1], html)
	}
	if !strings.Contains(html, `"start":3`) {
		t.Errorf("props JSON incomplete:\n%s", html)
	}
}

func TestIslandEscapesNameAndScriptTerminator(t *testing.T) {
	html := render(t, Island(`"><script>`, map[string]any{"evil": "</script><b>x</b>"}))

	if got := strings.Count(html, "<script"); got != 1 {
		t.Errorf("expected exactly one script tag, got %d:\n%s", got, html)
	}
	if !strings.Contains(html, `data-island="&#34;&gt;&lt;script&gt;"`) {
		t.Errorf("island name was not HTML-escaped:\n%s", html)
	}
	if strings.Contains(html, "</script><b>") {
		t.Errorf("props broke out of the JSON script:\n%s", html)
	}
}

func TestIslandIdsAreUnique(t *testing.T) {
	one := render(t, Island("A", nil))
	two := render(t, Island("A", nil))
	if one == two {
		t.Errorf("two islands of the same name rendered identically:\n%s", one)
	}
}

func TestScriptsLocalAndProduction(t *testing.T) {
	t.Setenv("AIRWAY_ENV", "production")
	prod := render(t, Scripts())
	if !strings.Contains(prod, `/assets/app.js?v=`) {
		t.Errorf("production script lacks cache-busted entry:\n%s", prod)
	}
	if !strings.Contains(prod, `type="module"`) {
		t.Errorf("script not a module:\n%s", prod)
	}

	t.Setenv("AIRWAY_ENV", "local")
	dev := render(t, Scripts())
	if strings.Contains(dev, "?v=") {
		t.Errorf("dev script should hit the in-memory build directly:\n%s", dev)
	}
	if !strings.Contains(dev, `src="/assets/app.js"`) {
		t.Errorf("dev script wrong:\n%s", dev)
	}
}
