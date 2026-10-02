package environment

import (
	"path/filepath"

	"github.com/msiuda/localops/internal/technology"
)

// phpUsageScanner recognizes getenv("NAME"), $_ENV["NAME"]/['NAME'], and
// $_SERVER["NAME"]/['NAME'] in PHP source files unconditionally, and
// Laravel's env("NAME") only when Laravel was actually detected for this
// project (see internal/technology) — env() is an ordinary-looking
// function name outside a Laravel context, so recognizing it
// unconditionally would be exactly the kind of false positive this
// package's scanners are designed to avoid.
type phpUsageScanner struct{}

func (phpUsageScanner) supports(path string) bool {
	return filepath.Ext(path) == ".php"
}

func (phpUsageScanner) scan(_ string, content []byte, detected map[technology.ID]bool) ([]usageHit, []Finding) {
	return phpEnvUsages(content, detected[technology.IDLaravel]), nil
}

// phpLexer adds PHP's comment syntax (// and # line comments, /* */ block
// comments) to lexCursor.
type phpLexer struct {
	lexCursor
}

func (s *phpLexer) skipSlashLineComment() {
	s.advance()
	s.advance()
	for !s.done() && s.peek() != '\n' {
		s.advance()
	}
}

func (s *phpLexer) skipHashLineComment() {
	s.advance()
	for !s.done() && s.peek() != '\n' {
		s.advance()
	}
}

func (s *phpLexer) skipBlockComment() {
	s.advance()
	s.advance()
	for !s.done() {
		if s.peek() == '*' && s.peekAt(1) == '/' {
			s.advance()
			s.advance()
			return
		}
		s.advance()
	}
}

func phpEnvUsages(data []byte, laravelDetected bool) []usageHit {
	s := &phpLexer{lexCursor: lexCursor{data: data, line: 1}}
	var hits []usageHit

	for !s.done() {
		c := s.peek()

		switch {
		case c == '/' && s.peekAt(1) == '/':
			s.skipSlashLineComment()
			continue
		case c == '#':
			s.skipHashLineComment()
			continue
		case c == '/' && s.peekAt(1) == '*':
			s.skipBlockComment()
			continue
		case c == '\'' || c == '"':
			s.skipString(c)
			continue
		}

		if (isIdentStart(c) || c == '$') && !s.precededByIdentChar() {
			switch {
			case s.tryMatchLiteral("getenv("):
				if hit, ok := scanLiteralCallArgPHP(s); ok {
					hits = append(hits, hit)
				}
				continue
			case s.tryMatchLiteral("$_ENV"):
				if hit, ok := scanPHPSuperglobalAccess(s); ok {
					hits = append(hits, hit)
				}
				continue
			case s.tryMatchLiteral("$_SERVER"):
				if hit, ok := scanPHPSuperglobalAccess(s); ok {
					hits = append(hits, hit)
				}
				continue
			case laravelDetected && s.tryMatchLiteral("env("):
				if hit, ok := scanLiteralCallArgPHP(s); ok {
					hits = append(hits, hit)
				}
				continue
			}
		}

		s.advance()
	}

	return hits
}

// scanPHPSuperglobalAccess is called immediately after consuming "$_ENV"
// or "$_SERVER", and attempts to read "[\"NAME\"]"/['NAME'].
func scanPHPSuperglobalAccess(s *phpLexer) (usageHit, bool) {
	startLine := s.line

	if s.peek() != '[' {
		return usageHit{}, false
	}
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

// scanLiteralCallArgPHP mirrors scanLiteralCallArg for phpLexer, which
// cannot share cLexer's method set.
func scanLiteralCallArgPHP(s *phpLexer) (usageHit, bool) {
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
