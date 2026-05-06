package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringTreatsFunctionExpressionAsTruthy(t *testing.T) {
	program := parseProgram(t, `function main() { if (() => 1) { return 47; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 47 {
		t.Fatalf("expected function expression truthy return code 47, got %d", value)
	}
}

func TestLoweringTreatsFunctionBindingAsTruthy(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (!f) { return 1; } return 48; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 48 {
		t.Fatalf("expected function binding truthy return code 48, got %d", value)
	}
}

func TestLoweringComparesSameFunctionBindingAsEqual(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (f === f) { return 58; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 58 {
		t.Fatalf("expected same function binding equality return code 58, got %d", value)
	}
}

func TestLoweringComparesDistinctFunctionBindingsAsNotEqual(t *testing.T) {
	program := parseProgram(t, `function main() { const left = () => 1; const right = () => 1; if (left !== right) { return 59; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 59 {
		t.Fatalf("expected distinct function binding inequality return code 59, got %d", value)
	}
}

func TestLoweringComparesFreshFunctionExpressionsAsNotEqual(t *testing.T) {
	program := parseProgram(t, `function main() { if ((() => 1) !== (() => 1)) { return 60; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 60 {
		t.Fatalf("expected fresh function expression inequality return code 60, got %d", value)
	}
}

func TestLoweringComparesFunctionBindingAgainstNullish(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (f != null && f !== undefined) { return 61; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 61 {
		t.Fatalf("expected function nullish comparison return code 61, got %d", value)
	}
}

func TestLoweringPreservesFunctionIdentityThroughBindingAlias(t *testing.T) {
	program := parseProgram(t, `function main() { const left = () => 1; const right = left; if (left === right) { return 62; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 62 {
		t.Fatalf("expected function alias equality return code 62, got %d", value)
	}
}

func TestLoweringPreservesFunctionIdentityThroughLogicalOr(t *testing.T) {
	program := parseProgram(t, `function main() { const left = () => 1; const right = () => 2; const chosen = left || right; if (chosen === left) { return 82; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 82 {
		t.Fatalf("expected function logical identity return code 82, got %d", value)
	}
}

func TestLoweringMaterializesFreshFunctionThroughLogicalAnd(t *testing.T) {
	program := parseProgram(t, `function main() { const chosen = (() => 1) && (() => 2); if (typeof chosen === "function" && chosen === chosen) { return 102; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 102 {
		t.Fatalf("expected fresh function logical-and materialization return code 102, got %d", value)
	}
}

func TestLoweringMaterializesFreshFunctionThroughNullishCoalescing(t *testing.T) {
	program := parseProgram(t, `function main() { const chosen = (() => 1) ?? (() => 2); if (typeof chosen === "function" && chosen === chosen) { return 94; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 94 {
		t.Fatalf("expected fresh function nullish materialization return code 94, got %d", value)
	}
}

func TestLoweringEvaluatesFreshFunctionNullishMaterializationOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, null) ?? (() => 2); if (typeof chosen === "function" && chosen === chosen) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated fresh function nullish materialization return code 21, got %d", value)
	}
}

func TestLoweringMaterializesFreshFunctionThroughConditional(t *testing.T) {
	program := parseProgram(t, `function main() { const chosen = true ? (() => 1) : (() => 2); if (typeof chosen === "function" && chosen === chosen) { return 95; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 95 {
		t.Fatalf("expected fresh function conditional materialization return code 95, got %d", value)
	}
}

func TestLoweringEvaluatesFunctionConditionalConditionOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, true) ? (() => 1) : (() => 2); if (typeof chosen === "function" && chosen === chosen) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated function conditional return code 21, got %d", value)
	}
}

func TestLoweringPreservesFunctionIdentityThroughComma(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = () => 1; const chosen = (0, fallback); if (chosen === fallback) { return 83; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 83 {
		t.Fatalf("expected function comma identity return code 83, got %d", value)
	}
}

func TestLoweringPreservesFunctionIdentityThroughLogicalAndAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var chosen = () => 1; const next = () => 2; chosen &&= next; if (chosen === next) { return 90; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 90 {
		t.Fatalf("expected function logical assignment identity return code 90, got %d", value)
	}
}

func TestLoweringPreservesFunctionIdentityThroughLogicalOrAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { const left = () => 1; const right = () => 2; var chosen = left; chosen ||= right; if (chosen === left) { return 91; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 91 {
		t.Fatalf("expected function logical or assignment identity return code 91, got %d", value)
	}
}
