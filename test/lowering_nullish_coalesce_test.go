package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringEvaluatesNonNullishCoalesceLeftUpdateOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = code++ ?? 9; return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated coalesce update return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesIntCommaCoalesceLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = (code++, 2) ?? 9; return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 22 {
		t.Fatalf("expected single-evaluated int comma coalesce return code 22, got %d", value)
	}
}

func TestLoweringEvaluatesNullishCoalesceVoidUpdateOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = void code++ ?? 9; return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 29 {
		t.Fatalf("expected nullish coalesce void update return code 29, got %d", value)
	}
}

func TestLoweringEvaluatesStringCoalesceLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = (code++, "ready") ?? "fallback"; if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated string coalesce return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesNullishStringCoalesceProbeOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = (code++, null) ?? "ready"; if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated nullish string coalesce return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesBooleanCoalesceLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = (code++, true) ?? false; if (value) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated boolean coalesce return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesNullishBooleanCoalesceLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = (code++, null) ?? true; if (value) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated nullish boolean coalesce return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesObjectCoalesceLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, []) ?? {}; if (typeof chosen === "object") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated object coalesce return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesFunctionCoalesceLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, () => 1) ?? (() => 2); if (typeof chosen === "function") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated function coalesce return code 21, got %d", value)
	}
}

func TestLoweringExtractsReturnFromNullishCoalescingExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = 18; return null ?? fallback; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 18 {
		t.Fatalf("expected nullish coalescing return code 18, got %d", value)
	}
}

func TestLoweringPreservesFalsyNonNullishCoalescingLeft(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 0 ?? 19; return value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 0 {
		t.Fatalf("expected falsy non-nullish coalescing return code 0, got %d", value)
	}
}

func TestLoweringUsesStringNullishCoalescingInCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const mode = undefined ?? "release"; if (mode === "release") { return 20; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 20 {
		t.Fatalf("expected string nullish coalescing return code 20, got %d", value)
	}
}

func TestLoweringAppliesNullishAssignmentForNullishLocal(t *testing.T) {
	program := parseProgram(t, `function main() { var code = undefined; code ??= 21; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected nullish assignment return code 21, got %d", value)
	}
}

func TestLoweringPreservesNonNullishLocalForNullishAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 0; code ??= 22; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 0 {
		t.Fatalf("expected preserved nullish assignment return code 0, got %d", value)
	}
}
