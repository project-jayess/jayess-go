package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringDoesNotUseFunctionTruthinessForLooseBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (f == true) { return 1; } if (f != true) { return 63; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 63 {
		t.Fatalf("expected function loose boolean inequality return code 63, got %d", value)
	}
}

func TestLoweringSeparatesStrictFunctionNumberEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (f === 1) { return 1; } if (f !== 1) { return 67; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 67 {
		t.Fatalf("expected strict function number inequality return code 67, got %d", value)
	}
}

func TestLoweringEvaluatesReferencePrimitiveMismatchOperandOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if ((code++, 1) !== (() => 1)) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated reference primitive mismatch return code 21, got %d", value)
	}
}

func TestLoweringSeparatesStrictFunctionStringEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (f === "fn") { return 1; } if (f !== "fn") { return 68; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 68 {
		t.Fatalf("expected strict function string inequality return code 68, got %d", value)
	}
}
