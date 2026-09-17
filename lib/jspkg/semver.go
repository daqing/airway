package jspkg

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// A minimal semver implementation covering what real npm "dependencies"
// ranges use: exact versions, "*" and "x" wildcards, "^", "~", comparison
// operators, space-separated AND lists, and "||" OR groups. Prerelease
// versions are only selectable when the range itself mentions a prerelease
// of the same [major, minor, patch] triple.

type semVer struct {
	major, minor, patch int64
	pre                 []string
}

func parseSemVer(s string) (semVer, error) {
	v := semVer{}
	core := s
	if i := strings.IndexByte(s, '-'); i >= 0 {
		core = s[:i]
		v.pre = strings.Split(s[i+1:], ".")
	}

	parts := strings.Split(core, ".")
	if len(parts) > 3 {
		return v, fmt.Errorf("invalid version %q", s)
	}
	nums := make([]int64, 3)
	for i, p := range parts {
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return v, fmt.Errorf("invalid version %q", s)
		}
		nums[i] = n
	}
	v.major, v.minor, v.patch = nums[0], nums[1], nums[2]
	return v, nil
}

func (v semVer) isPre() bool { return len(v.pre) > 0 }

func compareSemVer(a, b semVer) int {
	if c := cmpInt64(a.major, b.major); c != 0 {
		return c
	}
	if c := cmpInt64(a.minor, b.minor); c != 0 {
		return c
	}
	if c := cmpInt64(a.patch, b.patch); c != 0 {
		return c
	}
	// releases sort above prereleases
	switch {
	case !a.isPre() && !b.isPre():
		return 0
	case !a.isPre():
		return 1
	case !b.isPre():
		return -1
	}
	for i := 0; i < len(a.pre) && i < len(b.pre); i++ {
		x, y := a.pre[i], b.pre[i]
		xn, xe := strconv.ParseInt(x, 10, 64)
		yn, ye := strconv.ParseInt(y, 10, 64)
		var c int
		switch {
		case xe == nil && ye == nil:
			c = cmpInt64(xn, yn)
		case xe == nil: // numeric identifiers sort below alphanumeric
			c = -1
		case ye == nil:
			c = 1
		default:
			c = strings.Compare(x, y)
		}
		if c != 0 {
			return c
		}
	}
	return cmpInt64(int64(len(a.pre)), int64(len(b.pre)))
}

func cmpInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// --- ranges ---

type comparator struct {
	op string // "", "=", "^", "~", ">=", ">", "<=", "<"
	ver semVer
	// partial x-ranges: "1", "1.2", "1.x", "1.2.x" — major-only or
	// major.minor-only ranges match a whole interval instead of one version
	hasMinor, hasPatch bool
	any                bool // "*", "x", ""
}

type semRange struct {
	groups [][]comparator // "||" OR of AND groups
}

var rangeOps = []string{">=", "<=", "^", "~", ">", "<", "="}

func parseRange(s string) (semRange, error) {
	r := semRange{}
	for _, group := range strings.Split(s, "||") {
		comps, err := parseGroup(group)
		if err != nil {
			return r, err
		}
		if len(comps) > 0 {
			r.groups = append(r.groups, comps)
		}
	}
	if len(r.groups) == 0 {
		// empty range behaves like "*": any release version
		r.groups = [][]comparator{{{any: true}}}
	}
	return r, nil
}

func parseGroup(s string) ([]comparator, error) {
	tokens := strings.Fields(s)
	var comps []comparator
	for i := 0; i < len(tokens); i++ {
		op, rest := "", tokens[i]
		for _, candidate := range rangeOps {
			if strings.HasPrefix(rest, candidate) {
				op, rest = candidate, rest[len(candidate):]
				break
			}
		}
		// ">= 1.2.3" with a space between operator and version
		if rest == "" && i+1 < len(tokens) {
			i++
			rest = tokens[i]
		}
		c, err := parseComparator(op, rest)
		if err != nil {
			return nil, err
		}
		comps = append(comps, c)
	}
	return comps, nil
}

