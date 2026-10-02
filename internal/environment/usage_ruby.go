package environment

import (
	"path/filepath"

	"github.com/msiuda/localops/internal/technology"
)

// rubyUsageScanner recognizes ENV["NAME"], ENV['NAME'], ENV.fetch("NAME"),
// and ENV.fetch('NAME') in Ruby source files. ENV is a Ruby global
// constant, always available without an import, so (unlike Python's os or
// Rust's std::env) no import/alias resolution is needed here.
//
// Ruby's other string forms (%q{}, heredocs, and so on) are not
// recognized; only ordinary single/double-quoted literals are, consistent
// with this scanner's conservative, false-negatives-over-false-positives
// design.
type rubyUsageScanner struct{}

func (rubyUsageScanner) supports(path string) bool {
	return filepath.Ext(path) == ".rb"
}

func (rubyUsageScanner) scan(_ string, content []byte, _ map[technology.ID]bool) ([]usageHit, []Finding) {
	return rubyEnvUsages(content), nil
}

func rubyEnvUsages(data []byte) []usageHit {
	s := &hashLexer{lexCursor: lexCursor{data: data, line: 1}}
	var hits []usageHit

	for !s.done() {
		c := s.peek()

		switch {
		case c == '#':
			s.skipLineComment()
			continue
		case c == '\'' || c == '"':
			s.skipString(c)
			continue
		}

		if isIdentStart(c) && !s.precededByIdentChar() {
			if s.tryMatchLiteral("ENV") {
				if hit, ok := scanRubyENVAccess(s); ok {
					hits = append(hits, hit)
				}
				continue
			}
		}

		s.advance()
	}

	return hits
}

// scanRubyENVAccess is called immediately after consuming "ENV", and
// attempts to read "[\"NAME\"]"/['NAME'] or ".fetch(\"NAME\")"/('NAME').
func scanRubyENVAccess(s *hashLexer) (usageHit, bool) {
	startLine := s.line

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

	case s.tryMatchLiteral(".fetch("):
		return scanLiteralCallArgHash(s)
	}

	return usageHit{}, false
}
