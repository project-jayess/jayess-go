package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringEvaluatesNumericAddLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = (code++, 2) + 3; return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 25 {
		t.Fatalf("expected single-evaluated numeric add return code 25, got %d", value)
	}
}

func TestLoweringEvaluatesNumericAddAssignmentRightOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = 2; value += (code++, 3); return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 25 {
		t.Fatalf("expected single-evaluated numeric add-assignment return code 25, got %d", value)
	}
}

func TestLoweringEvaluatesScalarAssignmentProbeOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = 0; value = (code++, "ready"); if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated scalar assignment return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesNumericStringCoercionOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = +(code++, "2"); return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 22 {
		t.Fatalf("expected single-evaluated numeric string coercion return code 22, got %d", value)
	}
}

func TestLoweringDoesNotCommitFailedNumericStringCoercion(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; +(code++, "x"); return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 1 {
		t.Fatalf("expected failed numeric coercion to keep return code 1, got %d", value)
	}
}

func TestLoweringDoesNotCommitFailedNumericConditionalBoolProbe(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; +((code++, true) ? "x" : false); return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 1 {
		t.Fatalf("expected failed numeric conditional probe to keep return code 1, got %d", value)
	}
}

func TestLoweringUsesNumericBinaryFalsyCondition(t *testing.T) {
	program := parseProgram(t, `function main() { if (1 - 1) { return 1; } return 81; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 81 {
		t.Fatalf("expected numeric binary falsy return code 81, got %d", value)
	}
}

func TestLoweringUsesPrimitiveNumericBinaryOperators(t *testing.T) {
	program := parseProgram(t, `function main() { return ("7" - true) + (true * 4) + (null + 2); }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 12 {
		t.Fatalf("expected primitive numeric binary return code 12, got %d", value)
	}
}

func TestLoweringKeepsStringPlusAsConcatenation(t *testing.T) {
	program := parseProgram(t, `function main() { const value = "7" + 1; if (value === "71") { return 106; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 106 {
		t.Fatalf("expected string plus concatenation return code 106, got %d", value)
	}
}

func TestLoweringAppliesStringAddAssignmentBeforeCondition(t *testing.T) {
	program := parseProgram(t, `function main() { var value = "jay"; value += "ess"; if (value === "jayess") { return 28; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 28 {
		t.Fatalf("expected string add assignment return code 28, got %d", value)
	}
}

func TestLoweringAppliesStringNumberAddAssignmentBeforeCondition(t *testing.T) {
	program := parseProgram(t, `function main() { var value = "code:"; value += 7; if (value === "code:7") { return 74; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 74 {
		t.Fatalf("expected string number add assignment return code 74, got %d", value)
	}
}

func TestLoweringAppliesStringPrimitiveAddAssignmentBeforeCondition(t *testing.T) {
	program := parseProgram(t, `function main() { var value = "ready:"; value += false; value += ":"; value += undefined; if (value === "ready:false:undefined") { return 75; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 75 {
		t.Fatalf("expected string primitive add assignment return code 75, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeAfterNumericAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; code += 4; code *= 2; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 10 {
		t.Fatalf("expected folded assignment return code 10, got %d", value)
	}
}

func TestLoweringAppliesPrimitiveNumericCompoundAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 10; code -= "3"; code *= true; code += null; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected primitive numeric assignment return code 7, got %d", value)
	}
}

func TestLoweringKeepsStringAssignmentAsString(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 1; value = "7"; if (value === "7") { return 107; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 107 {
		t.Fatalf("expected string assignment return code 107, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeAfterBooleanAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var ready = false; ready = true; if (ready) { return 7; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected folded boolean assignment return code 7, got %d", value)
	}
}
