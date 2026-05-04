package lexer

import "testing"

func TestNextTokenScansCoreLexicalFeatures(t *testing.T) {
	input := "// lead\nvar $value = 12n; /* block */\nconst label = 'kimchi';\n"
	l := New(input)

	tests := []Token{
		{Type: TokenVar, Literal: "var", Line: 2, Column: 1},
		{Type: TokenIdent, Literal: "$value", Line: 2, Column: 5},
		{Type: TokenAssign, Literal: "=", Line: 2, Column: 12},
		{Type: TokenBigInt, Literal: "12", Line: 2, Column: 14},
		{Type: TokenSemicolon, Literal: ";", Line: 2, Column: 17},
		{Type: TokenConst, Literal: "const", Line: 3, Column: 1},
		{Type: TokenIdent, Literal: "label", Line: 3, Column: 7},
		{Type: TokenAssign, Literal: "=", Line: 3, Column: 13},
		{Type: TokenString, Literal: "kimchi", Line: 3, Column: 15},
		{Type: TokenSemicolon, Literal: ";", Line: 3, Column: 23},
		{Type: TokenEOF, Line: 4, Column: 1},
	}

	assertTokens(t, l, tests)
}

func TestNextTokenScansKeywordsAndOperators(t *testing.T) {
	input := "if (true && false || null ?? undefined) return value?.name === other !== 1;"
	l := New(input)

	tests := []Token{
		{Type: TokenIf, Literal: "if", Line: 1, Column: 1},
		{Type: TokenLParen, Literal: "(", Line: 1, Column: 4},
		{Type: TokenTrue, Literal: "true", Line: 1, Column: 5},
		{Type: TokenAnd, Literal: "&&", Line: 1, Column: 10},
		{Type: TokenFalse, Literal: "false", Line: 1, Column: 13},
		{Type: TokenOr, Literal: "||", Line: 1, Column: 19},
		{Type: TokenNull, Literal: "null", Line: 1, Column: 22},
		{Type: TokenNullish, Literal: "??", Line: 1, Column: 27},
		{Type: TokenUndefined, Literal: "undefined", Line: 1, Column: 30},
		{Type: TokenRParen, Literal: ")", Line: 1, Column: 39},
		{Type: TokenReturn, Literal: "return", Line: 1, Column: 41},
		{Type: TokenIdent, Literal: "value", Line: 1, Column: 48},
		{Type: TokenQuestionDot, Literal: "?.", Line: 1, Column: 53},
		{Type: TokenIdent, Literal: "name", Line: 1, Column: 55},
		{Type: TokenStrictEq, Literal: "===", Line: 1, Column: 60},
		{Type: TokenIdent, Literal: "other", Line: 1, Column: 64},
		{Type: TokenStrictNe, Literal: "!==", Line: 1, Column: 70},
		{Type: TokenNumber, Literal: "1", Line: 1, Column: 74},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 75},
		{Type: TokenEOF, Line: 1, Column: 76},
	}

	assertTokens(t, l, tests)
}

func TestNextTokenScansWithKeyword(t *testing.T) {
	l := New("with")
	tests := []Token{
		{Type: TokenWith, Literal: "with", Line: 1, Column: 1},
		{Type: TokenEOF, Line: 1, Column: 5},
	}
	assertTokens(t, l, tests)
}

func TestNextTokenScansEnumKeyword(t *testing.T) {
	l := New("enum")
	tests := []Token{
		{Type: TokenEnum, Literal: "enum", Line: 1, Column: 1},
		{Type: TokenEOF, Line: 1, Column: 5},
	}
	assertTokens(t, l, tests)
}

func TestUnterminatedBlockCommentIsIllegalToken(t *testing.T) {
	token := New("/* missing").NextToken()
	if token.Type != TokenIllegal || token.Literal != "unterminated block comment" {
		t.Fatalf("expected unterminated block comment diagnostic token, got %#v", token)
	}
}

func TestUnterminatedStringIsIllegalToken(t *testing.T) {
	token := New(`"missing`).NextToken()
	if token.Type != TokenIllegal || token.Literal != "unterminated string" {
		t.Fatalf("expected unterminated string diagnostic token, got %#v", token)
	}
}

func TestUnterminatedStringBeforeLineBreakIsIllegalToken(t *testing.T) {
	token := New("'missing\nnext").NextToken()
	if token.Type != TokenIllegal || token.Literal != "unterminated string" {
		t.Fatalf("expected unterminated string diagnostic token, got %#v", token)
	}
}

func TestUnterminatedTemplateIsIllegalToken(t *testing.T) {
	token := New("`missing ${value}").NextToken()
	if token.Type != TokenIllegal || token.Literal != "unterminated template" {
		t.Fatalf("expected unterminated template diagnostic token, got %#v", token)
	}
}

func TestUnexpectedCharacterIsIllegalToken(t *testing.T) {
	token := New("\\").NextToken()
	if token.Type != TokenIllegal || token.Literal != `unexpected character '\\'` {
		t.Fatalf("expected unexpected character diagnostic token, got %#v", token)
	}
}

func TestLexerReadsDecoratorPunctuation(t *testing.T) {
	token := New("@").NextToken()
	if token.Type != TokenAt || token.Literal != "@" {
		t.Fatalf("expected @ token, got %#v", token)
	}
}

