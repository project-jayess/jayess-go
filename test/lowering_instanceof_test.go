package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesPrimitiveInstanceofAsFalse(t *testing.T) {
	program := parseProgram(t, `function main() { if (1 instanceof function () { return 1; }) { return 1; } return 135; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 135 {
		t.Fatalf("expected primitive instanceof return code 135, got %d", value)
	}
}

func TestLoweringUsesPrimitiveInstanceofStrictBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (("x" instanceof function () { return 1; }) === false) { return 136; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 136 {
		t.Fatalf("expected strict primitive instanceof return code 136, got %d", value)
	}
}

func TestLoweringEvaluatesInstanceofOperandsOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (code++ instanceof (code++, function () { return 1; })) { return 1; } return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected single-evaluated instanceof return code 31, got %d", value)
	}
}

func TestLoweringUsesPrimitiveInstanceofStoredFunction(t *testing.T) {
	program := parseProgram(t, `function main() { const Type = function () { return 1; }; if (null instanceof Type) { return 1; } return 137; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 137 {
		t.Fatalf("expected stored function instanceof return code 137, got %d", value)
	}
}
