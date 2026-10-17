package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	upgradeManifestPath   = ".airway/manifest.json"
	upgradeManifestSchema = 1

	// maxDiffCells bounds the LCS table. The managed files are small; an
	// unexpectedly large file falls back to a conflict sidecar instead of
	// allocating a huge table.
	maxDiffCells = 4_000_000
)

// upgradeManifest records the skeleton version the project was last synced
// with and the hash of each file the framework wrote at that version. The
// hash is of the pristine template content, not of the file on disk, so a
// user-modified file is still detected after a clean merge.
type upgradeManifest struct {
	Schema          int               `json:"schema"`
	SkeletonVersion string            `json:"skeleton_version"`
	Module          string            `json:"module"`
	Files           map[string]string `json:"files"`
}

func readUpgradeManifest(dir string) (*upgradeManifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, upgradeManifestPath))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var manifest upgradeManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse %s: %w", upgradeManifestPath, err)
	}
	if manifest.Files == nil {
		manifest.Files = map[string]string{}
	}

	return &manifest, nil
}

// writeUpgradeManifest records version and the hash of every framework-owned
// file in target. It runs only after a fully successful sync, so the manifest
// never claims a version whose conflicts are still unresolved.
func writeUpgradeManifest(dir, module, version string, target map[string][]byte) error {
	manifest := upgradeManifest{
		Schema:          upgradeManifestSchema,
		SkeletonVersion: version,
		Module:          module,
		Files:           map[string]string{},
	}

	for rel, data := range target {
		mode, ok := classifySkeletonPath(rel)
		if !ok || mode == skeletonNever || mode == skeletonRegenerate {
			continue
		}
		manifest.Files[rel] = skeletonHash(data)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	path := filepath.Join(dir, upgradeManifestPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func skeletonHash(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// lineChange replaces base lines [start, end) with lines. start == end is an
// insertion.
type lineChange struct {
	start int
	end   int
	lines []string
}

// diffLineChanges returns the changes turning base into other. ok is false
// when the inputs are too large for the LCS table, in which case the caller
// must treat the file as conflicting.
func diffLineChanges(base, other []string) ([]lineChange, bool) {
	prefix := 0
	for prefix < len(base) && prefix < len(other) && base[prefix] == other[prefix] {
		prefix++
	}

	suffix := 0
	for suffix < len(base)-prefix && suffix < len(other)-prefix &&
		base[len(base)-1-suffix] == other[len(other)-1-suffix] {
		suffix++
	}

	b := base[prefix : len(base)-suffix]
	o := other[prefix : len(other)-suffix]
	if len(b) == 0 && len(o) == 0 {
		return nil, true
	}
	if len(b)*len(o) > maxDiffCells {
		return nil, false
	}

	n, m := len(b), len(o)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case b[i] == o[j]:
				dp[i][j] = dp[i+1][j+1] + 1
			case dp[i+1][j] >= dp[i][j+1]:
				dp[i][j] = dp[i+1][j]
			default:
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	type match struct{ base, other int }
	var matches []match

	for i, j := 0, 0; i < n && j < m; {
		switch {
		case b[i] == o[j]:
			matches = append(matches, match{i, j})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			i++
		default:
			j++
		}
	}

	var changes []lineChange
	baseStart, otherStart := 0, 0
	appendGap := func(baseEnd, otherEnd int) {
		if baseStart == baseEnd && otherStart == otherEnd {
			return
		}
		changes = append(changes, lineChange{
			start: baseStart + prefix,
			end:   baseEnd + prefix,
			lines: append([]string(nil), o[otherStart:otherEnd]...),
		})
	}

	for _, mt := range matches {
		appendGap(mt.base, mt.other)
		baseStart, otherStart = mt.base+1, mt.other+1
	}
	appendGap(n, m)

	return changes, true
}

func changesOverlap(a, b lineChange) bool {
	aInsert := a.start == a.end
	bInsert := b.start == b.end

	switch {
	case aInsert && bInsert:
		return a.start == b.start
	case aInsert:
		return b.start < a.start && a.start < b.end
	case bInsert:
		return a.start < b.start && b.start < a.end
	default:
		return a.start < b.end && b.start < a.end
	}
}

func sameChange(a, b lineChange) bool {
	return a.start == b.start && a.end == b.end && slices.Equal(a.lines, b.lines)
}

// linesSubsequence reports whether sub appears in seq in order (not
// necessarily contiguously).
func linesSubsequence(sub, seq []string) bool {
	i := 0
	for _, line := range seq {
		if i < len(sub) && line == sub[i] {
			i++
		}
	}
	return i == len(sub)
}

// mergeSkeletonFile applies the base -> target changes to local, keeping
// local edits that do not overlap the framework's changes. ok is false when
// the two sides touch the same region with different content.
func mergeSkeletonFile(local, base, target []byte) (merged []byte, ok bool) {
	localLines := splitLines(local)
	baseLines := splitLines(base)
	targetLines := splitLines(target)

	upstream, ok := diffLineChanges(baseLines, targetLines)
	if !ok {
		return nil, false
	}
	localChanges, ok := diffLineChanges(baseLines, localLines)
	if !ok {
		return nil, false
	}

	type appliedChange struct {
		start int
		end   int
		lines []string
	}
	var applied []appliedChange

	for _, up := range upstream {
		var overlapping []lineChange
		for _, localChange := range localChanges {
			if changesOverlap(up, localChange) {
				overlapping = append(overlapping, localChange)
			}
		}

		switch len(overlapping) {
		case 0:
			start, end := mapUpstreamChange(up, localChanges)
			applied = append(applied, appliedChange{start: start, end: end, lines: up.lines})
		case 1:
			if sameChange(up, overlapping[0]) {
				// The user already made the same change; nothing to apply.
				continue
			}
			// Two insertions at the same point are independent additions
			// (typically "a new function at the end of the file" on each
			// side); keep both instead of forcing a conflict. If the local
			// block already contains the upstream lines (a previously merged
			// file), there is nothing left to apply.
			if up.start == up.end && overlapping[0].start == overlapping[0].end &&
				up.start == overlapping[0].start {
				if linesSubsequence(up.lines, overlapping[0].lines) {
					continue
				}
				start, end := mapUpstreamChange(up, localChanges)
				applied = append(applied, appliedChange{start: start, end: end, lines: up.lines})
				continue
			}
			return nil, false
		default:
			return nil, false
		}
	}

	var out []string
	cursor := 0
	for _, change := range applied {
		if change.start < cursor || change.end < change.start {
			return nil, false
		}
		out = append(out, localLines[cursor:change.start]...)
		out = append(out, change.lines...)
		cursor = change.end
	}
	out = append(out, localLines[cursor:]...)

	return []byte(strings.Join(out, "")), true
}

// mapUpstreamChange maps one upstream change to local indices. An insertion
// maps to a single point; a replacement keeps separate start/end boundaries.
func mapUpstreamChange(up lineChange, localChanges []lineChange) (start, end int) {
	start = mapBaseIndex(up.start, localChanges, false)
	if up.start == up.end {
		return start, start
	}
	return start, mapBaseIndex(up.end, localChanges, true)
}

// mapBaseIndex maps a base line index into local, accounting for local
// changes that end at or before it. Insertions exactly at baseIndex count for
// a start boundary but are skipped for an end boundary, so an upstream
// replacement never swallows lines the user inserted at its edge.
func mapBaseIndex(baseIndex int, localChanges []lineChange, endBoundary bool) int {
	index := baseIndex
	for _, change := range localChanges {
		if change.end > baseIndex {
			break
		}
		if endBoundary && change.start == baseIndex && change.end == baseIndex {
			continue
		}
		index += len(change.lines) - (change.end - change.start)
	}
	return index
}

// splitLines keeps each line's trailing newline, so joining the result
// reproduces the input byte for byte.
func splitLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	lines := strings.SplitAfter(string(data), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
