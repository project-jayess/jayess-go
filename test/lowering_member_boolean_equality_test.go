package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesObjectBooleanMemberStrictEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ ok: true }).ok === true) { return 116; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 116 {
		t.Fatalf("expected object boolean member equality return code 116, got %d", value)
	}
}

func TestLoweringUsesObjectBooleanIndexStrictInequality(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ ok: false })["ok"] !== true) { return 117; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 117 {
		t.Fatalf("expected object boolean index inequality return code 117, got %d", value)
	}
}

func TestLoweringUsesArrayBooleanIndexStrictEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if ([false, true][1] === true) { return 118; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 118 {
		t.Fatalf("expected array boolean index equality return code 118, got %d", value)
	}
}

func TestLoweringKeepsNumericMemberDistinctFromBoolean(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ value: 1 }).value !== true) { return 119; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 119 {
		t.Fatalf("expected numeric member boolean mismatch return code 119, got %d", value)
	}
}
