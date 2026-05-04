package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesObjectLiteralSpreadMember(t *testing.T) {
	program := parseProgram(t, `function main() { return ({ ...{ value: 142 } }).value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 142 {
		t.Fatalf("expected object spread member return code 142, got %d", value)
	}
}

func TestLoweringUsesLaterObjectSpreadProperty(t *testing.T) {
	program := parseProgram(t, `function main() { return ({ value: 1, ...{ value: 143 } }).value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 143 {
		t.Fatalf("expected later object spread return code 143, got %d", value)
	}
}

func TestLoweringUsesLaterObjectPropertyAfterSpread(t *testing.T) {
	program := parseProgram(t, `function main() { return ({ ...{ value: 1 }, value: 144 }).value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 144 {
		t.Fatalf("expected later property after spread return code 144, got %d", value)
	}
}

func TestLoweringEvaluatesObjectLiteralSpreadInOrder(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (({ first: code++, ...{ second: code++ }, third: code }).third === 3) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected ordered object spread return code 31, got %d", value)
	}
}

func TestLoweringUsesObjectLiteralSpreadComputedKey(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ ...{ ["na" + "me"]: "jayess" } }).name === "jayess") { return 145; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 145 {
		t.Fatalf("expected object spread computed key return code 145, got %d", value)
	}
}
