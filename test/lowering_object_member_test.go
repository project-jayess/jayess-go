package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesObjectNumericMemberExpression(t *testing.T) {
	program := parseProgram(t, `function main() { return ({ value: 7 }).value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected object numeric member return code 7, got %d", value)
	}
}

func TestLoweringUsesObjectStringIndexExpression(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ name: "jayess" })["name"] === "jayess") { return 49; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 49 {
		t.Fatalf("expected object string index return code 49, got %d", value)
	}
}

func TestLoweringUsesObjectBooleanMemberExpression(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ ok: true }).ok) { return 50; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 50 {
		t.Fatalf("expected object boolean member return code 50, got %d", value)
	}
}

func TestLoweringUsesComputedObjectPropertyExpression(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ ["na" + "me"]: "jayess" }).name === "jayess") { return 51; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 51 {
		t.Fatalf("expected computed object property return code 51, got %d", value)
	}
}

func TestLoweringUsesLastDuplicateObjectProperty(t *testing.T) {
	program := parseProgram(t, `function main() { return ({ value: 1, value: 8 }).value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 8 {
		t.Fatalf("expected duplicate object property return code 8, got %d", value)
	}
}

func TestLoweringTreatsMissingObjectPropertyAsUndefined(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ value: 1 }).missing === undefined) { return 52; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 52 {
		t.Fatalf("expected missing object property return code 52, got %d", value)
	}
}

func TestLoweringEvaluatesObjectPropertyValuesOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (({ first: code++, second: code }).second === 2) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated object property return code 21, got %d", value)
	}
}
