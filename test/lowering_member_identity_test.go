package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesObjectFunctionMemberIdentity(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (({ f: f }).f === f) { return 104; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 104 {
		t.Fatalf("expected object function member identity return code 104, got %d", value)
	}
}

func TestLoweringUsesObjectFunctionIndexTypeof(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof ({ f: () => 1 })["f"] === "function") { return 105; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 105 {
		t.Fatalf("expected object function index typeof return code 105, got %d", value)
	}
}

func TestLoweringUsesArrayFunctionIndexTypeof(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof [() => 1][0] === "function") { return 106; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 106 {
		t.Fatalf("expected array function index typeof return code 106, got %d", value)
	}
}

func TestLoweringUsesObjectObjectMemberIdentity(t *testing.T) {
	program := parseProgram(t, `function main() { const child = []; if (({ child: child }).child === child) { return 107; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 107 {
		t.Fatalf("expected object object member identity return code 107, got %d", value)
	}
}

func TestLoweringUsesObjectObjectIndexTypeof(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof ({ child: {} })["child"] === "object") { return 108; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 108 {
		t.Fatalf("expected object object index typeof return code 108, got %d", value)
	}
}

func TestLoweringUsesArrayObjectIndexTypeof(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof [{}][0] === "object") { return 109; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 109 {
		t.Fatalf("expected array object index typeof return code 109, got %d", value)
	}
}
