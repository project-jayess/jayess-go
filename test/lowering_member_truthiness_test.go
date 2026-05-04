package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesObjectNumericMemberTruthiness(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ value: 3 }).value) { return 110; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 110 {
		t.Fatalf("expected object numeric member truthiness return code 110, got %d", value)
	}
}

func TestLoweringUsesObjectEmptyStringMemberTruthiness(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ value: "" }).value) { return 1; } return 111; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 111 {
		t.Fatalf("expected object empty string member truthiness return code 111, got %d", value)
	}
}

func TestLoweringUsesMissingObjectMemberTruthiness(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ value: 1 }).missing) { return 1; } return 112; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 112 {
		t.Fatalf("expected missing object member truthiness return code 112, got %d", value)
	}
}

func TestLoweringUsesFunctionMemberTruthiness(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ f: () => 1 }).f) { return 113; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 113 {
		t.Fatalf("expected function member truthiness return code 113, got %d", value)
	}
}

func TestLoweringUsesObjectArrayIndexTruthiness(t *testing.T) {
	program := parseProgram(t, `function main() { if ([{}][0]) { return 114; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 114 {
		t.Fatalf("expected object array index truthiness return code 114, got %d", value)
	}
}

func TestLoweringUsesOutOfRangeStringIndexTruthiness(t *testing.T) {
	program := parseProgram(t, `function main() { if ("jayess"[99]) { return 1; } return 115; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 115 {
		t.Fatalf("expected out-of-range string index truthiness return code 115, got %d", value)
	}
}

func TestLoweringEvaluatesMemberTruthinessOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (({ value: code++ }).value) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated member truthiness return code 21, got %d", value)
	}
}
