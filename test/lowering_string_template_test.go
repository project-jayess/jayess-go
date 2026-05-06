package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringEvaluatesStringNumberCoercionOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = "value:" + (code++, 3); if (value === "value:3") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated string number coercion return code 21, got %d", value)
	}
}

func TestLoweringUsesStringNumberConcatenationInCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const value = "code:" + 7; if (value === "code:7") { return 71; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 71 {
		t.Fatalf("expected string number concatenation return code 71, got %d", value)
	}
}

func TestLoweringUsesNumberStringConcatenationInCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 7 + ":code"; if (value === "7:code") { return 72; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 72 {
		t.Fatalf("expected number string concatenation return code 72, got %d", value)
	}
}

func TestLoweringEvaluatesNumberStringConcatenationLeftToRight(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = (code++, 7) + (code++, ":code"); if (value === "7:code") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected left-to-right number string concatenation return code 31, got %d", value)
	}
}

func TestLoweringUsesStringPrimitiveConcatenationInCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const value = "ready:" + true + ":" + null + ":" + undefined; if (value === "ready:true:null:undefined") { return 73; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 73 {
		t.Fatalf("expected string primitive concatenation return code 73, got %d", value)
	}
}

func TestLoweringUsesConstantTemplateLiteralInCondition(t *testing.T) {
	program := parseProgram(t, "function main() { const value = `jayess`; if (value === \"jayess\") { return 76; } return 1; }")

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 76 {
		t.Fatalf("expected template literal return code 76, got %d", value)
	}
}

func TestLoweringUsesTemplateLiteralPrimitiveInterpolationInCondition(t *testing.T) {
	program := parseProgram(t, "function main() { const code = 7; const value = `code:${code}:${true}:${null}:${undefined}`; if (value === \"code:7:true:null:undefined\") { return 77; } return 1; }")

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 77 {
		t.Fatalf("expected template interpolation return code 77, got %d", value)
	}
}

func TestLoweringUsesTemplateLiteralTruthiness(t *testing.T) {
	program := parseProgram(t, "function main() { if (`ready`) { return 78; } return 1; }")

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 78 {
		t.Fatalf("expected template literal truthiness return code 78, got %d", value)
	}
}

func TestLoweringUsesEmptyTemplateLiteralAsFalsy(t *testing.T) {
	program := parseProgram(t, "function main() { if (``) { return 1; } return 79; }")

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 79 {
		t.Fatalf("expected empty template literal falsy return code 79, got %d", value)
	}
}

func TestLoweringUsesStringBinaryTruthiness(t *testing.T) {
	program := parseProgram(t, `function main() { if ("code:" + 7) { return 80; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 80 {
		t.Fatalf("expected string binary truthiness return code 80, got %d", value)
	}
}

func TestLoweringEvaluatesFoldedStringTruthinessOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if ((code++, "ready") + "") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected folded string truthiness return code 21, got %d", value)
	}
}
