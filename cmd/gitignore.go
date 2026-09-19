package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// gitIgnore matches paths against the rules of a plugin's .gitignore file, so
// `plugin:install` skips the same files git would (node_modules, build
// outputs, ...). It supports the common rule set: blank lines, # comments,
// ! negation (the last matching rule wins), a trailing / for directory-only
// rules, a leading / or an inner slash for rules anchored to the plugin root,
// and the * / ? / ** wildcards. Like git, a negation cannot re-include files
// under a directory the walker has already skipped.
type gitIgnore struct {
	rules []gitIgnoreRule
}

type gitIgnoreRule struct {
	negated bool
	dirOnly bool
	re      *regexp.Regexp
}

// loadPluginGitIgnore reads the .gitignore at the plugin module root. A
// missing file yields a nil matcher, which matches nothing.
func loadPluginGitIgnore(moduleDir string) (*gitIgnore, error) {
	data, err := os.ReadFile(filepath.Join(moduleDir, ".gitignore"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return parseGitIgnore(string(data)), nil
}

// parseGitIgnore compiles the rules of a .gitignore file; lines it cannot
// make sense of are skipped.
func parseGitIgnore(content string) *gitIgnore {
	g := &gitIgnore{}
	for _, line := range strings.Split(content, "\n") {
		if rule, ok := compileGitIgnoreRule(line); ok {
			g.rules = append(g.rules, rule)
		}
	}

	return g
}

// Match reports whether path (slash-separated, relative to the .gitignore's
// directory) is ignored; isDir tells whether the path is a directory, so
// directory-only rules can apply to it alone. A nil matcher matches nothing.
func (g *gitIgnore) Match(path string, isDir bool) bool {
	if g == nil {
		return false
	}

	ignored := false
	for _, rule := range g.rules {
		if rule.dirOnly && !isDir {
			continue
		}
		if rule.re.MatchString(path) {
			ignored = !rule.negated
		}
	}

	return ignored
}

func compileGitIgnoreRule(line string) (gitIgnoreRule, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return gitIgnoreRule{}, false
	}

	rule := gitIgnoreRule{}
	if strings.HasPrefix(line, "!") {
		rule.negated = true
		line = strings.TrimPrefix(line, "!")
	}
	if strings.HasSuffix(line, "/") {
		rule.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}

	anchored := false
	if strings.HasPrefix(line, "/") {
		anchored = true
		line = strings.TrimPrefix(line, "/")
	}
	if strings.Contains(line, "/") {
		anchored = true
	}
	if line == "" {
		return gitIgnoreRule{}, false
	}

	segments := strings.Split(line, "/")
	expr := "^"
	switch {
	case segments[0] == "**":
		// A leading ** matches at any depth.
		segments = segments[1:]
		expr += `(?:.*/)?`
	case anchored:
		// An inner or leading slash anchors the rule to the plugin root.
	default:
		// A bare name matches at any depth.
		expr += `(?:.*/)?`
	}

	// A middle ** swallows the separator that follows it: its group already
	// ends with a slash per matched directory.
	skipSlash := false
	for i, segment := range segments {
		if i > 0 && !skipSlash {
			expr += "/"
		}
		skipSlash = false
		switch {
		case segment == "**" && i == len(segments)-1:
			// A trailing ** matches everything inside the directory.
			expr += ".*"
		case segment == "**":
			// A middle ** matches zero or more directories.
			expr += `(?:[^/]*/)*`
			skipSlash = true
		default:
			expr += translateGitIgnoreSegment(segment)
		}
	}

	switch {
	case len(segments) == 0:
		// The pattern was just "**": match everything.
		expr += ".*"
	case segments[len(segments)-1] != "**":
		expr += `(?:/.*)?$`
	}

	re, err := regexp.Compile(expr)
	if err != nil {
		return gitIgnoreRule{}, false
	}

	rule.re = re
	return rule, true
}

// translateGitIgnoreSegment translates one pattern segment into a regular
// expression; * and ? do not cross segment boundaries.
func translateGitIgnoreSegment(segment string) string {
	var expr strings.Builder
	for _, r := range segment {
		switch r {
		case '*':
			expr.WriteString("[^/]*")
		case '?':
			expr.WriteString("[^/]")
		default:
			expr.WriteString(regexp.QuoteMeta(string(r)))
		}
	}

	return expr.String()
}
