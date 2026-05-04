package lexer

import (
	"fmt"
	"unicode"
)

type Lexer struct {
	input  []rune
	pos    int
	line   int
	column int
}

type State struct {
	Pos    int
	Line   int
	Column int
}

func New(input string) *Lexer {
	return &Lexer{input: []rune(input), line: 1, column: 1}
}

func (l *Lexer) NextToken() Token {
	if illegal := l.skipIgnored(); illegal != nil {
		return *illegal
	}

	line, column := l.line, l.column
	ch := l.current()
	if ch == 0 {
		return Token{Type: TokenEOF, Line: line, Column: column}
	}

	if isIdentifierStart(ch) {
		literal := l.readIdentifier()
		return Token{Type: lookupIdent(literal), Literal: literal, Line: line, Column: column}
	}
	if unicode.IsDigit(ch) {
		literal, big := l.readNumber()
		if big {
			return Token{Type: TokenBigInt, Literal: literal, Line: line, Column: column}
		}
		return Token{Type: TokenNumber, Literal: literal, Line: line, Column: column}
	}

	switch ch {
	case '"', '\'':
		quote := ch
		l.advance()
		literal, ok := l.readString(quote)
		if !ok {
			return Token{Type: TokenIllegal, Literal: "unterminated string", Line: line, Column: column}
		}
		return Token{Type: TokenString, Literal: literal, Line: line, Column: column}
	case '`':
		l.advance()
		literal, ok := l.readTemplate()
		if !ok {
			return Token{Type: TokenIllegal, Literal: "unterminated template", Line: line, Column: column}
		}
		return Token{Type: TokenTemplate, Literal: literal, Line: line, Column: column}
	}

	if token, ok := l.readPunctuation(line, column); ok {
		return token
	}

	l.advance()
	return Token{Type: TokenIllegal, Literal: fmt.Sprintf("unexpected character %q", ch), Line: line, Column: column}
}

func (l *Lexer) Snapshot() State {
	return State{Pos: l.pos, Line: l.line, Column: l.column}
}

func (l *Lexer) Restore(state State) {
	l.pos = state.Pos
	l.line = state.Line
	l.column = state.Column
}
