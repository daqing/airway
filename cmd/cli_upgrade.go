package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/daqing/airway/cmd/clitemplate"
)

type upgradeOptions struct {
	Dir     string
	From    string
	FromDir string
	DryRun  bool
	Check   bool
	Force   bool
	NoDeps  bool
}

type upgradeAction int

const (
	upgradeNoop upgradeAction = iota
	upgradeCreate
	upgradeUpdate
	upgradeMerge
	upgradeConflict
	upgradeObsolete
	upgradeSkipped
)

type upgradePlan struct {
	Path    string
	Action  upgradeAction
	Content []byte
	Note    string
}

func runCLIUpgrade(args []string) error {
	opts := upgradeOptions{Dir: "."}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case isHelpArg(arg):
			printCLIUpgradeUsage(os.Stdout)
			return nil
		case arg == "--dry-run":
			opts.DryRun = true
		case arg == "--check":
			opts.Check = true
		case arg == "--force":
			opts.Force = true
		case arg == "--no-deps":
			opts.NoDeps = true
		case arg == "--from":
			i++
			if i >= len(args) {
				return errors.New("--from needs a version, e.g. --from v0.19.2")
			}
			opts.From = args[i]
		case strings.HasPrefix(arg, "--from="):
			opts.From = strings.TrimPrefix(arg, "--from=")
		case arg == "--from-dir":
			i++
			if i >= len(args) {
				return errors.New("--from-dir needs a path to an Airway checkout or template directory")
			}
			opts.FromDir = args[i]
		case strings.HasPrefix(arg, "--from-dir="):
			opts.FromDir = strings.TrimPrefix(arg, "--from-dir=")
		default:
			return fmt.Errorf("unknown argument %q; usage: airway upgrade [--dry-run] [--check] [--force] [--no-deps] [--from version] [--from-dir path]", arg)
		}
	}

	return runUpgrade(opts)
}

func runUpgrade(opts upgradeOptions) error {
	projectDir := opts.Dir
	if projectDir == "" {
		projectDir = "."
	}

	module, err := modulePathAt(projectDir)
	if err != nil {
		return fmt.Errorf("airway upgrade must run inside an Airway project: %w", err)
	}
	if module == airwayModulePath {
		return errors.New("airway upgrade runs in a generated project, not in the Airway framework repository")
	}
	if _, err := os.Stat(filepath.Join(projectDir, "main.go")); err != nil {
		return fmt.Errorf("%s does not look like an Airway project (main.go not found)", projectDir)
	}

	targetVersion, targetFiles, err := loadUpgradeTarget(projectDir, module)
	if err != nil {
		return err
	}

	manifest, err := readUpgradeManifest(projectDir)
	if err != nil {
		return err
	}
	pinnedVersion := projectAirwayVersion(projectDir)

	manifestVersion := ""
	if manifest != nil {
		manifestVersion = canonicalVersion(manifest.SkeletonVersion)
	}
	targetVersion = canonicalVersion(targetVersion)

	filesUpToDate := manifestVersion != "" && manifestVersion == targetVersion
	depsUpToDate := canonicalVersion(pinnedVersion) == targetVersion

	if opts.Check && filesUpToDate && depsUpToDate {
		fmt.Printf("Skeleton is up to date (%s).\n", displayVersion(targetVersion))
		return nil
	}

	if !opts.DryRun && !opts.Check && !(filesUpToDate && depsUpToDate) {
		if err := checkUpgradeWorkingTree(projectDir, opts.Force); err != nil {
			return err
		}
	}

	fmt.Printf("Upgrading skeleton %s", module)
	if manifestVersion != "" {
		fmt.Printf(" from %s", displayVersion(manifestVersion))
	}
	fmt.Printf(" to %s\n", displayVersion(targetVersion))

	plans := []upgradePlan{}
	if !filesUpToDate {
		baseFiles := map[string][]byte(nil)
		baseVersion := ""

		if manifestVersion == targetVersion ||
			(manifest == nil && canonicalVersion(pinnedVersion) == targetVersion) {
			baseFiles = targetFiles
			baseVersion = targetVersion
		} else {
			baseFiles, baseVersion, err = loadSkeletonBase(projectDir, module, opts, manifestVersion, pinnedVersion)
			if err != nil {
				if manifest == nil {
					return err
				}
				fmt.Printf("warning: %v\n", err)
				fmt.Println("         falling back to manifest hashes; modified files will be kept as conflict copies.")
				baseFiles = nil
				baseVersion = manifestVersion
			}
		}

		if baseVersion != "" && baseVersion != "unknown" && baseVersion != targetVersion {
			fmt.Printf("Baseline: %s\n", displayVersion(baseVersion))
		}

		plans, err = buildUpgradePlans(projectDir, targetFiles, baseFiles, manifest, opts.Force)
		if err != nil {
			return err
		}
		printUpgradePlans(plans)
	}

	changes, conflicts := countUpgradePlans(plans)

	if opts.DryRun {
		printUpgradeSummary(changes, conflicts, depsUpToDate)
		return nil
	}

	if opts.Check {
		printUpgradeSummary(changes, conflicts, depsUpToDate)
		if changes > 0 || conflicts > 0 || !depsUpToDate {
			return errors.New("skeleton is out of date; run `airway upgrade`")
		}
		return nil
	}

	if err := applyUpgradePlans(projectDir, plans); err != nil {
		return err
	}

	if conflicts > 0 {
		fmt.Printf("\n%d file(s) need manual resolution before the dependency can be upgraded:\n", conflicts)
		for _, plan := range plans {
			if plan.Action == upgradeConflict {
				fmt.Printf("  %s   (new version: %s.airway-new)\n", plan.Path, plan.Path)
			}
		}
		fmt.Println("\nMerge the .airway-new contents into the listed files, then re-run `airway upgrade`.")
		return errors.New("skeleton upgrade has unresolved conflicts")
	}

	if !opts.NoDeps && !depsUpToDate {
		if err := upgradeSkeletonDependency(projectDir, targetVersion); err != nil {
			return err
		}
	}

	if !filesUpToDate || manifest == nil {
		if err := writeUpgradeManifest(projectDir, module, targetVersion, targetFiles); err != nil {
			return err
		}
	}

	printUpgradeSummary(changes, conflicts, depsUpToDate)
	printUpgradeNextSteps(projectDir, plans, opts)

	return nil
}

