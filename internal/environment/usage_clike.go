package environment

// findLiteralCallUsages scans data (using the shared cLexer's C-style
// comment/string skipping) for every occurrence of any prefix in prefixes
// (e.g. "System.getenv(") at a word boundary, outside comments and
// strings, immediately followed by a single statically knowable string
// literal argument and a closing parenthesis. A call whose argument is
// anything else (a variable, concatenation, a method call, ...) is not
// statically knowable and is skipped.
//
// This single, shared scanner backs several of this package's simplest
// environment-usage forms — Java/Kotlin's System.getenv, C#'s
// Environment.GetEnvironmentVariable, and C/C++'s getenv — since each of
// those is exactly this shape: a fixed call prefix plus one string literal
// argument, over C-like lexical rules. Rust's std::env::var needs its own
// scanner instead, since it additionally has to reason about whether
// std::env was actually imported for the unqualified form (see
// usage_rust.go).
func findLiteralCallUsages(data []byte, prefixes []string) []usageHit {
	s := &cLexer{lexCursor: lexCursor{data: data, line: 1}}
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
		case c == '"' || c == '\'':
			s.skipString(c)
			continue
		}

		if !s.precededByIdentChar() {
			matched := false
			for _, prefix := range prefixes {
				if s.tryMatchLiteral(prefix) {
					if hit, ok := scanLiteralCallArg(s); ok {
						hits = append(hits, hit)
					}
					matched = true
					break
				}
			}
			if matched {
				continue
			}
		}

		s.advance()
	}

	return hits
}

// scanLiteralCallArg is called immediately after consuming a recognized
// prefix ending in "(", and attempts to read a single statically knowable
// string literal argument followed by ")".
func scanLiteralCallArg(s *cLexer) (usageHit, bool) {
	startLine := s.line

	s.skipInlineWhitespace()
	if s.peek() != '"' {
		return usageHit{}, false
	}
	name, ok := s.readQuotedLiteral('"')
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
