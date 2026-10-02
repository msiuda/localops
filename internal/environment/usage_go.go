package environment

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"

	"github.com/msiuda/localops/internal/technology"
)

// goUsageScanner recognizes os.Getenv("NAME") and os.LookupEnv("NAME")
// calls in Go source files.
type goUsageScanner struct{}

func (goUsageScanner) supports(path string) bool {
	return filepath.Ext(path) == ".go"
}

func (goUsageScanner) scan(path string, content []byte, _ map[technology.ID]bool) ([]usageHit, []Finding) {
	hits, err := goEnvUsages(content)
	if err != nil {
		return nil, []Finding{{Source: path, Detail: "could not be scanned"}}
	}
	return hits, nil
}

// goEnvUsages uses go/parser and go/ast — the standard library's own Go
// parser — to recognize os.Getenv("NAME") and os.LookupEnv("NAME") calls
// where NAME is a string literal. This is AST-based static recognition of
// those two supported forms, not a general semantic analysis: it resolves
// the call's base identifier only against the file's own import
// declarations (plain "os" import or an explicit single-level alias such
// as `stdos "os"`), without type-checking the package or its
// dependencies. A dot-import of "os" is not recognized.
func goEnvUsages(data []byte) ([]usageHit, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", data, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	osNames := importedPackageNames(file, "os")
	if len(osNames) == 0 {
		return nil, nil
	}

	var hits []usageHit
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || !osNames[ident.Name] {
			return true
		}
		if sel.Sel.Name != "Getenv" && sel.Sel.Name != "LookupEnv" {
			return true
		}
		if len(call.Args) == 0 {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		name, err := strconv.Unquote(lit.Value)
		if err != nil || !isValidEnvKey(name) {
			return true
		}
		hits = append(hits, usageHit{Name: name, Line: fset.Position(call.Pos()).Line})
		return true
	})

	return hits, nil
}

// importedPackageNames returns the local identifier(s) that importPath is
// bound to in file: the package's own name for a plain import, or the
// explicit alias for `alias "importPath"`. A dot-import ("." alias) and a
// blank import ("_") are deliberately not included, since neither binds
// importPath to a usable selector base.
func importedPackageNames(file *ast.File, importPath string) map[string]bool {
	names := make(map[string]bool)
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil || path != importPath {
			continue
		}
		if imp.Name != nil {
			if imp.Name.Name != "_" && imp.Name.Name != "." {
				names[imp.Name.Name] = true
			}
			continue
		}
		names[filepath.Base(importPath)] = true
	}
	return names
}
