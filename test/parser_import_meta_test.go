package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserImportMetaExpression(t *testing.T) {
	expr := parseExpression(t, `import.meta`)
	requireType[*ast.ImportMetaExpression](t, expr)
}

func TestParserImportMetaMemberExpressionStatement(t *testing.T) {
	program := parseProgram(t, `import.meta.url;`)
	stmt := requireType[*ast.ExpressionStatement](t, program.Statements[0])
	member := requireType[*ast.MemberExpression](t, stmt.Expression)
	requireType[*ast.ImportMetaExpression](t, member.Target)
	if member.Property != "url" {
		t.Fatalf("expected import.meta.url property, got %q", member.Property)
	}
}
