package environment

import (
	"path/filepath"

	"github.com/msiuda/localops/internal/technology"
)

// kotlinUsageScanner recognizes System.getenv("NAME") — the same JVM
// stdlib call Kotlin shares with Java — in Kotlin source files.
type kotlinUsageScanner struct{}

var kotlinExtensions = map[string]bool{".kt": true, ".kts": true}

func (kotlinUsageScanner) supports(path string) bool {
	return kotlinExtensions[filepath.Ext(path)]
}

func (kotlinUsageScanner) scan(_ string, content []byte, _ map[technology.ID]bool) ([]usageHit, []Finding) {
	return findLiteralCallUsages(content, []string{"System.getenv("}), nil
}
