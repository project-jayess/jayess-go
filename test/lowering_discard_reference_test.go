package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringDiscardsObjectLiteralWithSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; ({ value: code++ }); return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected discarded object side effects return code 21, got %d", value)
	}
}

func TestLoweringDiscardsArrayLiteralWithSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; [code++, code++]; return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected discarded array side effects return code 31, got %d", value)
	}
}

func TestLoweringDiscardsObjectSpreadWithSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; ({ ...{ value: code++ }, second: code++ }); return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected discarded object spread side effects return code 31, got %d", value)
	}
}

func TestLoweringDiscardsEmptyNewExpressionWithArguments(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; new (function () {})(code++); return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected discarded new arguments return code 21, got %d", value)
	}
}
