package lexer

import "testing"

func TestNextTokenSkipsCommentsAndTracksLocations(t *testing.T) {
	input := "// lead\nvar value = 12n; /* block */\nconst label = \"kimchi\";\n"
	l := New(input)

	tests := []struct {
		typ     TokenType
		literal string
		line    int
		column  int
	}{
		{TokenVar, "var", 2, 1},
		{TokenIdent, "value", 2, 5},
		{TokenAssign, "=", 2, 11},
		{TokenBigInt, "12", 2, 13},
		{TokenSemicolon, ";", 2, 16},
		{TokenConst, "const", 3, 1},
		{TokenIdent, "label", 3, 7},
		{TokenAssign, "=", 3, 13},
		{TokenString, "kimchi", 3, 15},
		{TokenSemicolon, ";", 3, 23},
		{TokenEOF, "", 4, 1},
	}

	for i, tt := range tests {
		token := l.NextToken()
		if token.Type != tt.typ || token.Literal != tt.literal || token.Line != tt.line || token.Column != tt.column {
			t.Fatalf("token %d: expected (%s, %q, %d, %d), got (%s, %q, %d, %d)", i, tt.typ, tt.literal, tt.line, tt.column, token.Type, token.Literal, token.Line, token.Column)
		}
	}
}

func TestLexerSnapshotRestore(t *testing.T) {
	l := New("var count = 1;")
	first := l.NextToken()
	if first.Type != TokenVar {
		t.Fatalf("expected first token var, got %s", first.Type)
	}
	state := l.Snapshot()
	second := l.NextToken()
	if second.Type != TokenIdent || second.Literal != "count" {
		t.Fatalf("expected identifier count, got %s %q", second.Type, second.Literal)
	}
	l.Restore(state)
	replayed := l.NextToken()
	if replayed.Type != second.Type || replayed.Literal != second.Literal || replayed.Line != second.Line || replayed.Column != second.Column {
		t.Fatalf("expected restored token to match replayed token, got %#v vs %#v", second, replayed)
	}
}
