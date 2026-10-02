package environment

import (
	"path/filepath"

	"github.com/msiuda/localops/internal/technology"
)

// javaUsageScanner recognizes System.getenv("NAME") in Java source files.
type javaUsageScanner struct{}

func (javaUsageScanner) supports(path string) bool {
	return filepath.Ext(path) == ".java"
}

func (javaUsageScanner) scan(_ string, content []byte, _ map[technology.ID]bool) ([]usageHit, []Finding) {
	return findLiteralCallUsages(content, []string{"System.getenv("}), nil
}
