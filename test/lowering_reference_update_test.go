package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringPreservesMemberUpdateTargetSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; ({ value: code++ }).value++; return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected member update side effects return code 21, got %d", value)
	}
}

func TestLoweringPreservesIndexUpdateTargetKeySideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; ({ value: code++ })[code++]++; return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected index update side effects return code 31, got %d", value)
	}
}

func TestLoweringPreservesArrayIndexUpdateSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; [code++][code++]++; return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected array index update side effects return code 31, got %d", value)
	}
}

func TestLoweringPreservesPrefixMemberUpdateTargetSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; ++({ value: code++ }).value; return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected prefix member update side effects return code 21, got %d", value)
	}
}
