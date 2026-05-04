package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesEmptyNewExpressionAsTruthy(t *testing.T) {
	program := parseProgram(t, `function main() { if (new (function () {})()) { return 138; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 138 {
		t.Fatalf("expected new expression return code 138, got %d", value)
	}
}

func TestLoweringEvaluatesNewExpressionArgumentsOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (new (function () {})(code++)) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated new arguments return code 21, got %d", value)
	}
}

func TestLoweringUsesNewExpressionTypeofObject(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof new (function () {})() === "object") { return 139; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 139 {
		t.Fatalf("expected new typeof return code 139, got %d", value)
	}
}

func TestLoweringUsesNewExpressionObjectIdentity(t *testing.T) {
	program := parseProgram(t, `function main() { const value = new (function () {})(); if (value === value) { return 140; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 140 {
		t.Fatalf("expected new identity return code 140, got %d", value)
	}
}
