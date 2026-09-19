package cmd

import (
	"testing"
)

func TestParseGitIgnoreMatches(t *testing.T) {
	g := parseGitIgnore(`
# built frontend dependencies
node_modules/
*.log
/build
temp
**/deep/*.tmp
!keep.log
docs/**/*.md
`)

	cases := map[string]struct {
		isDir bool
		want  bool
	}{
		"node_modules":              {true, true},
		"install/host/node_modules": {true, true},
		// A directory-only rule does not match file paths; children of an
		// ignored directory are kept out by the walker pruning the directory.
		"node_modules/x.js":   {false, false},
		"debug.log":           {false, true},
		"sub/nested.log":      {false, true},
		"build":               {true, true},
		"sub/build":           {true, false},
		"temp":                {false, true},
		"install/deps/a/temp": {false, true},
		"deep/f.tmp":          {false, true},
		"a/b/deep/f.tmp":      {false, true},
		"docs/a.md":           {false, true},
		"docs/x/y.md":         {false, true},
		"docs.md":             {false, false},
		"keep.log":            {false, false},
		"sub/keep.log":        {false, false},
		"app.js":              {false, false},
		"node_modulesx":       {true, false},
	}

	for path, tc := range cases {
		if got := g.Match(path, tc.isDir); got != tc.want {
			t.Fatalf("gitIgnore.Match(%q, %v) = %v, want %v", path, tc.isDir, got, tc.want)
		}
	}
}

func TestParseGitIgnoreNegationCannotReincludeUnderSkippedDir(t *testing.T) {
	g := parseGitIgnore("node_modules/\n!node_modules/.gitkeep\n")

	// The walker prunes the ignored directory, so the negated child is never
	// reached — the directory staying ignored is what keeps it out.
	if !g.Match("node_modules", true) {
		t.Fatal("expected the node_modules directory to stay ignored")
	}
	if g.Match("node_modules/.gitkeep", false) {
		t.Fatal("expected the negation to win for a directly matched file path")
	}
}

func TestLoadPluginGitIgnoreWithoutFileIsNil(t *testing.T) {
	g, err := loadPluginGitIgnore(t.TempDir())
	if err != nil {
		t.Fatalf("load plugin gitignore: %v", err)
	}
	if g != nil {
		t.Fatal("expected a nil matcher without a .gitignore file")
	}
	if g.Match("node_modules", true) {
		t.Fatal("expected a nil matcher to match nothing")
	}
}
