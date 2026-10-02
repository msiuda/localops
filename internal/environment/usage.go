package environment

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/msiuda/localops/internal/technology"
)

// Usage is a single statically detected source-code reference to an
// environment variable. Like the rest of this package, it never carries
// the variable's value — only where it was referenced.
type Usage struct {
	File string // project-relative, forward-slash separated
	Line int
}

// usageHit is a single in-progress detection within one already-read
// file, before it is grouped by variable name into the project-wide
// per-variable map scanSourceUsages returns.
type usageHit struct {
	Name string
	Line int
}

// maxSourceFileBytes bounds how large a single source file may be before
// source-usage scanning skips it, mirroring maxEnvLineBytes's role for
// env-style files: a fixed limit, not a configurable option, well above
// any realistic hand-written source file.
const maxSourceFileBytes = 1 << 20 // 1 MiB

// ignoredSourceDirs are directory names skipped entirely during source
// usage scanning: dependency, build, and VCS trees that are never meant
// to be read as project source.
//
// "bin" is deliberately not included: it is reliably generated output in
// some ecosystems but holds genuine project source/scripts in others (for
// example Ruby's bin/rails, bin/setup, or Symfony's bin/console, which
// Environment and technology detection both treat as meaningful). A name
// is only added here when it is reliably generated/dependency content
// across the ecosystems LocalOps supports.
var ignoredSourceDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"coverage":     true,
	".next":        true,
	"target":       true,
	"obj":          true,
	".gradle":      true,
	".idea":        true,
	".vscode":      true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
}

// usageScanner recognizes statically-knowable environment-variable usage
// for one supported source language.
//
// Environment introduces this small, module-local contract because it
// already has two genuinely different implementations behind it — a
// conservative lexical scanner for JavaScript/TypeScript, and an
// AST-based scanner for Go using the standard library's own parser — not
// as a speculative plugin framework. It describes only Environment
// source-usage scanning; it is not, and is not meant to become, a
// general LocalOps "technology" abstraction. See docs/architecture.md
// for the general principle this follows.
type usageScanner interface {
	// supports reports whether this scanner recognizes path's language,
	// based on its extension.
	supports(path string) bool
	// scan recognizes supported environment-variable usage in content, a
	// single already-read source file at the project-relative path.
	// detected is the project's Technology Intelligence detection result
	// (see internal/technology) — most scanners ignore it, but a
	// framework-specific form (Laravel's env(), Symfony's %env()%) must
	// only be recognized when that framework was actually detected,
	// never merely because the syntax looks right. scan never fails: an
	// unparseable file is represented as a Finding (naming only path,
	// never its contents), not an error, so one bad file never aborts
	// the whole scan.
	scan(path string, content []byte, detected map[technology.ID]bool) ([]usageHit, []Finding)
}

// builtinUsageScanners is the static, built-in list of supported
// language scanners. Adding a future language means adding its
// implementation (its own usage_<language>.go) and registering it here —
// existing scanners never need to change. There is deliberately no
// runtime plugin loading, reflection, init-based registration, or global
// mutable registry: just this obvious, deterministic list.
var builtinUsageScanners = []usageScanner{
	javascriptUsageScanner{},
	goUsageScanner{},
	pythonUsageScanner{},
	phpUsageScanner{},
	rustUsageScanner{},
	javaUsageScanner{},
	csharpUsageScanner{},
	rubyUsageScanner{},
	kotlinUsageScanner{},
	cUsageScanner{},
	cppUsageScanner{},
	symfonyUsageScanner{},
}

// scannerFor returns the first built-in scanner that recognizes path, or
// nil if none does.
func scannerFor(path string) usageScanner {
	for _, s := range builtinUsageScanners {
		if s.supports(path) {
			return s
		}
	}
	return nil
}

// scanSourceUsages recursively scans root for statically recognizable
// environment-variable usages via the built-in usageScanners, returning
// the detected usages grouped by variable name and any safe, isolated
// scanning findings (an oversized, unreadable, or unparseable file never
// aborts the scan — it becomes a Finding naming only the file).
//
// detected is the project's Technology Intelligence detection result,
// gating framework-specific forms (see usageScanner.scan). root is walked
// exactly once; every supported file is dispatched to its scanner from
// that single traversal, never one traversal per language.
//
// readFile is the same seam Analyze uses for contract/local files, so
// tests can substitute it without touching the real filesystem content.
// Directory traversal itself always uses the real filesystem, since
// recognizing which files exist is not a value-carrying operation.
func scanSourceUsages(root string, readFile readFileFunc, detected map[technology.ID]bool) (map[string][]Usage, []Finding, error) {
	usages := make(map[string][]Usage)
	var findings []Finding

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			rel := relSourcePath(root, path)
			findings = append(findings, Finding{Source: rel, Detail: "could not be scanned"})
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			if path != root && ignoredSourceDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		// filepath.WalkDir reports a symlink's own (unfollowed) type, so a
		// symlinked file is never read as source.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		scanner := scannerFor(path)
		if scanner == nil {
			return nil
		}
		rel := relSourcePath(root, path)

		info, err := d.Info()
		if err != nil {
			findings = append(findings, Finding{Source: rel, Detail: "could not be scanned"})
			return nil
		}
		if info.Size() > maxSourceFileBytes {
			findings = append(findings, Finding{Source: rel, Detail: "source file exceeds scan size limit"})
			return nil
		}

		data, err := readFile(path)
		if err != nil {
			findings = append(findings, Finding{Source: rel, Detail: "could not be scanned"})
			return nil
		}

		hits, fileFindings := scanner.scan(rel, data, detected)
		findings = append(findings, fileFindings...)
		for _, hit := range hits {
			usages[hit.Name] = append(usages[hit.Name], Usage{File: rel, Line: hit.Line})
		}
		return nil
	})
	if walkErr != nil {
		return nil, nil, fmt.Errorf("scan source usages: %w", walkErr)
	}

	for name, locs := range usages {
		sort.Slice(locs, func(i, j int) bool {
			if locs[i].File != locs[j].File {
				return locs[i].File < locs[j].File
			}
			return locs[i].Line < locs[j].Line
		})
		usages[name] = locs
	}

	return usages, findings, nil
}

// relSourcePath returns path relative to root, using forward slashes so
// locations are stable across platforms. If the relative path cannot be
// computed, path itself is used.
func relSourcePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	return filepath.ToSlash(rel)
}
