package test

import (
	"testing"

	"jayess-go/ast"
)

func TestParserTaggedTemplateWithIdentifierTag(t *testing.T) {
	expr := parseExpression(t, "html`<p>${name}</p>`")
	call := requireType[*ast.CallExpression](t, expr)
	if call.Callee != "html" {
		t.Fatalf("expected html callee, got %q", call.Callee)
	}
	if len(call.Arguments) != 1 {
		t.Fatalf("expected one template argument, got %d", len(call.Arguments))
	}
	template := requireType[*ast.TemplateLiteral](t, call.Arguments[0])
	if template.Value != "<p>${name}</p>" {
		t.Fatalf("unexpected template value %q", template.Value)
	}
}

func TestParserTaggedTemplateWithMemberTag(t *testing.T) {
	expr := parseExpression(t, "html.raw`value`")
	invoke := requireType[*ast.InvokeExpression](t, expr)
	member := requireType[*ast.MemberExpression](t, invoke.Callee)
	if member.Property != "raw" {
		t.Fatalf("expected raw member, got %q", member.Property)
	}
	if len(invoke.Arguments) != 1 {
		t.Fatalf("expected one template argument, got %d", len(invoke.Arguments))
	}
	requireType[*ast.TemplateLiteral](t, invoke.Arguments[0])
}
