package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesArrayLiteralSpreadLength(t *testing.T) {
	program := parseProgram(t, `function main() { return [1, ...[2, 3], 4].length; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 4 {
		t.Fatalf("expected array spread length return code 4, got %d", value)
	}
}

func TestLoweringUsesArrayLiteralSpreadIndex(t *testing.T) {
	program := parseProgram(t, `function main() { return [1, ...[2, 3], 4][2]; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 3 {
		t.Fatalf("expected array spread index return code 3, got %d", value)
	}
}

func TestLoweringEvaluatesArrayLiteralSpreadElementsInOrder(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if ([code++, ...[code++, code]][2] === 3) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected ordered array spread return code 31, got %d", value)
	}
}

func TestLoweringPreservesArrayLiteralSpreadElisions(t *testing.T) {
	program := parseProgram(t, `function main() { if ([...[ , 2]][0] === undefined) { return 141; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 141 {
		t.Fatalf("expected array spread elision return code 141, got %d", value)
	}
}
