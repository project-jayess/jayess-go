package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesDeleteMemberAsTrue(t *testing.T) {
	program := parseProgram(t, `function main() { if (delete ({ value: 1 }).value) { return 129; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 129 {
		t.Fatalf("expected delete member return code 129, got %d", value)
	}
}

func TestLoweringUsesDeleteIndexAsTrue(t *testing.T) {
	program := parseProgram(t, `function main() { if (delete [1, 2][0]) { return 130; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 130 {
		t.Fatalf("expected delete index return code 130, got %d", value)
	}
}

func TestLoweringEvaluatesDeleteIndexKeyOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (delete ({ value: 1 })[code++]) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated delete key return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesDeleteTargetObjectLiteralOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (delete ({ value: code++ }).value) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated delete target return code 21, got %d", value)
	}
}

func TestLoweringUsesDeleteStrictBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if ((delete ({ value: 1 }).value) === true) { return 131; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 131 {
		t.Fatalf("expected delete strict boolean equality return code 131, got %d", value)
	}
}