func parseComparator(op, s string) (comparator, error) {
	c := comparator{op: op}
	if s == "" || s == "*" || s == "x" || s == "X" {
		c.any = true
		return c, nil
	}

	core := s
	if i := strings.IndexByte(s, '-'); i >= 0 {
		core = s[:i]
	}

	parts := strings.Split(core, ".")
	pick := func(ver string, hasMinor, hasPatch bool) error {
		v, err := parseSemVer(ver)
		if err != nil {
			return err
		}
		c.ver, c.hasMinor, c.hasPatch = v, hasMinor, hasPatch
		return nil
	}

	switch len(parts) {
	case 1:
		if parts[0] == "x" || parts[0] == "X" || parts[0] == "*" {
			c.any = true
			return c, nil
		}
		if err := pick(parts[0]+".0.0"+prereleaseSuffix(s), false, false); err != nil {
			return c, fmt.Errorf("invalid range %q", s)
		}
	case 2:
		if parts[1] == "x" || parts[1] == "X" || parts[1] == "*" {
			err := pick(parts[0]+".0.0", false, false)
			return c, err
		}
		if err := pick(parts[0]+"."+parts[1]+".0"+prereleaseSuffix(s), true, false); err != nil {
			return c, fmt.Errorf("invalid range %q", s)
		}
	case 3:
		if parts[2] == "x" || parts[2] == "X" || parts[2] == "*" {
			err := pick(parts[0]+"."+parts[1]+".0", true, false)
			return c, err
		}
		if err := pick(s, true, true); err != nil {
			return c, fmt.Errorf("invalid range %q", s)
		}
	default:
		return c, fmt.Errorf("invalid range %q", s)
	}
	return c, nil
}

func prereleaseSuffix(s string) string {
	if i := strings.IndexByte(s, '-'); i >= 0 {
		return s[i:]
	}
	return ""
}

// satisfies reports whether version v matches comparator c. Prerelease
// versions require a comparator naming a prerelease of the same core
// version (npm's gating rule, simplified).
func (c comparator) satisfies(v semVer) bool {
	if c.any {
		return !v.isPre()
	}
	if v.isPre() {
		if !(c.ver.isPre() && c.ver.major == v.major && c.ver.minor == v.minor && c.ver.patch == v.patch) {
			return false
		}
	}

	switch c.op {
	case ">":
		return compareSemVer(v, c.ver) > 0
	case ">=":
		return compareSemVer(v, c.ver) >= 0
	case "<":
		return compareSemVer(v, c.ver) < 0
	case "<=":
		return compareSemVer(v, c.ver) <= 0
	}

	// exact /^/~/x-range: express as a half-open [lower, upper) interval.
	if c.hasMinor && c.hasPatch && (c.op == "" || c.op == "=") {
		return compareSemVer(v, c.ver) == 0 // exact match, prerelease included
	}

	lower, upper := c.bounds()
	if lower != nil && compareSemVer(v, *lower) < 0 {
		return false
	}
	if upper != nil && compareSemVer(v, *upper) >= 0 {
		return false
	}
	return true
}

// bounds converts ^/~ /x-range comparators into [lower, upper); nil means
// unbounded on that side.
func (c comparator) bounds() (lower, upper *semVer) {
	lo := c.ver
	u := lo
	switch c.op {
	case "^":
		switch {
		case !c.hasMinor: // "^1" → <2.0.0, "^0" → <1.0.0
			if lo.major == 0 {
				u = bump(lo, 2)
			} else {
				u = bump(lo, 2)
			}
		case !c.hasPatch: // "^1.2" → <2.0.0, "^0.2" → <0.3.0
			if lo.major == 0 {
				u = bump(lo, 1)
			} else {
				u = bump(lo, 2)
			}
		default: // "^1.2.3" / "^0.2.3" / "^0.0.3": bump the leftmost nonzero
			switch {
			case lo.major > 0:
				u = bump(lo, 2)
			case lo.minor > 0:
				u = bump(lo, 1)
			default:
				u = bump(lo, 0)
			}
		}
	case "~":
		if c.hasMinor { // "~1.2.3" / "~1.2" → <1.3.0
			u = bump(lo, 1)
		} else { // "~1" → <2.0.0
			u = bump(lo, 2)
		}
	default: // partial exact: "1" → <2.0.0, "1.2" → <1.3.0
		if c.hasMinor {
			u = bump(lo, 1)
		} else {
			u = bump(lo, 2)
		}
	}
	return &lo, &u
}

func bump(v semVer, level int) semVer {
	u := v
	switch level {
	case 0:
		u.patch++
	case 1:
		u.minor++
		u.patch = 0
	case 2:
		u.major++
		u.minor, u.patch = 0, 0
	}
	u.pre = nil
	return u
}

func (r semRange) matches(v semVer) bool {
	for _, group := range r.groups {
		ok := true
		for _, c := range group {
			if !c.satisfies(v) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// ResolveVersion picks the highest published version satisfying rangeStr.
// available entries that do not parse as semver are ignored.
func ResolveVersion(available []string, rangeStr string) (string, error) {
	rng, err := parseRange(rangeStr)
	if err != nil {
		return "", err
	}

	var best string
	var bestVer semVer
	for _, cand := range available {
		v, err := parseSemVer(cand)
		if err != nil {
			continue
		}
		if !rng.matches(v) {
			continue
		}
		if best == "" || compareSemVer(v, bestVer) > 0 {
			best, bestVer = cand, v
		}
	}
	if best == "" {
		sorted := append([]string(nil), available...)
		sort.Strings(sorted)
		return "", fmt.Errorf("no version satisfies %q (available: %v)", rangeStr, sorted)
	}
	return best, nil
}
