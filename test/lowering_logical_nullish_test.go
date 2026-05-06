package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringEvaluatesObjectOrLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, []) || {}; if (typeof chosen === "object") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated object or return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesFunctionOrLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, () => 1) || (() => 2); if (typeof chosen === "function") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated function or return code 21, got %d", value)
	}
}

func TestLoweringSelectsMainReturnCodeFromLogicalCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const count = 2; const ready = true; if (ready && count < 2) { return 1; } return 6; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 6 {
		t.Fatalf("expected folded logical return code 6, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromLogicalAndValue(t *testing.T) {
	program := parseProgram(t, `function main() { return true && 5; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 5 {
		t.Fatalf("expected logical and value return code 5, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromLogicalOrValue(t *testing.T) {
	program := parseProgram(t, `function main() { return 0 || 7; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected logical or value return code 7, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromShortCircuitLogicalValue(t *testing.T) {
	program := parseProgram(t, `function main() { return 6 || 9; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 6 {
		t.Fatalf("expected short circuit logical value return code 6, got %d", value)
	}
}

func TestLoweringEvaluatesIntLogicalOrLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = code++ || 9; return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated int logical or return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesIntLogicalAndLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 0; var value = code++ && 9; return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 10 {
		t.Fatalf("expected single-evaluated int logical and return code 10, got %d", value)
	}
}

func TestLoweringUsesStringLogicalAndValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = "ready" && "go"; if (value === "go") { return 8; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 8 {
		t.Fatalf("expected string logical and return code 8, got %d", value)
	}
}

func TestLoweringUsesStringLogicalOrValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = "" || "fallback"; if (value === "fallback") { return 9; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 9 {
		t.Fatalf("expected string logical or return code 9, got %d", value)
	}
}

func TestLoweringEvaluatesStringLogicalOrLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = (code++, "ready") || "fallback"; if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated string logical or return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesStringLogicalAndFallbackLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = (code++, 1) && "ready"; if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated string logical and return code 21, got %d", value)
	}
}

func TestLoweringUsesNullishLogicalAndValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = null && undefined; if (value === null) { return 10; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 10 {
		t.Fatalf("expected nullish logical and return code 10, got %d", value)
	}
}

func TestLoweringEvaluatesNullishLogicalAndLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = (code++, null) && undefined; if (value === null) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated nullish logical and return code 21, got %d", value)
	}
}

func TestLoweringAppliesLogicalOrAndAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 0; code ||= 23; var ready = true; ready &&= false; if (!ready) { return code; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 23 {
		t.Fatalf("expected logical assignment return code 23, got %d", value)
	}
}

func TestLoweringEvaluatesLogicalScalarAssignmentProbeOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = ""; value ||= (code++, "ready"); if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated logical scalar assignment return code 21, got %d", value)
	}
}