// writeScaffoldManifest records the skeleton version a fresh project was
// generated with. Dev builds have no meaningful version to compare against
// later, so they skip the manifest and let a released CLI infer the baseline
// from go.mod instead.
func writeScaffoldManifest(destDir, module, version string) error {
	version = canonicalVersion(version)
	if version == "" || version == "dev" {
		return nil
	}

	files, err := clitemplate.Render(module)
	if err != nil {
		return err
	}
	rewriteSkeletonPorts(files)

	return writeUpgradeManifest(destDir, module, version, files)
}

// loadUpgradeTarget renders the skeleton this binary ships. A project that
// replaces airway with a local checkout upgrades to that checkout instead, so
// framework development and released CLIs behave consistently.
func loadUpgradeTarget(projectDir, module string) (string, map[string][]byte, error) {
	if dir, local, ok := goModAirwayReplace(projectDir); ok && local {
		files, err := renderSkeletonFromDir(dir, module)
		if err != nil {
			return "", nil, err
		}

		version := Version
		if data, err := os.ReadFile(filepath.Join(dir, "VERSION")); err == nil {
			if v := strings.TrimSpace(string(data)); v != "" {
				version = v
			}
		}
		return version, files, nil
	}

	if Version == "" || Version == "dev" {
		return "", nil, errors.New("this airway build has no release version; install a released CLI with `go install github.com/daqing/airway@latest`")
	}

	files, err := clitemplate.Render(module)
	if err != nil {
		return "", nil, err
	}
	rewriteSkeletonPorts(files)

	return Version, files, nil
}

