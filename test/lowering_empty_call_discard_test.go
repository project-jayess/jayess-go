package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringDiscardsEmptyFunctionCallArguments(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; (function () {})(code++); return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected empty call argument side effects return code 21, got %d", value)
	}
}

func TestLoweringDiscardsEmptyFunctionCallCalleeAndArguments(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; (code++, function () {})(code++); return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected empty call callee and argument side effects return code 31, got %d", value)
	}
}

func TestLoweringDiscardsConditionalEmptyFunctionCall(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; (true ? function () {} : function () { return code++; })(code++); return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected conditional empty call side effects return code 21, got %d", value)
	}
}

func TestLoweringDoesNotEvaluateNonEmptyFunctionCallBody(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; (function () { return code++; })(); return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 11 {
		t.Fatalf("expected non-empty function call body to remain unevaluated with return code 11, got %d", value)
	}
}
