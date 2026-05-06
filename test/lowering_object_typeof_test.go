package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesTypeofForObjectAndArrayLiterals(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof ({ value: 1 }) === "object" && typeof [] === "object") { return 49; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 49 {
		t.Fatalf("expected typeof object and array literals return code 49, got %d", value)
	}
}

func TestLoweringUsesTypeofForObjectBindingCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const value = {}; if (typeof value === "object") { return 50; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 50 {
		t.Fatalf("expected typeof object binding return code 50, got %d", value)
	}
}

func TestLoweringTreatsArrayBindingAsTruthy(t *testing.T) {
	program := parseProgram(t, `function main() { const items = []; if (items) { return 51; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 51 {
		t.Fatalf("expected array binding truthy return code 51, got %d", value)
	}
}

func TestLoweringUsesTypeofForAssignedArrayBinding(t *testing.T) {
	program := parseProgram(t, `function main() { var items = undefined; items = []; if (typeof items === "object") { return 52; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 52 {
		t.Fatalf("expected typeof assigned array binding return code 52, got %d", value)
	}
}

func TestLoweringUsesTypeofForObjectNullishExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = []; if (typeof (undefined ?? fallback) === "object") { return 88; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 88 {
		t.Fatalf("expected typeof object nullish return code 88, got %d", value)
	}
}

func TestLoweringUsesTypeofForObjectCommaExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = {}; if (typeof (0, fallback) === "object") { return 89; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 89 {
		t.Fatalf("expected typeof object comma return code 89, got %d", value)
	}
}

func TestLoweringUsesTypeofForFreshObjectNullishExpression(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof ([] ?? {}) === "object") { return 100; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 100 {
		t.Fatalf("expected typeof fresh object nullish return code 100, got %d", value)
	}
}

func TestLoweringUsesTypeofForFreshObjectLogicalExpression(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof ({} || []) === "object") { return 101; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 101 {
		t.Fatalf("expected typeof fresh object logical return code 101, got %d", value)
	}
}

func TestLoweringEvaluatesTypeofFreshObjectLogicalOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (typeof ((code++, {}) || []) === "object") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated typeof fresh object logical return code 21, got %d", value)
	}
}
