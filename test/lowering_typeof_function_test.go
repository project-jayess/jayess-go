package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesTypeofForKnownLocalCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const code = 24; if (typeof code === "number") { return code; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 24 {
		t.Fatalf("expected typeof known local return code 24, got %d", value)
	}
}

func TestLoweringEvaluatesTypeofOperandOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (typeof (code++, "x") === "string") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated typeof operand return code 21, got %d", value)
	}
}

func TestLoweringUsesTypeofForMissingIdentifierCondition(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof missing === "undefined") { return 25; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 25 {
		t.Fatalf("expected typeof missing identifier return code 25, got %d", value)
	}
}

func TestLoweringUsesTypeofForFunctionExpressionCondition(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof (() => 1) === "function") { return 44; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 44 {
		t.Fatalf("expected typeof function expression return code 44, got %d", value)
	}
}

func TestLoweringUsesTypeofForFunctionBindingCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (typeof f === "function") { return 45; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 45 {
		t.Fatalf("expected typeof function binding return code 45, got %d", value)
	}
}

func TestLoweringUsesTypeofForAssignedFunctionBinding(t *testing.T) {
	program := parseProgram(t, `function main() { var f = undefined; f = function () { return 1; }; if (typeof f === "function") { return 46; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 46 {
		t.Fatalf("expected typeof assigned function binding return code 46, got %d", value)
	}
}

func TestLoweringUsesTypeofForFunctionConditionalExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const left = () => 1; const right = () => 2; if (typeof (true ? left : right) === "function") { return 86; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 86 {
		t.Fatalf("expected typeof function conditional return code 86, got %d", value)
	}
}

func TestLoweringUsesTypeofForFunctionLogicalExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const left = () => 1; const right = () => 2; if (typeof (left || right) === "function") { return 87; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 87 {
		t.Fatalf("expected typeof function logical return code 87, got %d", value)
	}
}

func TestLoweringEvaluatesTypeofFreshFunctionLogicalOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (typeof ((code++, (() => 1)) || (() => 2)) === "function") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated typeof fresh function logical return code 21, got %d", value)
	}
}

func TestLoweringUsesTypeofForFreshFunctionNullishExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = () => 2; if (typeof ((() => 1) ?? fallback) === "function") { return 98; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 98 {
		t.Fatalf("expected typeof fresh function nullish return code 98, got %d", value)
	}
}

func TestLoweringUsesTypeofForFreshFunctionConditionalExpression(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof (true ? (() => 1) : (() => 2)) === "function") { return 99; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 99 {
		t.Fatalf("expected typeof fresh function conditional return code 99, got %d", value)
	}
}

func TestLoweringEvaluatesTypeofFreshFunctionConditionalOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (typeof ((code++, true) ? (() => 1) : (() => 2)) === "function") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated typeof fresh function conditional return code 21, got %d", value)
	}
}
