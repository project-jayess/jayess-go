package lexer

func (l *Lexer) readPunctuation(line, column int) (Token, bool) {
	token := func(typ TokenType, literal string) (Token, bool) {
		for range literal {
			l.advance()
		}
		return Token{Type: typ, Literal: literal, Line: line, Column: column}, true
	}

	switch l.current() {
	case '(':
		return token(TokenLParen, "(")
	case ')':
		return token(TokenRParen, ")")
	case '{':
		return token(TokenLBrace, "{")
	case '}':
		return token(TokenRBrace, "}")
	case '[':
		return token(TokenLBracket, "[")
	case ']':
		return token(TokenRBracket, "]")
	case ';':
		return token(TokenSemicolon, ";")
	case ',':
		return token(TokenComma, ",")
	case ':':
		return token(TokenColon, ":")
	case '#':
		return token(TokenHash, "#")
	case '@':
		return token(TokenAt, "@")
	case '.':
		if l.peek() == '.' && l.peekSecond() == '.' {
			return token(TokenEllipsis, "...")
		}
		return token(TokenDot, ".")
	case '=':
		if l.peek() == '=' && l.peekSecond() == '=' {
			return token(TokenStrictEq, "===")
		}
		if l.peek() == '=' {
			return token(TokenEq, "==")
		}
		if l.peek() == '>' {
			return token(TokenArrow, "=>")
		}
		return token(TokenAssign, "=")
	case '!':
		if l.peek() == '=' && l.peekSecond() == '=' {
			return token(TokenStrictNe, "!==")
		}
		if l.peek() == '=' {
			return token(TokenNe, "!=")
		}
		return token(TokenBang, "!")
	case '?':
		if l.peek() == '?' && l.peekSecond() == '=' {
			return token(TokenNullishAssign, "??=")
		}
		if l.peek() == '?' {
			return token(TokenNullish, "??")
		}
		if l.peek() == '.' {
			return token(TokenQuestionDot, "?.")
		}
		return token(TokenQuestion, "?")
	case '+':
		if l.peek() == '+' {
			return token(TokenIncrement, "++")
		}
		if l.peek() == '=' {
			return token(TokenAddAssign, "+=")
		}
		return token(TokenPlus, "+")
	case '-':
		if l.peek() == '-' {
			return token(TokenDecrement, "--")
		}
		if l.peek() == '=' {
			return token(TokenSubAssign, "-=")
		}
		return token(TokenMinus, "-")
	case '*':
		if l.peek() == '*' && l.peekSecond() == '=' {
			return token(TokenPowAssign, "**=")
		}
		if l.peek() == '*' {
			return token(TokenPower, "**")
		}
		if l.peek() == '=' {
			return token(TokenMulAssign, "*=")
		}
		return token(TokenStar, "*")
	case '/':
		if l.peek() == '=' {
			return token(TokenDivAssign, "/=")
		}
		return token(TokenSlash, "/")
	case '%':
		if l.peek() == '=' {
			return token(TokenModAssign, "%=")
		}
		return token(TokenPercent, "%")
	case '&':
		if l.peek() == '&' && l.peekSecond() == '=' {
			return token(TokenAndAssign, "&&=")
		}
		if l.peek() == '&' {
			return token(TokenAnd, "&&")
		}
		if l.peek() == '=' {
			return token(TokenBitAndAssign, "&=")
		}
		return token(TokenBitAnd, "&")
	case '|':
		if l.peek() == '|' && l.peekSecond() == '=' {
			return token(TokenOrAssign, "||=")
		}
		if l.peek() == '|' {
			return token(TokenOr, "||")
		}
		if l.peek() == '=' {
			return token(TokenBitOrAssign, "|=")
		}
		return token(TokenBitOr, "|")
	case '^':
		if l.peek() == '=' {
			return token(TokenBitXorAssign, "^=")
		}
		return token(TokenBitXor, "^")
	case '~':
		return token(TokenBitNot, "~")
	case '<':
		if l.peek() == '<' && l.peekSecond() == '=' {
			return token(TokenShlAssign, "<<=")
		}
		if l.peek() == '<' {
			return token(TokenShiftLeft, "<<")
		}
		if l.peek() == '=' {
			return token(TokenLte, "<=")
		}
		return token(TokenLt, "<")
	case '>':
		if l.peek() == '>' && l.peekSecond() == '>' && l.peekThird() == '=' {
			return token(TokenUShrAssign, ">>>=")
		}
		if l.peek() == '>' && l.peekSecond() == '>' {
			return token(TokenUnsignedShift, ">>>")
		}
		if l.peek() == '>' && l.peekSecond() == '=' {
			return token(TokenShrAssign, ">>=")
		}
		if l.peek() == '>' {
			return token(TokenShiftRight, ">>")
		}
		if l.peek() == '=' {
			return token(TokenGte, ">=")
		}
		return token(TokenGt, ">")
	default:
		return Token{}, false
	}
}
