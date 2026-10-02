package environment

import (
	"path/filepath"

	"github.com/msiuda/localops/internal/technology"
)

// jsExtensions are the JavaScript/TypeScript source extensions scanned for
// environment-variable usage.
var jsExtensions = map[string]bool{
	".js":  true,
	".jsx": true,
	".ts":  true,
	".tsx": true,
	".mjs": true,
	".cjs": true,
	".mts": true,
	".cts": true,
}

// javascriptUsageScanner recognizes process.env and import.meta.env
// usage in JavaScript/TypeScript source files.
type javascriptUsageScanner struct{}

func (javascriptUsageScanner) supports(path string) bool {
	return jsExtensions[filepath.Ext(path)]
}

func (javascriptUsageScanner) scan(_ string, content []byte, _ map[technology.ID]bool) ([]usageHit, []Finding) {
	return jsEnvUsages(content), nil
}

// viteBuiltinEnvKeys are the fixed keys Vite itself injects into
// `import.meta.env` for every project (https://vite.dev/guide/env-and-mode):
// never user-defined Environment Contract variables, so they must never
// enter the used/declared/satisfied correlation — not reported as
// undeclared, not expected in .env.example. This exclusion is deliberately
// scoped to the import.meta.env forms only: a project's own
// `process.env.DEV` (a different, user-controlled namespace) is an
// ordinary usage and is still recognized normally.
var viteBuiltinEnvKeys = map[string]bool{
	"MODE":     true,
	"BASE_URL": true,
	"PROD":     true,
	"DEV":      true,
	"SSR":      true,
}

// jsEnvUsages recognizes the supported JavaScript/TypeScript forms
// (process.env.FOO, process.env["FOO"]/['FOO'], and the import.meta.env
// equivalents) using a small, purpose-built lexical scanner rather than a
// full JavaScript parser.
//
// It skips // and /* */ comments, and ignores environment-looking text
// inside ordinary string literals; it only inspects a bracket access's
// string content when that bracket immediately follows a recognized
// "process.env" or "import.meta.env" prefix, which is how a statically
// knowable bracket key is told apart from an unrelated string elsewhere in
// the file. A key reached through concatenation, a variable, or any other
// dynamic expression is not statically knowable and is ignored, as is any
// usage inside a template literal's interpolated expression (template
// literals are treated as opaque strings, consistent with this milestone
// not implementing JavaScript template-expression semantics). This never
// fails: unrecognized or exotic syntax is simply not reported, preferring
// false negatives to false positives.
func jsEnvUsages(data []byte) []usageHit {
	s := &jsScanner{cLexer: cLexer{lexCursor: lexCursor{data: data, line: 1}}}
	var hits []usageHit

	for !s.done() {
		c := s.peek()

		switch {
		case c == '/' && s.peekAt(1) == '/':
			s.skipLineComment()
			continue
		case c == '/' && s.peekAt(1) == '*':
			s.skipBlockComment()
			continue
		case c == '\'' || c == '"' || c == '`':
			s.skipString(c)
			continue
		}

		if isIdentStart(c) && !s.precededByIdentChar() {
			isImportMetaEnv := s.tryMatchLiteral("import.meta.env")
			if isImportMetaEnv || s.tryMatchLiteral("process.env") {
				if hit, ok := s.scanEnvAccess(); ok {
					if !isImportMetaEnv || !viteBuiltinEnvKeys[hit.Name] {
						hits = append(hits, hit)
					}
				}
				continue
			}
		}

		s.advance()
	}

	return hits
}

// jsScanner is the JavaScript/TypeScript-specific scanner, built on the
// shared cLexer cursor (JS/TS shares C's comment and string lexical rules)
// plus its own bracket-access handling, which no other supported language
// needs in the same shape.
type jsScanner struct {
	cLexer
}

// scanEnvAccess is called immediately after consuming a recognized
// "process.env" or "import.meta.env" prefix, and attempts to read the
// following static property access: either ".NAME" or a quoted,
// non-dynamic "[\"NAME\"]"/['NAME'].
func (s *jsScanner) scanEnvAccess() (usageHit, bool) {
	startLine := s.line

	switch s.peek() {
	case '.':
		s.advance()
		name := s.readIdent()
		if name == "" || !isValidEnvKey(name) {
			return usageHit{}, false
		}
		return usageHit{Name: name, Line: startLine}, true

	case '[':
		s.advance()
		s.skipInlineWhitespace()
		quote := s.peek()
		if quote != '\'' && quote != '"' {
			return usageHit{}, false
		}
		key, ok := s.readQuotedLiteral(quote)
		if !ok {
			return usageHit{}, false
		}
		s.skipInlineWhitespace()
		if s.peek() != ']' {
			return usageHit{}, false
		}
		s.advance()
		if !isValidEnvKey(key) {
			return usageHit{}, false
		}
		return usageHit{Name: key, Line: startLine}, true
	}

	return usageHit{}, false
}

func isIdentByte(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
