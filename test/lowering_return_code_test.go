package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringExtractsMainReturnCodeFromNumericExpression(t *testing.T) {
	program := parseProgram(t, `function main() { return 1 + 2 * 3; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected folded return code 7, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromLocalBinding(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 4; return value + 3; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected folded local return code 7, got %d", value)
	}
}

func TestLoweringIgnoresUnresolvedIdentifierReturn(t *testing.T) {
	program := parseProgram(t, `function main() { return value; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected unresolved identifier return to stay unresolved")
	}
}

func TestLoweringSelectsMainReturnCodeFromBooleanBinding(t *testing.T) {
	program := parseProgram(t, `function main() { const enabled = true; if (!enabled) { return 1; } return 3; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 3 {
		t.Fatalf("expected folded boolean binding return code 3, got %d", value)
	}
}

func TestLoweringSelectsMainReturnCodeFromComparisonCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const count = 3; if (count >= 3) { return 8; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 8 {
		t.Fatalf("expected folded comparison return code 8, got %d", value)
	}
}

func TestLoweringUsesStringConcatenationInCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const value = "jay" + "ess"; if (value === "jayess") { return 27; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 27 {
		t.Fatalf("expected string concatenation return code 27, got %d", value)
	}
}

func TestLoweringSelectsReturnFromStringComparison(t *testing.T) {
	program := parseProgram(t, `function main() { const mode = "release"; if (mode === "release") { return 14; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 14 {
		t.Fatalf("expected string comparison return code 14, got %d", value)
	}
}

func TestLoweringSelectsReturnFromStringRelationalComparison(t *testing.T) {
	program := parseProgram(t, `function main() { if ("beta" > "alpha" && "same" <= "same") { return 43; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 43 {
		t.Fatalf("expected string relational comparison return code 43, got %d", value)
	}
}

func TestLoweringUsesNumericRelationalPrimitiveCoercion(t *testing.T) {
	program := parseProgram(t, `function main() { if ("7" > 3 && true <= 1 && null >= 0) { return 108; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 108 {
		t.Fatalf("expected numeric relational primitive return code 108, got %d", value)
	}
}

func TestLoweringKeepsStringRelationalComparisonLexical(t *testing.T) {
	program := parseProgram(t, `function main() { if ("10" < "2") { return 109; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 109 {
		t.Fatalf("expected lexical string relational return code 109, got %d", value)
	}
}

func TestLoweringSelectsReturnFromBoolAndNullishComparison(t *testing.T) {
	program := parseProgram(t, `function main() { const ready = true; const missing = undefined; if (ready !== false && missing === undefined) { return 15; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 15 {
		t.Fatalf("expected bool/nullish comparison return code 15, got %d", value)
	}
}

func TestLoweringUsesBooleanValuesForLooseBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (true == false) { return 1; } if (true != false) { return 64; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 64 {
		t.Fatalf("expected boolean loose inequality return code 64, got %d", value)
	}
}
