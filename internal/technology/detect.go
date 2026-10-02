package technology

import (
	"fmt"
	"os"
	"sort"
)

// readFileFunc mirrors os.ReadFile's signature, allowing tests to
// substitute file reading instead of depending on the real filesystem.
type readFileFunc func(name string) ([]byte, error)

// detector recognizes one technology family from a project's root-level
// manifests/markers.
//
// This is a small, module-local contract: internal/technology introduces it
// because it already needs multiple genuinely different per-ecosystem
// implementations (JavaScript's package.json, Go's go.mod, PHP's
// composer.json, and so on), not as a speculative plugin framework. It
// describes only technology detection; it is not, and is not meant to
// become, a universal "Technology" god-interface — Doctor, Validation, and
// Environment each keep their own small, independent contracts. See
// docs/architecture.md.
type detector interface {
	// detect inspects root (an already-resolved, existing project
	// directory) and returns every technology it found evidence for,
	// plus any safe, isolated finding (for example, a manifest that
	// exists but could not be parsed). It never fails outright for one
	// bad manifest — detection continues with everything else.
	detect(root string, readFile readFileFunc) ([]Detected, []Finding)
}

// builtinDetectors is the static, built-in list of supported ecosystem
// detectors. Adding a future ecosystem means adding its own detector
// implementation (its own file) and appending one line here — existing
// detectors never need to change. There is deliberately no runtime plugin
// loading, reflection, init-based registration, or global mutable registry:
// just this obvious, deterministic list.
var builtinDetectors = []detector{
	javascriptDetector{},
	pythonDetector{},
	phpDetector{},
	goDetector{},
	rustDetector{},
	javaDetector{},
	dotnetDetector{},
	rubyDetector{},
	kotlinDetector{},
	cppDetector{},
}

// Detect inspects the project at root for every technology LocalOps
// recognizes evidence for.
//
// Detect operates purely on a filesystem path — never a registered project
// or any desktop-facing type — so a future Project Discovery feature can
// call it directly against candidate folders.
//
// Detection only ever reads known root-level manifest/marker files; it does
// not walk the project's source tree (that is Environment's own, separate
// concern). A manifest that cannot be read or parsed becomes a Finding, not
// a fatal error — detection continues with every other detector. An error is
// returned only when root itself cannot be resolved as a directory.
func Detect(root string) (Result, error) {
	return detect(root, os.ReadFile)
}

// detect implements Detect, taking file reading as a small function seam so
// tests can substitute it instead of depending on the real filesystem.
func detect(root string, readFile readFileFunc) (Result, error) {
	info, err := os.Stat(root)
	if err != nil {
		return Result{}, fmt.Errorf("technology: %w", err)
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("technology: %q is not a directory", root)
	}

	var all []Detected
	var findings []Finding
	for _, d := range builtinDetectors {
		detected, fileFindings := d.detect(root, readFile)
		all = append(all, detected...)
		findings = append(findings, fileFindings...)
	}

	all = dedupeDetected(all)
	sortDetected(all)

	return Result{Path: root, Detected: all, Findings: findings}, nil
}

// dedupeDetected merges entries that share the same ID (for example, Spring
// Boot evidence found independently by both the Java and Kotlin detectors)
// into a single Detected with a combined, deduplicated Evidence list, so a
// technology is never reported twice.
func dedupeDetected(all []Detected) []Detected {
	byID := make(map[ID]*Detected, len(all))
	var order []ID

	for _, d := range all {
		existing, ok := byID[d.ID]
		if !ok {
			merged := d
			merged.Evidence = append([]Evidence(nil), d.Evidence...)
			byID[d.ID] = &merged
			order = append(order, d.ID)
			continue
		}
		existing.Evidence = dedupeEvidence(append(existing.Evidence, d.Evidence...))
	}

	deduped := make([]Detected, 0, len(order))
	for _, id := range order {
		deduped = append(deduped, *byID[id])
	}
	return deduped
}

// dedupeEvidence removes exact duplicate Evidence entries, preserving
// order.
func dedupeEvidence(evidence []Evidence) []Evidence {
	seen := make(map[Evidence]bool, len(evidence))
	deduped := make([]Evidence, 0, len(evidence))
	for _, e := range evidence {
		if seen[e] {
			continue
		}
		seen[e] = true
		deduped = append(deduped, e)
	}
	return deduped
}

// kindOrder gives languages priority over runtimes/platforms over
// frameworks, so technologies are deterministically grouped for display
// regardless of detector iteration order.
var kindOrder = map[Kind]int{
	KindLanguage:  0,
	KindRuntime:   1,
	KindPlatform:  1,
	KindFramework: 2,
}

// sortDetected sorts in place by kind (language, then runtime/platform,
// then framework), then by display name, so results are deterministic
// regardless of detector iteration order or filesystem traversal order.
func sortDetected(all []Detected) {
	sort.SliceStable(all, func(i, j int) bool {
		if kindOrder[all[i].Kind] != kindOrder[all[j].Kind] {
			return kindOrder[all[i].Kind] < kindOrder[all[j].Kind]
		}
		return all[i].Name < all[j].Name
	})
}
