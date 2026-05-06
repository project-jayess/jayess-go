package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringComparesSameObjectBindingAsEqual(t *testing.T) {
	program := parseProgram(t, `function main() { const value = {}; if (value === value) { return 53; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 53 {
		t.Fatalf("expected same object binding equality return code 53, got %d", value)
	}
}

func TestLoweringComparesDistinctObjectBindingsAsNotEqual(t *testing.T) {
	program := parseProgram(t, `function main() { const left = {}; const right = {}; if (left !== right) { return 54; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 54 {
		t.Fatalf("expected distinct object binding inequality return code 54, got %d", value)
	}
}

func TestLoweringComparesFreshObjectLiteralsAsNotEqual(t *testing.T) {
	program := parseProgram(t, `function main() { if (({}) !== ({})) { return 55; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 55 {
		t.Fatalf("expected fresh object literal inequality return code 55, got %d", value)
	}
}

func TestLoweringComparesObjectBindingAgainstNullish(t *testing.T) {
	program := parseProgram(t, `function main() { const value = {}; if (value != null && value !== undefined) { return 56; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 56 {
		t.Fatalf("expected object nullish comparison return code 56, got %d", value)
	}
}

func TestLoweringPreservesObjectIdentityThroughBindingAlias(t *testing.T) {
	program := parseProgram(t, `function main() { const left = []; const right = left; if (left === right) { return 57; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 57 {
		t.Fatalf("expected object alias equality return code 57, got %d", value)
	}
}

func TestLoweringPreservesObjectIdentityThroughNullishCoalescing(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = []; const chosen = undefined ?? fallback; if (chosen === fallback) { return 84; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 84 {
		t.Fatalf("expected object nullish identity return code 84, got %d", value)
	}
}

func TestLoweringMaterializesFreshObjectThroughNullishCoalescing(t *testing.T) {
	program := parseProgram(t, `function main() { const chosen = [] ?? {}; if (typeof chosen === "object" && chosen === chosen) { return 96; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 96 {
		t.Fatalf("expected fresh object nullish materialization return code 96, got %d", value)
	}
}

func TestLoweringMaterializesFreshObjectThroughLogicalOr(t *testing.T) {
	program := parseProgram(t, `function main() { const chosen = {} || []; if (typeof chosen === "object" && chosen === chosen) { return 97; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 97 {
		t.Fatalf("expected fresh object logical materialization return code 97, got %d", value)
	}
}

func TestLoweringMaterializesFreshObjectThroughLogicalAnd(t *testing.T) {
	program := parseProgram(t, `function main() { const chosen = {} && []; if (typeof chosen === "object" && chosen === chosen) { return 103; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 103 {
		t.Fatalf("expected fresh object logical-and materialization return code 103, got %d", value)
	}
}

func TestLoweringPreservesObjectIdentityThroughConditional(t *testing.T) {
	program := parseProgram(t, `function main() { const left = []; const right = {}; const chosen = true ? left : right; if (chosen === left) { return 85; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 85 {
		t.Fatalf("expected object conditional identity return code 85, got %d", value)
	}
}

func TestLoweringEvaluatesObjectConditionalConditionOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, true) ? {} : []; if (typeof chosen === "object" && chosen === chosen) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated object conditional return code 21, got %d", value)
	}
}

func TestLoweringPreservesObjectIdentityThroughNullishAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = []; var chosen = null; chosen ??= fallback; if (chosen === fallback) { return 92; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 92 {
		t.Fatalf("expected object nullish assignment identity return code 92, got %d", value)
	}
}

func TestLoweringPreservesObjectIdentityThroughLogicalOrAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = {}; var chosen = null; chosen ||= fallback; if (chosen === fallback) { return 93; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 93 {
		t.Fatalf("expected object logical or assignment identity return code 93, got %d", value)
	}
}