func buildUpgradePlans(projectDir string, target, base map[string][]byte, manifest *upgradeManifest, force bool) ([]upgradePlan, error) {
	paths := make([]string, 0, len(target))
	for rel := range target {
		paths = append(paths, rel)
	}
	sort.Strings(paths)

	var plans []upgradePlan

	for _, rel := range paths {
		mode, ok := classifySkeletonPath(rel)
		if !ok {
			plans = append(plans, upgradePlan{Path: rel, Action: upgradeSkipped, Note: "no upgrade rule (framework bug)"})
			continue
		}
		if mode == skeletonNever || mode == skeletonRegenerate {
			continue
		}

		targetData := target[rel]
		localPath := filepath.Join(projectDir, filepath.FromSlash(rel))
		local, err := os.ReadFile(localPath)
		missing := os.IsNotExist(err)
		if err != nil && !missing {
			return nil, err
		}

		if mode == skeletonCreate {
			switch {
			case missing:
				plans = append(plans, upgradePlan{Path: rel, Action: upgradeCreate, Content: targetData})
			case !bytes.Equal(local, targetData):
				plans = append(plans, upgradePlan{Path: rel, Action: upgradeSkipped, Note: "kept existing file"})
			}
			continue
		}

		if missing {
			_, knownBase := base[rel]
			knownManifest := manifest != nil && manifest.Files[rel] != ""
			switch {
			case knownBase || knownManifest:
				if force {
					plans = append(plans, upgradePlan{Path: rel, Action: upgradeCreate, Content: targetData, Note: "restored"})
				} else {
					plans = append(plans, upgradePlan{Path: rel, Action: upgradeSkipped, Note: "deleted locally; pass --force to restore"})
				}
			default:
				plans = append(plans, upgradePlan{Path: rel, Action: upgradeCreate, Content: targetData})
			}
			continue
		}

		if bytes.Equal(local, targetData) {
			continue
		}

		baseData, hasBase := base[rel]
		if hasBase {
			switch {
			case bytes.Equal(baseData, targetData):
				continue
			case bytes.Equal(local, baseData):
				plans = append(plans, upgradePlan{Path: rel, Action: upgradeUpdate, Content: targetData})
			default:
				merged, ok := mergeSkeletonFile(local, baseData, targetData)
				if !ok {
					plans = append(plans, upgradePlan{Path: rel, Action: upgradeConflict, Content: targetData})
				} else if !bytes.Equal(merged, local) {
					plans = append(plans, upgradePlan{Path: rel, Action: upgradeMerge, Content: merged})
				}
			}
			continue
		}

		if manifest != nil && manifest.Files[rel] != "" && skeletonHash(local) == manifest.Files[rel] {
			plans = append(plans, upgradePlan{Path: rel, Action: upgradeUpdate, Content: targetData})
			continue
		}

		plans = append(plans, upgradePlan{Path: rel, Action: upgradeConflict, Content: targetData})
	}

	obsolete := map[string]bool{}
	for rel := range base {
		obsolete[rel] = true
	}
	if manifest != nil {
		for rel := range manifest.Files {
			obsolete[rel] = true
		}
	}
	for rel := range obsolete {
		if _, ok := target[rel]; ok {
			continue
		}
		if mode, ok := classifySkeletonPath(rel); ok && (mode == skeletonNever || mode == skeletonRegenerate) {
			continue
		}
		if _, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(rel))); err == nil {
			plans = append(plans, upgradePlan{Path: rel, Action: upgradeObsolete})
		}
	}

	sort.SliceStable(plans, func(i, j int) bool { return plans[i].Path < plans[j].Path })

	return plans, nil
}

