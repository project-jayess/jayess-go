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

func (l *Lexer) skipWhitespace() {
	for {
		ch := l.current()
		if ch == 0 {
			return
		}
		if unicode.IsSpace(ch) {
			l.advance()
			continue
		}
		if ch == '/' && l.peek() == '/' {
			for ch != 0 && ch != '\n' {
				l.advance()
				ch = l.current()
			}
			continue
		}
		if ch == '/' && l.peek() == '*' {
			l.advance()
			l.advance()
			for {
				ch = l.current()
				if ch == 0 {
					return
				}
				if ch == '*' && l.peek() == '/' {
					l.advance()
					l.advance()
					break
				}
				l.advance()
			}
			continue
		}
		return
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
			l.advance()
			continue
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

func (l *Lexer) peekSecond() rune {
	if l.pos+2 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+2]
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

func (l *Lexer) Snapshot() State {
	return State{Pos: l.pos, Line: l.line, Column: l.column}
}

func (l *Lexer) Restore(state State) {
	l.pos = state.Pos
	l.line = state.Line
	l.column = state.Column
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
	case "class":
		return TokenClass
	case "extends":
		return TokenExtends
	case "extern":
		return TokenExtern
	case "import":
		return TokenImport
	case "native":
		return TokenNative
	case "static":
		return TokenStatic
	case "new":
		return TokenNew
	case "typeof":
		return TokenTypeof
	case "instanceof":
		return TokenInstanceof
	case "is":
		return TokenIs
	case "this":
		return TokenThis
	case "super":
		return TokenSuper
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
	case "do":
		return TokenDo
	case "while":
		return TokenWhile
	case "for":
		return TokenFor
	case "of":
		return TokenOf
	case "in":
		return TokenIn
	case "switch":
		return TokenSwitch
	case "case":
		return TokenCase
	case "default":
		return TokenDefault
	case "break":
		return TokenBreak
	case "continue":
		return TokenContinue
	case "delete":
		return TokenDelete
	case "try":
		return TokenTry
	case "catch":
		return TokenCatch
	case "finally":
		return TokenFinally
	case "throw":
		return TokenThrow
	case "await":
		return TokenAwait
	case "async":
		return TokenAsync
	case "yield":
		return TokenYield
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
