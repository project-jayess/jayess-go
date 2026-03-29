package lexer

import "unicode"

type Lexer struct {
	input  []rune
	pos    int
	line   int
	column int
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
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenEq, Literal: "==", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenAssign, Literal: "=", Line: startLine, Column: startColumn}
	case '!':
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenNe, Literal: "!=", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenBang, Literal: "!", Line: startLine, Column: startColumn}
	case '&':
		if l.peek() == '&' {
			l.advance()
			l.advance()
			return Token{Type: TokenAnd, Literal: "&&", Line: startLine, Column: startColumn}
		}
	case '|':
		if l.peek() == '|' {
			l.advance()
			l.advance()
			return Token{Type: TokenOr, Literal: "||", Line: startLine, Column: startColumn}
		}
	case '<':
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenLte, Literal: "<=", Line: startLine, Column: startColumn}
		}
		l.advance()
		return Token{Type: TokenLt, Literal: "<", Line: startLine, Column: startColumn}
	case '>':
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
		l.advance()
		return Token{Type: TokenDot, Literal: ".", Line: startLine, Column: startColumn}
	case '+':
		l.advance()
		return Token{Type: TokenPlus, Literal: "+", Line: startLine, Column: startColumn}
	case '-':
		l.advance()
		return Token{Type: TokenMinus, Literal: "-", Line: startLine, Column: startColumn}
	case '*':
		l.advance()
		return Token{Type: TokenStar, Literal: "*", Line: startLine, Column: startColumn}
	case '/':
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
		literal := l.readNumber()
		return Token{Type: TokenNumber, Literal: literal, Line: startLine, Column: startColumn}
	}

	l.advance()
	return Token{Type: TokenIllegal, Literal: string(ch), Line: startLine, Column: startColumn}
}

func (l *Lexer) skipWhitespace() {
	for {
		ch := l.current()
		if ch == 0 || !unicode.IsSpace(ch) {
			return
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

func (l *Lexer) readNumber() string {
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
		default:
			return string(l.input[start:l.pos])
		}
	}
}

func (l *Lexer) readString() (string, bool) {
	start := l.pos
	for {
		ch := l.current()
		if ch == 0 || ch == '\n' {
			return "", false
		}
		if ch == '"' {
			literal := string(l.input[start:l.pos])
			l.advance()
			return literal, true
		}
		l.advance()
	}
}

func (l *Lexer) current() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peek() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

func (l *Lexer) advance() {
	if l.pos >= len(l.input) {
		return
	}
	if l.input[l.pos] == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	l.pos++
}

func isIdentifierStart(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch)
}

func isIdentifierPart(ch rune) bool {
	return isIdentifierStart(ch) || unicode.IsDigit(ch)
}

func lookupIdent(literal string) TokenType {
	switch literal {
	case "function":
		return TokenFunction
	case "extern":
		return TokenExtern
	case "import":
		return TokenImport
	case "native":
		return TokenNative
	case "var":
		return TokenVar
	case "let":
		return TokenLet
	case "const":
		return TokenConst
	case "private":
		return TokenPrivate
	case "public":
		return TokenPublic
	case "return":
		return TokenReturn
	case "if":
		return TokenIf
	case "else":
		return TokenElse
	case "while":
		return TokenWhile
	case "for":
		return TokenFor
	case "break":
		return TokenBreak
	case "continue":
		return TokenContinue
	case "true":
		return TokenTrue
	case "false":
		return TokenFalse
	case "null":
		return TokenNull
	case "undefined":
		return TokenUndefined
	default:
		return TokenIdent
	}
}
