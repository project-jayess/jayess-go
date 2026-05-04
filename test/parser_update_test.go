package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserPostfixUpdateExpression(t *testing.T) {
	program := parseProgram(t, `count++; item.value--;`)
	first := requireType[*ast.ExpressionStatement](t, program.Statements[0])
	update := requireType[*ast.UpdateExpression](t, first.Expression)
	if update.Operator != ast.UpdateIncrement || update.Prefix {
		t.Fatalf("unexpected postfix increment: %#v", update)
	}
	requireType[*ast.Identifier](t, update.Target)

	second := requireType[*ast.ExpressionStatement](t, program.Statements[1])
	memberUpdate := requireType[*ast.UpdateExpression](t, second.Expression)
	if memberUpdate.Operator != ast.UpdateDecrement || memberUpdate.Prefix {
		t.Fatalf("unexpected postfix decrement: %#v", memberUpdate)
	}
	requireType[*ast.MemberExpression](t, memberUpdate.Target)
}

func TestParserPrefixUpdateExpression(t *testing.T) {
	expr := parseExpression(t, `++count`)
	update := requireType[*ast.UpdateExpression](t, expr)
	if update.Operator != ast.UpdateIncrement || !update.Prefix {
		t.Fatalf("unexpected prefix increment: %#v", update)
	}
}

func TestParserLineBreakBeforeIncrementStartsNewStatement(t *testing.T) {
	program := parseProgram(t, "count\n++next;")
	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}
	first := requireType[*ast.ExpressionStatement](t, program.Statements[0])
	ident := requireType[*ast.Identifier](t, first.Expression)
	if ident.Name != "count" {
		t.Fatalf("expected first expression count, got %q", ident.Name)
	}
	second := requireType[*ast.ExpressionStatement](t, program.Statements[1])
	update := requireType[*ast.UpdateExpression](t, second.Expression)
	if !update.Prefix || update.Operator != ast.UpdateIncrement {
		t.Fatalf("expected prefix increment after line break, got %#v", update)
	}
}

func TestParserRejectsInvalidUpdateTarget(t *testing.T) {
	_, err := parseProgramError(`(count + 1)++;`)
	if err == nil {
		t.Fatalf("expected invalid update target error")
	}
}
