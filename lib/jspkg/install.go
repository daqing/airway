package jspkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Options configures Add/Install.
type Options struct {
	// Registry overrides DefaultRegistry (typically from AIRWAY_JS_REGISTRY).
	Registry string
	// Log receives progress output; nil stays silent.
	Log func(format string, args ...any)
	// VendorDir overrides the destination directory relative to root
	// (tests point it at a temp dir).
	VendorDir string
}

// ParseSpec splits a "name" / "name@version" / "@scope/name@version"
// argument.
func ParseSpec(spec string) (name, version string, err error) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == "@" {
		return "", "", fmt.Errorf("empty package spec")
	}
	if strings.HasPrefix(spec, "@") {
		rest := spec[1:]
		if i := strings.IndexByte(rest, '@'); i >= 0 {
			name, version = spec[:1+i], rest[i+1:]
		} else {
			name = spec
		}
	} else if i := strings.LastIndexByte(spec, '@'); i > 0 {
		name, version = spec[:i], spec[i+1:]
	} else {
		name = spec
	}
	if version != "" && strings.ContainsAny(version, "/ ") {
		return "", "", fmt.Errorf("invalid version %q in %q", version, spec)
	}
	return name, version, nil
}

// Add resolves spec to an exact version, adds it to manifest deps, resolves
// the full dependency tree into lock, downloads everything missing into the
// vendor directory and writes the manifest back.
func Add(root, spec string, opt Options) error {
	name, version, err := ParseSpec(spec)
	if err != nil {
		return err
	}

	client := NewClient(opt.Registry, opt.Log)
	m, err := ReadManifest(root)
	if err != nil {
		return err
	}

	pack, err := client.FetchPackument(name)
	if err != nil {
		return err
	}
	if version == "" {
		version = pack.DistTags["latest"]
		if version == "" {
			return fmt.Errorf("package %s has no latest tag", name)
		}
	}
	exact, err := ResolveVersion(versionKeys(pack), version)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	m.Deps[name] = exact
	if err := resolveTree(client, m); err != nil {
		return err
	}
	if err := downloadAll(client, root, m, opt); err != nil {
		return err
	}
	return WriteManifest(root, m)
}

// Install makes the vendor directory match js.pkg.json. With an up-to-date
// lock it installs exactly the pinned versions (no version resolution); a
// missing or stale lock is re-resolved first. Installing an already-correct
// tree is a no-op.
func Install(root string, opt Options) error {
	m, err := ReadManifest(root)
	if err != nil {
		return err
	}
	if len(m.Deps) == 0 {
		return fmt.Errorf("no dependencies in %s; add one with `airway js:add <pkg>[@version]`", ManifestFile)
	}

	client := NewClient(opt.Registry, opt.Log)
	if !lockCovers(m) {
		if err := resolveTree(client, m); err != nil {
			return err
		}
	}
	if err := downloadAll(client, root, m, opt); err != nil {
		return err
	}
	return WriteManifest(root, m)
}

// lockCovers reports whether every dep is pinned in lock at the same
// version.
func lockCovers(m *Manifest) bool {
	for name, version := range m.Deps {
		if lp, ok := m.Lock[name]; !ok || lp.Version != version {
			return false
		}
	}
	return true
}

func versionKeys(p *Packument) []string {
	keys := make([]string, 0, len(p.Versions))
	for v := range p.Versions {
		keys = append(keys, v)
	}
	return keys
}

// resolveTree rebuilds manifest.Lock from manifest.Deps by walking the
// dependency graph. peerDependencies are ignored (satisfied at build time
// by the react → preact/compat alias); optionalDependencies join the walk
// and are best-effort. The vendor layout is flat, so when different ranges
// ask for different versions of one package, a single version satisfying
// every range seen so far is chosen; if none exists the error names the
// conflicting package.
func resolveTree(client *Client, m *Manifest) error {
	type entry struct{ name, rng string }
	queue := make([]entry, 0, len(m.Deps))
	for name, version := range m.Deps {
		queue = append(queue, entry{name, version})
	}

	packuments := map[string]*Packument{}
	rangesSeen := map[string][]string{}
	lock := map[string]LockedPkg{}
	expanded := map[string]bool{} // "name@version" whose deps are enqueued

	for len(queue) > 0 {
		e := queue[0]
		queue = queue[1:]

		if _, seen := rangesSeen[e.name]; !seen {
			rangesSeen[e.name] = nil
		}
		rangesSeen[e.name] = append(rangesSeen[e.name], e.rng)

		pack, ok := packuments[e.name]
		if !ok {
			var err error
			pack, err = client.FetchPackument(e.name)
			if err != nil {
				return err
			}
			packuments[e.name] = pack
		}

		if chosen, ok := lock[e.name]; ok {
			if _, err := ResolveVersion([]string{chosen.Version}, e.rng); err == nil {
				continue // pinned version still satisfies the new range
			}
		}

		// pick the highest version satisfying every range seen for this package
		exact, err := satisfyingAll(versionKeys(pack), rangesSeen[e.name])
		if err != nil {
			return fmt.Errorf("%s: %w", e.name, err)
		}
		meta := pack.Versions[exact]
		lock[e.name] = LockedPkg{Version: exact, Integrity: meta.Dist.Integrity}

		key := e.name + "@" + exact
		if !expanded[key] {
			expanded[key] = true
			deps := map[string]string{}
			for k, v := range meta.Dependencies {
				deps[k] = v
			}
			for k, v := range meta.OptionalDependencies {
				deps[k] = v
			}
			for depName := range deps {
				queue = append(queue, entry{depName, deps[depName]})
			}
		}
	}

	m.Lock = lock
	return nil
}

// satisfyingAll filters available versions through every range, then picks
// the highest survivor.
func satisfyingAll(available []string, ranges []string) (string, error) {
	candidates := append([]string(nil), available...)
	for _, rng := range ranges {
		r, err := parseRange(rng)
		if err != nil {
			return "", err
		}
		kept := make([]string, 0, len(candidates))
		for _, cand := range candidates {
			v, err := parseSemVer(cand)
			if err != nil {
				continue
			}
			if r.matches(v) {
				kept = append(kept, cand)
			}
		}
		candidates = kept
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("ranges %v have no common version", ranges)
	}
	return ResolveVersion(candidates, "*")
}

// downloadAll unpacks every locked package into the vendor directory,
// skipping ones already installed at the pinned version.
func downloadAll(client *Client, root string, m *Manifest, opt Options) error {
	vendor := opt.VendorDir
	if vendor == "" {
		vendor = VendorDir
	}
	vendor = filepath.Join(root, vendor)

	names := make([]string, 0, len(m.Lock))
	for name := range m.Lock {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		lp := m.Lock[name]
		dir := filepath.Join(vendor, filepath.FromSlash(name))
		if installedVersion(dir) == lp.Version {
			client.logf("cached %s@%s", name, lp.Version)
			continue
		}

		client.logf("downloading %s@%s", name, lp.Version)
		meta, err := client.FetchVersion(name, lp.Version)
		if err != nil {
			return err
		}
		data, err := client.FetchTarball(meta.Dist.Tarball)
		if err != nil {
			return err
		}
		integrity := lp.Integrity
		if integrity == "" {
			integrity = meta.Dist.Integrity
		}
		if err := CheckIntegrity(data, integrity); err != nil {
			return fmt.Errorf("%s@%s: %w", name, lp.Version, err)
		}
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		if err := ExtractTarballBytes(data, dir); err != nil {
			return err
		}
	}
	return nil
}
