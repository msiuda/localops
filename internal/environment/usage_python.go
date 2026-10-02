package environment

import (
	"path/filepath"
	"regexp"

	"github.com/msiuda/localops/internal/technology"
)

// pythonUsageScanner recognizes os.getenv("NAME"), os.environ["NAME"], and
// os.environ.get("NAME") in Python source files, resolving the base
// identifier against the file's own `import os` (or `import os as alias`)
// statement — conservatively, by text search, since no Python parser is
// available — so an unrelated local module or variable is never mistaken
// for the standard library's os module.
type pythonUsageScanner struct{}

func (pythonUsageScanner) supports(path string) bool {
	return filepath.Ext(path) == ".py"
}

func (pythonUsageScanner) scan(_ string, content []byte, _ map[technology.ID]bool) ([]usageHit, []Finding) {
	return pythonEnvUsages(content), nil
}

var pythonOSImport = regexp.MustCompile(`(?m)^\s*import\s+os(?:\s+as\s+(\w+))?\s*$`)

// pythonEnvUsages skips # comments and ordinary (including triple-quoted)
// string literals, then recognizes each bound os alias's .getenv(...),
// .environ[...], and .environ.get(...) forms.
func pythonEnvUsages(data []byte) []usageHit {
	aliases := osAliases(data)
	if len(aliases) == 0 {
		return nil
	}

	s := &hashLexer{lexCursor: lexCursor{data: data, line: 1}}
	var hits []usageHit

	for !s.done() {
		c := s.peek()

		switch {
		case c == '#':
			s.skipLineComment()
			continue
		case isPythonTripleQuote(s):
			skipPythonTripleQuote(s)
			continue
		case c == '\'' || c == '"':
			s.skipString(c)
			continue
		}

		if isIdentStart(c) && !s.precededByIdentChar() {
			matchedAlias := ""
			for _, alias := range aliases {
				if s.tryMatchLiteral(alias + ".") {
					matchedAlias = alias
					break
				}
			}
			if matchedAlias != "" {
				if hit, ok := scanPythonOSAccess(s); ok {
					hits = append(hits, hit)
				}
				continue
			}
		}

		s.advance()
	}

	return hits
}

// osAliases returns the local identifier(s) `import os` (optionally
// `as alias`) binds in data.
func osAliases(data []byte) []string {
	var aliases []string
	for _, m := range pythonOSImport.FindAllSubmatch(data, -1) {
		if len(m) > 1 && len(m[1]) > 0 {
			aliases = append(aliases, string(m[1]))
		} else {
			aliases = append(aliases, "os")
		}
	}
	return aliases
}

// scanPythonOSAccess is called immediately after consuming "<alias>.", and
// attempts to read "getenv(\"NAME\")", "environ[\"NAME\"]", or
// "environ.get(\"NAME\")".
func scanPythonOSAccess(s *hashLexer) (usageHit, bool) {
	startLine := s.line

	if s.tryMatchLiteral("getenv(") {
		return scanLiteralCallArgHash(s)
	}

	if s.tryMatchLiteral("environ") {
		switch {
		case s.peek() == '[':
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

		case s.tryMatchLiteral(".get("):
			return scanLiteralCallArgHash(s)
		}
	}

	return usageHit{}, false
}

// scanLiteralCallArgHash mirrors scanLiteralCallArg for hashLexer-based
// scanners (Python, Ruby), which cannot share cLexer's method set.
func scanLiteralCallArgHash(s *hashLexer) (usageHit, bool) {
	startLine := s.line

	s.skipInlineWhitespace()
	quote := s.peek()
	if quote != '"' && quote != '\'' {
		return usageHit{}, false
	}
	name, ok := s.readQuotedLiteral(quote)
	if !ok {
		return usageHit{}, false
	}
	s.skipInlineWhitespace()
	if s.peek() != ')' {
		return usageHit{}, false
	}
	s.advance()
	if !isValidEnvKey(name) {
		return usageHit{}, false
	}
	return usageHit{Name: name, Line: startLine}, true
}

func isPythonTripleQuote(s *hashLexer) bool {
	c := s.peek()
	return (c == '\'' || c == '"') && s.peekAt(1) == c && s.peekAt(2) == c
}

func skipPythonTripleQuote(s *hashLexer) {
	quote := s.peek()
	s.advance()
	s.advance()
	s.advance()
	for !s.done() {
		if s.peek() == quote && s.peekAt(1) == quote && s.peekAt(2) == quote {
			s.advance()
			s.advance()
			s.advance()
			return
		}
		s.advance()
	}
}
