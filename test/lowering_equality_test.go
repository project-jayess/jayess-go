package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringSeparatesStrictBooleanEqualityFromNumbers(t *testing.T) {
	program := parseProgram(t, `function main() { if (true === 1) { return 1; } if (false !== 0) { return 31; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected strict boolean/number equality return code 31, got %d", value)
	}
}

func TestLoweringKeepsLooseBooleanNumberEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (true == 1 && false == 0) { return 32; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 32 {
		t.Fatalf("expected loose boolean/number equality return code 32, got %d", value)
	}
}

func TestLoweringSeparatesLooseBooleanNumberInequality(t *testing.T) {
	program := parseProgram(t, `function main() { if (true == 2) { return 1; } if (false != 1) { return 41; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 41 {
		t.Fatalf("expected loose boolean/number inequality return code 41, got %d", value)
	}
}

func TestLoweringSeparatesStrictStringNumberEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if ("1" === 1) { return 1; } if ("1" !== 1) { return 33; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 33 {
		t.Fatalf("expected strict string/number equality return code 33, got %d", value)
	}
}

func TestLoweringEvaluatesPrimitiveKindOperandOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if ((code++, "1") !== 1) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated primitive kind operand return code 21, got %d", value)
	}
}

func TestLoweringUsesLooseStringNumberEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if ("1" == 1 && "2" != 1) { return 35; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 35 {
		t.Fatalf("expected loose string/number equality return code 35, got %d", value)
	}
}

func TestLoweringEvaluatesLooseStringNumberOperandOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if ((code++, 1) == "1") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated loose string-number return code 21, got %d", value)
	}
}

func TestLoweringRejectsNonNumericLooseStringNumberEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if ("ready" == 1) { return 1; } if ("ready" != 1) { return 36; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 36 {
		t.Fatalf("expected non-numeric string/number equality return code 36, got %d", value)
	}
}

func TestLoweringUsesEmptyStringLooseNumberEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if ("" == 0 && "  " == 0) { return 42; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 42 {
		t.Fatalf("expected empty string loose number equality return code 42, got %d", value)
	}
}

func TestLoweringSeparatesStrictStringBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if ("true" === true) { return 1; } if ("true" !== true) { return 34; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 34 {
		t.Fatalf("expected strict string/boolean equality return code 34, got %d", value)
	}
}

func TestLoweringUsesLooseStringBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if ("1" == true && "0" == false) { return 37; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 37 {
		t.Fatalf("expected loose string/boolean equality return code 37, got %d", value)
	}
}

func TestLoweringRejectsNonNumericLooseStringBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if ("true" == true) { return 1; } if ("true" != true) { return 38; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 38 {
		t.Fatalf("expected non-numeric string/boolean equality return code 38, got %d", value)
	}
}

func TestLoweringEvaluatesLooseBooleanNumberOperandOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if ((code++, 1) == true) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated loose boolean-number return code 21, got %d", value)
	}
}

func TestLoweringUsesLooseNullishEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (null == undefined) { return 29; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 29 {
		t.Fatalf("expected loose nullish equality return code 29, got %d", value)
	}
}

func TestLoweringSeparatesLooseNullishBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (false == null) { return 1; } if (false != null) { return 39; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 39 {
		t.Fatalf("expected loose nullish/boolean equality return code 39, got %d", value)
	}
}

func TestLoweringSeparatesLooseNullishNumberEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (0 == undefined) { return 1; } if (0 != undefined) { return 40; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 40 {
		t.Fatalf("expected loose nullish/number equality return code 40, got %d", value)
	}
}

func TestLoweringSeparatesStrictNullishEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (null === undefined) { return 1; } if (null !== undefined) { return 30; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 30 {
		t.Fatalf("expected strict nullish equality return code 30, got %d", value)
	}
}
