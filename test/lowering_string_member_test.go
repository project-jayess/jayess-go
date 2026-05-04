package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesStringLengthMember(t *testing.T) {
	program := parseProgram(t, `function main() { return "hello".length; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 5 {
		t.Fatalf("expected string length return code 5, got %d", value)
	}
}

func TestLoweringUsesStringIndexExpression(t *testing.T) {
	program := parseProgram(t, `function main() { if ("hello"[1] === "e") { return 44; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 44 {
		t.Fatalf("expected string index return code 44, got %d", value)
	}
}

func TestLoweringTreatsOutOfRangeStringIndexAsUndefined(t *testing.T) {
	program := parseProgram(t, `function main() { if ("hello"[9] === undefined) { return 45; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 45 {
		t.Fatalf("expected out-of-range string index return code 45, got %d", value)
	}
}

func TestLoweringEvaluatesStringIndexTargetOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if ((code++, "jayess")[2] === "y") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated string index return code 21, got %d", value)
	}
}
