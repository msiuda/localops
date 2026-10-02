package environment

import (
	"path/filepath"

	"github.com/msiuda/localops/internal/technology"
)

// csharpUsageScanner recognizes Environment.GetEnvironmentVariable("NAME")
// in C# source files. The fully-qualified System.Environment... form is
// also recognized: a preceding "System." does not break the identifier
// boundary before "Environment" (it is a '.', not an identifier
// character), so the bare prefix below matches both forms.
type csharpUsageScanner struct{}

func (csharpUsageScanner) supports(path string) bool {
	return filepath.Ext(path) == ".cs"
}

func (csharpUsageScanner) scan(_ string, content []byte, _ map[technology.ID]bool) ([]usageHit, []Finding) {
	return findLiteralCallUsages(content, []string{"Environment.GetEnvironmentVariable("}), nil
}
