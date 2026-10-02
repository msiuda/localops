package environment

// lexCursor is the shared, comment-syntax-agnostic cursor machinery reused
// by every lexical environment-usage scanner in this package: byte-level
// position/line tracking, string-literal skipping/reading, literal
// substring matching, and identifier-boundary checks. Each language's
// lexer embeds lexCursor and adds only its own comment syntax (C-style //
// and /* */ for cLexer; #-style for hashLexer) — that genuinely is the
// only lexical rule that differs across the languages sharing a lexer
// here. Python's and Ruby's own scanners build on hashLexer directly.
type lexCursor struct {
	data []byte
	pos  int
	line int
}

func (s *lexCursor) done() bool {
	return s.pos >= len(s.data)
}

func (s *lexCursor) peek() byte {
	if s.done() {
		return 0
	}
	return s.data[s.pos]
}

func (s *lexCursor) peekAt(offset int) byte {
	idx := s.pos + offset
	if idx < 0 || idx >= len(s.data) {
		return 0
	}
	return s.data[idx]
}

func (s *lexCursor) advance() byte {
	c := s.data[s.pos]
	s.pos++
	if c == '\n' {
		s.line++
	}
	return c
}

// skipString consumes a string literal starting at the current quote
// character, respecting backslash escapes.
func (s *lexCursor) skipString(quote byte) {
	s.advance()
	for !s.done() {
		c := s.peek()
		if c == '\\' {
			s.advance()
			if !s.done() {
				s.advance()
			}
			continue
		}
		if c == quote {
			s.advance()
			return
		}
		s.advance()
	}
}

// readQuotedLiteral reads a string literal's full content, always
// consuming through its closing quote (even when the content turns out
// not to be statically knowable) so the cursor's quote-nesting stays
// balanced for the rest of the file. ok is false when the literal
// contains an escape sequence, an embedded newline, or is left
// unterminated by EOF.
func (s *lexCursor) readQuotedLiteral(quote byte) (string, bool) {
	s.advance()
	var buf []byte
	dynamic := false
	for !s.done() {
		c := s.peek()
		if c == '\\' {
			dynamic = true
			s.advance()
			if !s.done() {
				s.advance()
			}
			continue
		}
		if c == quote {
			s.advance()
			if dynamic {
				return "", false
			}
			return string(buf), true
		}
		if c == '\n' {
			dynamic = true
		}
		buf = append(buf, c)
		s.advance()
	}
	return "", false
}

func (s *lexCursor) skipInlineWhitespace() {
	for !s.done() {
		switch s.peek() {
		case ' ', '\t', '\r', '\n':
			s.advance()
		default:
			return
		}
	}
}

// tryMatchLiteral reports whether lit occurs exactly at the current
// position, consuming it if so. It does not itself check word boundaries;
// callers are responsible for that (via precededByIdentChar).
func (s *lexCursor) tryMatchLiteral(lit string) bool {
	if s.pos+len(lit) > len(s.data) {
		return false
	}
	if string(s.data[s.pos:s.pos+len(lit)]) != lit {
		return false
	}
	for range lit {
		s.advance()
	}
	return true
}

func (s *lexCursor) precededByIdentChar() bool {
	return s.pos > 0 && isIdentByte(s.data[s.pos-1])
}

func (s *lexCursor) readIdent() string {
	start := s.pos
	for !s.done() && isIdentByte(s.peek()) {
		s.advance()
	}
	return string(s.data[start:s.pos])
}

// cLexer adds C-style // and /* */ comment skipping to lexCursor. It backs
// JavaScript/TypeScript's own scanner and the shared generic literal-call
// scanner used by Java, Kotlin, C#, C, and C++.
type cLexer struct {
	lexCursor
}

func (s *cLexer) skipLineComment() {
	s.advance()
	s.advance()
	for !s.done() && s.peek() != '\n' {
		s.advance()
	}
}

func (s *cLexer) skipBlockComment() {
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

// hashLexer adds #-style line comment skipping (Python, Ruby) to
// lexCursor. Neither language has a block-comment form this package needs
// to recognize for environment-usage scanning.
type hashLexer struct {
	lexCursor
}

func (s *hashLexer) skipLineComment() {
	s.advance()
	for !s.done() && s.peek() != '\n' {
		s.advance()
	}
}
