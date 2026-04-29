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

type DiagnosticError struct {
	Line    int
	Column  int
	Message string
}

func (e *DiagnosticError) Error() string {
	if e == nil {
		return ""
	}
	if e.Line > 0 {
		return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Message)
	}
	return e.Message
}

func New(input string) *Lexer {
	return &Lexer{input: []rune(input), line: 1, column: 1}
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	startLine := l.line
	startColumn := l.column
	ch := l.current()
	if ch == 0 {
		return Token{Type: TokenEOF, Line: startLine, Column: startColumn}
	}

	switch ch {
	case '`':
		l.advance()
		literal, ok := l.readTemplate()
		if !ok {
			return Token{Type: TokenIllegal, Literal: "unterminated template", Line: startLine, Column: startColumn}
		}
		return Token{Type: TokenTemplate, Literal: literal, Line: startLine, Column: startColumn}
	case '(':
		l.advance()
		return Token{Type: TokenLParen, Literal: "(", Line: startLine, Column: startColumn}
	case ')':
		l.advance()
		return Token{Type: TokenRParen, Literal: ")", Line: startLine, Column: startColumn}
	case '{':
		l.advance()
		return Token{Type: TokenLBrace, Literal: "{", Line: startLine, Column: startColumn}
	case '}':
		l.advance()
		return Token{Type: TokenRBrace, Literal: "}", Line: startLine, Column: startColumn}
	case '[':
		l.advance()
		return Token{Type: TokenLBracket, Literal: "[", Line: startLine, Column: startColumn}
	case ']':
		l.advance()
		return Token{Type: TokenRBracket, Literal: "]", Line: startLine, Column: startColumn}
	case ';':
		l.advance()
		return Token{Type: TokenSemicolon, Literal: ";", Line: startLine, Column: startColumn}
	case '=':
		if l.peek() == '=' && l.peekSecond() == '=' {
			l.advance()
			l.advance()
			l.advance()
			return Token{Type: TokenStrictEq, Literal: "===", Line: startLine, Column: startColumn}
		}
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenEq, Literal: "==", Line: startLine, Column: startColumn}
		}
		if l.peek() == '>' {
			l.advance()
			l.advance()
			return Token{Type: TokenArrow, Literal: "=>", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenAssign, Literal: "=", Line: startLine, Column: startColumn}
	case '!':
		if l.peek() == '=' && l.peekSecond() == '=' {
			l.advance()
			l.advance()
			l.advance()
			return Token{Type: TokenStrictNe, Literal: "!==", Line: startLine, Column: startColumn}
		}
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenNe, Literal: "!=", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenBang, Literal: "!", Line: startLine, Column: startColumn}
	case '?':
		if l.peek() == '?' && l.peekSecond() == '=' {
			l.advance()
			l.advance()
			l.advance()
			return Token{Type: TokenNullishAssign, Literal: "??=", Line: startLine, Column: startColumn}
		}
		if l.peek() == '.' {
			l.advance()
			l.advance()
			return Token{Type: TokenQuestionDot, Literal: "?.", Line: startLine, Column: startColumn}
		}
		if l.peek() == '?' {
			l.advance()
			l.advance()
			return Token{Type: TokenNullish, Literal: "??", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenQuestion, Literal: "?", Line: startLine, Column: startColumn}
	case '&':
		if l.peek() == '&' && l.peekSecond() == '=' {
			l.advance()
			l.advance()
			l.advance()
			return Token{Type: TokenAndAssign, Literal: "&&=", Line: startLine, Column: startColumn}
		}
		if l.peek() == '&' {
			l.advance()
			l.advance()
			return Token{Type: TokenAnd, Literal: "&&", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenBitAnd, Literal: "&", Line: startLine, Column: startColumn}
	case '|':
		if l.peek() == '|' && l.peekSecond() == '=' {
			l.advance()
			l.advance()
			l.advance()
			return Token{Type: TokenOrAssign, Literal: "||=", Line: startLine, Column: startColumn}
		}
		if l.peek() == '|' {
			l.advance()
			l.advance()
			return Token{Type: TokenOr, Literal: "||", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenBitOr, Literal: "|", Line: startLine, Column: startColumn}
	case '^':
		l.advance()
		return Token{Type: TokenBitXor, Literal: "^", Line: startLine, Column: startColumn}
	case '~':
		l.advance()
		return Token{Type: TokenBitNot, Literal: "~", Line: startLine, Column: startColumn}
	case '<':
		if l.peek() == '<' {
			l.advance()
			l.advance()
			return Token{Type: TokenShiftLeft, Literal: "<<", Line: startLine, Column: startColumn}
		}
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenLte, Literal: "<=", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenLt, Literal: "<", Line: startLine, Column: startColumn}
	case '>':
		if l.peek() == '>' && l.peekSecond() == '>' {
			l.advance()
			l.advance()
			l.advance()
			return Token{Type: TokenUnsignedShift, Literal: ">>>", Line: startLine, Column: startColumn}
		}
		if l.peek() == '>' {
			l.advance()
			l.advance()
			return Token{Type: TokenShiftRight, Literal: ">>", Line: startLine, Column: startColumn}
		}
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenGte, Literal: ">=", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenGt, Literal: ">", Line: startLine, Column: startColumn}
	case ',':
		l.advance()
		return Token{Type: TokenComma, Literal: ",", Line: startLine, Column: startColumn}
	case ':':
		l.advance()
		return Token{Type: TokenColon, Literal: ":", Line: startLine, Column: startColumn}
	case '.':
		if l.peek() == '.' && l.peekSecond() == '.' {
			l.advance()
			l.advance()
			l.advance()
			return Token{Type: TokenEllipsis, Literal: "...", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenDot, Literal: ".", Line: startLine, Column: startColumn}
	case '#':
		l.advance()
		return Token{Type: TokenHash, Literal: "#", Line: startLine, Column: startColumn}
	case '+':
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenAddAssign, Literal: "+=", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenPlus, Literal: "+", Line: startLine, Column: startColumn}
	case '-':
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenSubAssign, Literal: "-=", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenMinus, Literal: "-", Line: startLine, Column: startColumn}
	case '*':
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenMulAssign, Literal: "*=", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenStar, Literal: "*", Line: startLine, Column: startColumn}
	case '/':
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenDivAssign, Literal: "/=", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenSlash, Literal: "/", Line: startLine, Column: startColumn}
	case '"':
		l.advance()
		literal, ok := l.readString()
		if !ok {
			return Token{Type: TokenIllegal, Literal: "unterminated string", Line: startLine, Column: startColumn}
		}
		return Token{Type: TokenString, Literal: literal, Line: startLine, Column: startColumn}
	}

	if isIdentifierStart(ch) {
		literal := l.readIdentifier()
		return Token{Type: lookupIdent(literal), Literal: literal, Line: startLine, Column: startColumn}
	}
	if unicode.IsDigit(ch) {
		literal, isBigInt := l.readNumber()
		if isBigInt {
			return Token{Type: TokenBigInt, Literal: literal, Line: startLine, Column: startColumn}
		}
		return Token{Type: TokenNumber, Literal: literal, Line: startLine, Column: startColumn}
	}

	l.advance()
	return Token{Type: TokenIllegal, Literal: string(ch), Line: startLine, Column: startColumn}
}
