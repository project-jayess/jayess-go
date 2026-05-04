package lexer

import "unicode"

func (l *Lexer) skipIgnored() *Token {
	for {
		ch := l.current()
		switch {
		case ch == 0:
			return nil
		case l.pos == 0 && ch == '#' && l.peek() == '!':
			l.skipLineComment()
		case unicode.IsSpace(ch):
			l.advance()
		case ch == '/' && l.peek() == '/':
			l.skipLineComment()
		case ch == '/' && l.peek() == '*':
			line, column := l.line, l.column
			if !l.skipBlockComment() {
				return &Token{Type: TokenIllegal, Literal: "unterminated block comment", Line: line, Column: column}
			}
		default:
			return nil
		}
	}
}

func (l *Lexer) skipLineComment() {
	for l.current() != 0 && l.current() != '\n' {
		l.advance()
	}
}

func (l *Lexer) skipBlockComment() bool {
	l.advance()
	l.advance()
	for {
		if l.current() == 0 {
			return false
		}
		if l.current() == '*' && l.peek() == '/' {
			l.advance()
			l.advance()
			return true
		}
		l.advance()
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.pos
	for isIdentifierPart(l.current()) {
		l.advance()
	}
	return string(l.input[start:l.pos])
}

func (l *Lexer) readNumber() (string, bool) {
	start := l.pos
	dotSeen := false
	for {
		ch := l.current()
		switch {
		case unicode.IsDigit(ch):
			l.advance()
		case ch == '.' && !dotSeen:
			dotSeen = true
			l.advance()
		case ch == 'n' && !dotSeen:
			literal := string(l.input[start:l.pos])
			l.advance()
			return literal, true
		default:
			return string(l.input[start:l.pos]), false
		}
	}
}

func (l *Lexer) readString(quote rune) (string, bool) {
	start := l.pos
	for {
		ch := l.current()
		if ch == 0 || ch == '\n' {
			return "", false
		}
		if ch == '\\' {
			l.advance()
			if l.current() == 0 || l.current() == '\n' {
				return "", false
			}
			l.advance()
			continue
		}
		if ch == quote {
			literal := string(l.input[start:l.pos])
			l.advance()
			return literal, true
		}
		l.advance()
	}
}

func (l *Lexer) readTemplate() (string, bool) {
	start := l.pos
	depth := 0
	for {
		ch := l.current()
		if ch == 0 {
			return "", false
		}
		if ch == '`' && depth == 0 {
			literal := string(l.input[start:l.pos])
			l.advance()
			return literal, true
		}
		if ch == '$' && l.peek() == '{' {
			depth++
			l.advance()
			l.advance()
			continue
		}
		if ch == '}' && depth > 0 {
			depth--
		}
		l.advance()
	}
}