func TestNextTokenScansModuloOperators(t *testing.T) {
	l := New("value % 2; value %= 3;")
	tests := []Token{
		{Type: TokenIdent, Literal: "value", Line: 1, Column: 1},
		{Type: TokenPercent, Literal: "%", Line: 1, Column: 7},
		{Type: TokenNumber, Literal: "2", Line: 1, Column: 9},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 10},
		{Type: TokenIdent, Literal: "value", Line: 1, Column: 12},
		{Type: TokenModAssign, Literal: "%=", Line: 1, Column: 18},
		{Type: TokenNumber, Literal: "3", Line: 1, Column: 21},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 22},
		{Type: TokenEOF, Line: 1, Column: 23},
	}
	assertTokens(t, l, tests)
}

func TestNextTokenScansBitwiseAssignmentOperators(t *testing.T) {
	l := New("a &= b; a |= b; a ^= b; a <<= b; a >>= b; a >>>= b;")
	tests := []Token{
		{Type: TokenIdent, Literal: "a", Line: 1, Column: 1},
		{Type: TokenBitAndAssign, Literal: "&=", Line: 1, Column: 3},
		{Type: TokenIdent, Literal: "b", Line: 1, Column: 6},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 7},
		{Type: TokenIdent, Literal: "a", Line: 1, Column: 9},
		{Type: TokenBitOrAssign, Literal: "|=", Line: 1, Column: 11},
		{Type: TokenIdent, Literal: "b", Line: 1, Column: 14},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 15},
		{Type: TokenIdent, Literal: "a", Line: 1, Column: 17},
		{Type: TokenBitXorAssign, Literal: "^=", Line: 1, Column: 19},
		{Type: TokenIdent, Literal: "b", Line: 1, Column: 22},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 23},
		{Type: TokenIdent, Literal: "a", Line: 1, Column: 25},
		{Type: TokenShlAssign, Literal: "<<=", Line: 1, Column: 27},
		{Type: TokenIdent, Literal: "b", Line: 1, Column: 31},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 32},
		{Type: TokenIdent, Literal: "a", Line: 1, Column: 34},
		{Type: TokenShrAssign, Literal: ">>=", Line: 1, Column: 36},
		{Type: TokenIdent, Literal: "b", Line: 1, Column: 40},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 41},
		{Type: TokenIdent, Literal: "a", Line: 1, Column: 43},
		{Type: TokenUShrAssign, Literal: ">>>=", Line: 1, Column: 45},
		{Type: TokenIdent, Literal: "b", Line: 1, Column: 50},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 51},
		{Type: TokenEOF, Line: 1, Column: 52},
	}
	assertTokens(t, l, tests)
}

func TestNextTokenScansUpdateOperators(t *testing.T) {
	l := New("count++; --count;")
	tests := []Token{
		{Type: TokenIdent, Literal: "count", Line: 1, Column: 1},
		{Type: TokenIncrement, Literal: "++", Line: 1, Column: 6},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 8},
		{Type: TokenDecrement, Literal: "--", Line: 1, Column: 10},
		{Type: TokenIdent, Literal: "count", Line: 1, Column: 12},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 17},
		{Type: TokenEOF, Line: 1, Column: 18},
	}
	assertTokens(t, l, tests)
}

func TestNextTokenScansExponentiationOperators(t *testing.T) {
	l := New("value ** 2; value **= 3;")
	tests := []Token{
		{Type: TokenIdent, Literal: "value", Line: 1, Column: 1},
		{Type: TokenPower, Literal: "**", Line: 1, Column: 7},
		{Type: TokenNumber, Literal: "2", Line: 1, Column: 10},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 11},
		{Type: TokenIdent, Literal: "value", Line: 1, Column: 13},
		{Type: TokenPowAssign, Literal: "**=", Line: 1, Column: 19},
		{Type: TokenNumber, Literal: "3", Line: 1, Column: 23},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 24},
		{Type: TokenEOF, Line: 1, Column: 25},
	}
	assertTokens(t, l, tests)
}

func TestNextTokenScansVoidKeyword(t *testing.T) {
	l := New("void value;")
	tests := []Token{
		{Type: TokenVoid, Literal: "void", Line: 1, Column: 1},
		{Type: TokenIdent, Literal: "value", Line: 1, Column: 6},
		{Type: TokenSemicolon, Literal: ";", Line: 1, Column: 11},
		{Type: TokenEOF, Line: 1, Column: 12},
	}
	assertTokens(t, l, tests)
}

func TestLexerSnapshotRestore(t *testing.T) {
	l := New("var count = 1;")
	first := l.NextToken()
	if first.Type != TokenVar {
		t.Fatalf("expected first token var, got %s", first.Type)
	}
	state := l.Snapshot()
	second := l.NextToken()
	l.Restore(state)
	replayed := l.NextToken()
	if replayed != second {
		t.Fatalf("expected restored token to match replayed token, got %#v vs %#v", second, replayed)
	}
}

func assertTokens(t *testing.T, l *Lexer, expected []Token) {
	t.Helper()
	for i, want := range expected {
		got := l.NextToken()
		if got != want {
			t.Fatalf("token %d: expected %#v, got %#v", i, want, got)
		}
	}
}
