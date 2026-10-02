package environment

import (
	"path/filepath"
	"regexp"

	"github.com/msiuda/localops/internal/technology"
)

// rustUsageScanner recognizes std::env::var("NAME") and
// std::env::var_os("NAME") in Rust source files. The fully-qualified form
// is always recognized; the bare env::var(...)/env::var_os(...) form is
// only recognized when the file's own `use std::env` (or a more specific
// `use std::env::...`) import can be found — conservatively, by text
// search, since no Rust parser is available — so an unrelated local
// function or variable named "env" is never mistaken for the standard
// library module.
type rustUsageScanner struct{}

func (rustUsageScanner) supports(path string) bool {
	return filepath.Ext(path) == ".rs"
}

func (rustUsageScanner) scan(_ string, content []byte, _ map[technology.ID]bool) ([]usageHit, []Finding) {
	return rustEnvUsages(content), nil
}

var rustStdEnvImport = regexp.MustCompile(`(?m)^\s*use\s+std::env(::\S*)?\s*;`)

func rustEnvUsages(data []byte) []usageHit {
	prefixes := []string{"std::env::var_os(", "std::env::var("}
	if rustStdEnvImport.Match(data) {
		prefixes = append(prefixes, "env::var_os(", "env::var(")
	}
	return findLiteralCallUsages(data, prefixes)
}
