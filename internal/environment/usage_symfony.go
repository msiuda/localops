package environment

import (
	"bufio"
	"bytes"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/msiuda/localops/internal/technology"
)

// symfonyUsageScanner recognizes Symfony's statically knowable
// `%env(NAME)%` configuration syntax in YAML/XML configuration files, but
// only when Symfony was actually detected for this project (see
// internal/technology) — this is framework-specific configuration
// scanning, not generic YAML/XML scanning, and section 17 of this
// milestone's own scope is explicit that the two must not be conflated.
// Symfony's own, much larger configuration grammar (processors, defaults,
// nested resolution) is not implemented; only the literal `%env(NAME)%`
// token is recognized, never a dynamic or processor-qualified form such
// as `%env(resolve:NAME)%`.
type symfonyUsageScanner struct{}

var symfonyConfigExtensions = map[string]bool{".yaml": true, ".yml": true, ".xml": true}

func (symfonyUsageScanner) supports(path string) bool {
	return symfonyConfigExtensions[filepath.Ext(path)]
}

func (symfonyUsageScanner) scan(_ string, content []byte, detected map[technology.ID]bool) ([]usageHit, []Finding) {
	if !detected[technology.IDSymfony] {
		return nil, nil
	}
	return symfonyEnvUsages(content), nil
}

// symfonyEnvToken matches only the literal, non-processor-qualified
// `%env(NAME)%` form.
var symfonyEnvToken = regexp.MustCompile(`%env\(([A-Za-z_][A-Za-z0-9_]*)\)%`)

// symfonyEnvUsages scans data line by line (YAML/XML comments are both
// whole-line-oriented enough for this narrow, specific token that a full
// comment-aware lexer is not needed here), skipping a line whose trimmed
// content starts with a YAML `#` or XML `<!--` comment marker.
func symfonyEnvUsages(data []byte) []usageHit {
	var hits []usageHit

	scanner := bufio.NewScanner(bytes.NewReader(data))
	line := 0
	for scanner.Scan() {
		line++
		trimmed := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "<!--") {
			continue
		}
		for _, m := range symfonyEnvToken.FindAllStringSubmatch(scanner.Text(), -1) {
			name := m[1]
			if isValidEnvKey(name) {
				hits = append(hits, usageHit{Name: name, Line: line})
			}
		}
	}

	return hits
}
