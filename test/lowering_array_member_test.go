package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesArrayLengthMember(t *testing.T) {
	program := parseProgram(t, `function main() { return [1, 2, 3].length; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 3 {
		t.Fatalf("expected array length return code 3, got %d", value)
	}
}

func TestLoweringUsesArrayNumericIndexExpression(t *testing.T) {
	program := parseProgram(t, `function main() { return [4, 5, 6][1]; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 5 {
		t.Fatalf("expected array index return code 5, got %d", value)
	}
}

func TestLoweringUsesArrayStringIndexExpression(t *testing.T) {
	program := parseProgram(t, `function main() { if (["a", "b"][1] === "b") { return 46; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 46 {
		t.Fatalf("expected array string index return code 46, got %d", value)
	}
}

func TestLoweringUsesArrayBooleanIndexExpression(t *testing.T) {
	program := parseProgram(t, `function main() { if ([false, true][1]) { return 47; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 47 {
		t.Fatalf("expected array boolean index return code 47, got %d", value)
	}
}

func TestLoweringTreatsArrayElisionAsUndefined(t *testing.T) {
	program := parseProgram(t, `function main() { if ([, 2][0] === undefined) { return 48; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 48 {
		t.Fatalf("expected array elision return code 48, got %d", value)
	}
}

func TestLoweringEvaluatesArrayIndexElementsOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if ([code++, code][1] === 2) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated array index return code 21, got %d", value)
	}
}
