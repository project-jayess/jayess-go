package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringExtractsMainReturnCodeFromConditionalExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const count = 4; return count > 3 ? 11 : 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 11 {
		t.Fatalf("expected folded conditional expression return code 11, got %d", value)
	}
}

func TestLoweringEvaluatesStringConditionalConditionOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = (code++, true) ? "ready" : "fallback"; if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated string conditional return code 21, got %d", value)
	}
}

func TestLoweringUsesConditionalExpressionForBooleanBinding(t *testing.T) {
	program := parseProgram(t, `function main() { const count = 0; const ready = count ? false : true; if (ready) { return 5; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 5 {
		t.Fatalf("expected folded conditional boolean return code 5, got %d", value)
	}
}

func TestLoweringAppliesConditionalExpressionStatementSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; true ? code++ : code--; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 2 {
		t.Fatalf("expected conditional expression statement return code 2, got %d", value)
	}
}

func TestLoweringAppliesCommaExpressionStatementSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; code++, code++; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 3 {
		t.Fatalf("expected comma expression statement return code 3, got %d", value)
	}
}

func TestLoweringEvaluatesDiscardExpressionOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; code++, "done"; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 2 {
		t.Fatalf("expected single-evaluated discard expression return code 2, got %d", value)
	}
}

func TestLoweringEvaluatesStringCommaExpressionProbeOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = (code++, "ready"); if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated string comma expression return code 21, got %d", value)
	}
}

func TestLoweringExtractsReturnFromCommaExpression(t *testing.T) {
	program := parseProgram(t, `function main() { return (1, 16); }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 16 {
		t.Fatalf("expected comma expression return code 16, got %d", value)
	}
}

func TestLoweringUsesCommaExpressionInConstantCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const mode = ("debug", "release"); if (mode === "release") { return 17; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 17 {
		t.Fatalf("expected comma condition return code 17, got %d", value)
	}
}

func TestLoweringTreatsNullAndUndefinedConditionsAsFalse(t *testing.T) {
	program := parseProgram(t, `function main() { const value = null; if (value) { return 1; } const missing = undefined; if (missing) { return 2; } return 6; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 6 {
		t.Fatalf("expected nullish condition return code 6, got %d", value)
	}
}

func TestLoweringUsesStringBindingTruthiness(t *testing.T) {
	program := parseProgram(t, `function main() { const label = "ready"; if (label) { return 24; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 24 {
		t.Fatalf("expected string truthiness return code 24, got %d", value)
	}
}

func TestLoweringUsesEmptyStringLiteralAsFalsy(t *testing.T) {
	program := parseProgram(t, `function main() { if ("") { return 1; } return 25; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 25 {
		t.Fatalf("expected empty string falsy return code 25, got %d", value)
	}
}

func TestLoweringUsesVoidAsUndefinedCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const value = void 0; if (value === undefined) { return 26; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 26 {
		t.Fatalf("expected void undefined return code 26, got %d", value)
	}
}

func TestLoweringUsesVoidExpressionAsFalsyCondition(t *testing.T) {
	program := parseProgram(t, `function main() { if (void 0) { return 1; } return 102; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 102 {
		t.Fatalf("expected void falsy return code 102, got %d", value)
	}
}
