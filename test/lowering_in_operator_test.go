package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesInOperatorForObjectProperty(t *testing.T) {
	program := parseProgram(t, `function main() { if ("value" in ({ value: 1 })) { return 132; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 132 {
		t.Fatalf("expected object in-operator return code 132, got %d", value)
	}
}

func TestLoweringUsesInOperatorForMissingObjectProperty(t *testing.T) {
	program := parseProgram(t, `function main() { if ("missing" in ({ value: 1 })) { return 1; } return 133; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 133 {
		t.Fatalf("expected missing object in-operator return code 133, got %d", value)
	}
}

func TestLoweringEvaluatesInOperatorLeftAndRightOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if ((code++ + "") in ({ "1": code++ })) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected single-evaluated in-operator return code 31, got %d", value)
	}
}

func TestLoweringUsesInOperatorComputedObjectProperty(t *testing.T) {
	program := parseProgram(t, `function main() { if ("name" in ({ ["na" + "me"]: "jayess" })) { return 134; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 134 {
		t.Fatalf("expected computed object in-operator return code 134, got %d", value)
	}
}
