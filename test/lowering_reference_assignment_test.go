package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringPreservesMemberAssignmentTargetAndValueSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; ({ target: code++ }).value = code++; return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected member assignment side effects return code 31, got %d", value)
	}
}

func TestLoweringPreservesIndexAssignmentTargetKeyAndValueSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; ({ target: code++ })[code++] = code++; return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 41 {
		t.Fatalf("expected index assignment side effects return code 41, got %d", value)
	}
}

func TestLoweringPreservesArrayIndexAssignmentSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; [code++][code++] = code++; return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 41 {
		t.Fatalf("expected array index assignment side effects return code 41, got %d", value)
	}
}

func TestLoweringPreservesMemberAssignmentReferenceValueSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; ({}).value = { value: code++ }; return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected member assignment reference value return code 21, got %d", value)
	}
}
