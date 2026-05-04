package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesBooleanComputedObjectPropertyKey(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ [true]: 124 }).true === 124) { return 124; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 124 {
		t.Fatalf("expected boolean computed key return code 124, got %d", value)
	}
}

func TestLoweringUsesFalseComputedObjectPropertyKey(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ [false]: "no" })["false"] === "no") { return 125; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 125 {
		t.Fatalf("expected false computed key return code 125, got %d", value)
	}
}

func TestLoweringUsesNullComputedObjectPropertyKey(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ [null]: 126 }).null === 126) { return 126; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 126 {
		t.Fatalf("expected null computed key return code 126, got %d", value)
	}
}

func TestLoweringUsesUndefinedComputedObjectPropertyKey(t *testing.T) {
	program := parseProgram(t, `function main() { if (({ [undefined]: 127 }).undefined === 127) { return 127; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 127 {
		t.Fatalf("expected undefined computed key return code 127, got %d", value)
	}
}

func TestLoweringEvaluatesComputedObjectPropertyKeyOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (({ [code++]: 128 })[1] === 128) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated computed key return code 21, got %d", value)
	}
}
