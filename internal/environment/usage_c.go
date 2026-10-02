package environment

import (
	"path/filepath"

	"github.com/msiuda/localops/internal/technology"
)

// cUsageScanner recognizes getenv("NAME") in C source/header files.
type cUsageScanner struct{}

var cExtensions = map[string]bool{".c": true, ".h": true}

func (cUsageScanner) supports(path string) bool {
	return cExtensions[filepath.Ext(path)]
}

func (cUsageScanner) scan(_ string, content []byte, _ map[technology.ID]bool) ([]usageHit, []Finding) {
	return findLiteralCallUsages(content, []string{"getenv("}), nil
}
