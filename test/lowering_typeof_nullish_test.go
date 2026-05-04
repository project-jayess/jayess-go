package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesTypeofForMissingObjectMemberAsUndefined(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof ({ value: 1 }).missing === "undefined") { return 120; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 120 {
		t.Fatalf("expected missing object member typeof return code 120, got %d", value)
	}
}

func TestLoweringUsesTypeofForOutOfRangeArrayIndexAsUndefined(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof [1][9] === "undefined") { return 121; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 121 {
		t.Fatalf("expected out-of-range array index typeof return code 121, got %d", value)
	}
}

func TestLoweringUsesTypeofForOutOfRangeStringIndexAsUndefined(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof "jayess"[9] === "undefined") { return 122; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 122 {
		t.Fatalf("expected out-of-range string index typeof return code 122, got %d", value)
	}
}

func TestLoweringKeepsTypeofBooleanMemberAsBoolean(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof ({ ok: false }).ok === "boolean") { return 123; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 123 {
		t.Fatalf("expected boolean member typeof return code 123, got %d", value)
	}
}
