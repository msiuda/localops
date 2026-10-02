package environment

import (
	"path/filepath"

	"github.com/msiuda/localops/internal/technology"
)

// cppUsageScanner recognizes std::getenv("NAME") and bare getenv("NAME")
// in C++ source/header files. ".c"/".h" are deliberately not claimed here
// — cUsageScanner already handles them, and both scanners recognize the
// identical "getenv(" form, so there is nothing to gain from scanning a
// file twice. A qualifying "std::" prefix does not break the identifier
// boundary before "getenv" (it ends in ':', not an identifier character),
// so the single prefix below matches both the qualified and bare forms.
type cppUsageScanner struct{}

var cppExtensions = map[string]bool{".cpp": true, ".cc": true, ".cxx": true, ".hpp": true, ".hh": true, ".hxx": true}

func (cppUsageScanner) supports(path string) bool {
	return cppExtensions[filepath.Ext(path)]
}

func (cppUsageScanner) scan(_ string, content []byte, _ map[technology.ID]bool) ([]usageHit, []Finding) {
	return findLiteralCallUsages(content, []string{"getenv("}), nil
}