func applyUpgradePlans(projectDir string, plans []upgradePlan) error {
	for _, plan := range plans {
		path := filepath.Join(projectDir, filepath.FromSlash(plan.Path))

		switch plan.Action {
		case upgradeCreate, upgradeUpdate, upgradeMerge:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if info, err := os.Stat(path); err == nil {
				mode = info.Mode().Perm()
			}
			if err := os.WriteFile(path, plan.Content, mode); err != nil {
				return err
			}
		case upgradeConflict:
			sidecar := path + ".airway-new"
			if err := os.MkdirAll(filepath.Dir(sidecar), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(sidecar, plan.Content, 0o644); err != nil {
				return err
			}
		}
	}

	return nil
}

func printUpgradePlans(plans []upgradePlan) {
	for _, plan := range plans {
		switch plan.Action {
		case upgradeCreate:
			if plan.Note != "" {
				fmt.Printf("  create    %s (%s)\n", plan.Path, plan.Note)
			} else {
				fmt.Printf("  create    %s\n", plan.Path)
			}
		case upgradeUpdate:
			fmt.Printf("  update    %s\n", plan.Path)
		case upgradeMerge:
			fmt.Printf("  merge     %s\n", plan.Path)
		case upgradeConflict:
			fmt.Printf("  conflict  %s -> %s.airway-new\n", plan.Path, plan.Path)
		case upgradeObsolete:
			fmt.Printf("  obsolete  %s (left in place)\n", plan.Path)
		case upgradeSkipped:
			if plan.Note != "" {
				fmt.Printf("  skip      %s (%s)\n", plan.Path, plan.Note)
			}
		}
	}
}

func countUpgradePlans(plans []upgradePlan) (changes, conflicts int) {
	for _, plan := range plans {
		switch plan.Action {
		case upgradeCreate, upgradeUpdate, upgradeMerge:
			changes++
		case upgradeConflict:
			changes++
			conflicts++
		}
	}
	return changes, conflicts
}

func printUpgradeSummary(changes, conflicts int, depsUpToDate bool) {
	if changes == 0 && conflicts == 0 && depsUpToDate {
		fmt.Println("\nAlready up to date.")
		return
	}
	fmt.Printf("\n%d file(s) changed", changes)
	if conflicts > 0 {
		fmt.Printf(", %d conflict(s)", conflicts)
	}
	if !depsUpToDate {
		fmt.Print("; dependency updated")
	}
	fmt.Println()
}

func printUpgradeNextSteps(projectDir string, plans []upgradePlan, opts upgradeOptions) {
	if missing, err := missingEnvKeys(projectDir); err == nil && len(missing) > 0 {
		fmt.Println("\nNew keys in .env.example are not set in .env:")
		for _, key := range missing {
			fmt.Printf("  %s\n", key)
		}
		fmt.Println("Add them to .env to pick up the new defaults (upgrade never edits .env itself).")
	}

	viewsChanged := false
	jsChanged := false
	jsManifestChanged := false
	for _, plan := range plans {
		switch plan.Action {
		case upgradeCreate, upgradeUpdate, upgradeMerge:
			if strings.HasSuffix(plan.Path, ".templ") {
				viewsChanged = true
			}
			if strings.HasPrefix(plan.Path, "app/assets/js/") || strings.HasPrefix(plan.Path, "app/assets/css/") {
				jsChanged = true
			}
			if plan.Path == "js.pkg.json" {
				jsManifestChanged = true
			}
		}
	}

	if viewsChanged && !opts.NoDeps {
		if err := generateTemplViewsIn(projectDir); err != nil {
			fmt.Printf("\nCould not regenerate the templ views automatically: %v\n", err)
			fmt.Println("Run `airway templates:compile` before building the project.")
			viewsChanged = false
		}
	}

	if viewsChanged {
		fmt.Println("\nViews changed; run `airway templates:compile` if you did not already.")
	}
	if jsManifestChanged {
		fmt.Println("js.pkg.json changed; run `airway js:install`.")
	}
	if jsChanged {
		fmt.Println("Frontend sources changed; run `airway js:build` and commit app/assets/dist.")
	}
}

func generateTemplViewsIn(dir string) error {
	gen := exec.Command("go", "tool", "templ", "generate", "-path", filepath.Join("app", "views"))
	gen.Dir = dir
	gen.Stdout = os.Stdout
	gen.Stderr = os.Stderr
	return gen.Run()
}

func upgradeSkeletonDependency(projectDir, version string) error {
	if _, local, ok := goModAirwayReplace(projectDir); ok && local {
		fmt.Println("\nKeeping the local airway replace directive; go.mod left unchanged.")
		return nil
	}

	fmt.Printf("\nUpdating github.com/daqing/airway to %s...\n", version)
	if err := runScaffoldCommand(projectDir, scaffoldCommandTimeout, false, "go", "get", airwayModulePath+"@"+version); err != nil {
		return fmt.Errorf("go get %s@%s: %w", airwayModulePath, version, err)
	}
	if err := runScaffoldCommand(projectDir, scaffoldCommandTimeout, false, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	return nil
}

func checkUpgradeWorkingTree(dir string, force bool) error {
	git := exec.Command("git", "rev-parse", "--git-dir")
	git.Dir = dir
	if err := git.Run(); err != nil {
		if force {
			return nil
		}
		return errors.New("not a git repository; upgrade rewrites project files, so commit the project first or pass --force")
	}

	status := exec.Command("git", "status", "--porcelain", "--untracked-files=no")
	status.Dir = dir
	out, err := status.Output()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if changes := strings.TrimSpace(string(out)); changes != "" && !force {
		return fmt.Errorf("working tree has uncommitted changes; commit or stash them first (or pass --force):\n%s", changes)
	}

	return nil
}

// projectAirwayVersion returns the framework version the project builds with.
// The go.mod require line is authoritative; `go list` covers projects whose
// require was dropped by a tidy fallback.
func projectAirwayVersion(dir string) string {
	if version, ok := goModAirwayVersion(dir); ok && version != "" {
		return version
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-f", "{{.Version}}", airwayModulePath)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

// canonicalVersion keeps versions in the v-prefixed form go.mod and VERSION
// use. "dev" and an empty string pass through unchanged.
func canonicalVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || version == "dev" {
		return version
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

// displayVersion renders a version for user-facing output, prefixing a bare
// number with v but leaving "dev"/"unknown" alone.
func displayVersion(version string) string {
	switch version {
	case "", "dev", "unknown":
		return version
	default:
		return "v" + normalizeVersion(version)
	}
}

func printCLIUpgradeUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway upgrade [--dry-run] [--check] [--force] [--no-deps] [--from version] [--from-dir path]")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Upgrades the project skeleton and the github.com/daqing/airway requirement")
	_, _ = fmt.Fprintln(w, "to this CLI's version. Framework-owned files are merged; user edits are kept.")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "examples:")
	_, _ = fmt.Fprintln(w, "  airway upgrade                    # sync this project with the installed CLI")
	_, _ = fmt.Fprintln(w, "  airway upgrade --dry-run          # show what would change")
	_, _ = fmt.Fprintln(w, "  airway upgrade --check            # exit non-zero when the project is behind")
	_, _ = fmt.Fprintln(w, "  airway upgrade --from v0.19.2     # legacy project with no .airway/manifest.json")
}
