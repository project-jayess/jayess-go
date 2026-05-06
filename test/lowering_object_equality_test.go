package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringDoesNotUseObjectTruthinessForLooseBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const value = {}; if (value == true) { return 1; } if (value != true) { return 65; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 65 {
		t.Fatalf("expected object loose boolean inequality return code 65, got %d", value)
	}
}

func TestLoweringDoesNotUseArrayTruthinessForLooseBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const value = []; if (value == true) { return 1; } if (value != true) { return 66; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 66 {
		t.Fatalf("expected array loose boolean inequality return code 66, got %d", value)
	}
}

func TestLoweringSeparatesStrictObjectNumberEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const value = {}; if (value === 1) { return 1; } if (value !== 1) { return 69; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 69 {
		t.Fatalf("expected strict object number inequality return code 69, got %d", value)
	}
}

func TestLoweringEvaluatesObjectPrimitiveMismatchOperandOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if ((code++, "x") !== ({})) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated object primitive mismatch return code 21, got %d", value)
	}
}

func TestLoweringSeparatesStrictArrayStringEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const value = []; if (value === "items") { return 1; } if (value !== "items") { return 70; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 70 {
		t.Fatalf("expected strict array string inequality return code 70, got %d", value)
	}
}
