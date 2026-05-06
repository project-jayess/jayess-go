package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesUnaryNumericExpressionAsFalsyCondition(t *testing.T) {
	program := parseProgram(t, `function main() { if (-0) { return 1; } return 103; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 103 {
		t.Fatalf("expected unary numeric falsy return code 103, got %d", value)
	}
}

func TestLoweringUsesUnaryBitwiseExpressionAsTruthyCondition(t *testing.T) {
	program := parseProgram(t, `function main() { if (~0) { return 104; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 104 {
		t.Fatalf("expected unary bitwise truthy return code 104, got %d", value)
	}
}

func TestLoweringUsesUnaryPlusStringNumberReturn(t *testing.T) {
	program := parseProgram(t, `function main() { return +"7" + +true + +null; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 8 {
		t.Fatalf("expected unary plus primitive return code 8, got %d", value)
	}
}

func TestLoweringUsesUnaryNegateBooleanReturn(t *testing.T) {
	program := parseProgram(t, `function main() { return -true + 10; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 9 {
		t.Fatalf("expected unary negate boolean return code 9, got %d", value)
	}
}

func TestLoweringUsesUnaryPlusEmptyStringAsFalsyCondition(t *testing.T) {
	program := parseProgram(t, `function main() { if (+"") { return 1; } return 105; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 105 {
		t.Fatalf("expected unary plus empty string falsy return code 105, got %d", value)
	}
}

func TestLoweringExtractsReturnFromBitwiseExpression(t *testing.T) {
	program := parseProgram(t, `function main() { return (5 & 3) + (4 | 1) + (7 ^ 2); }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 11 {
		t.Fatalf("expected bitwise expression return code 11, got %d", value)
	}
}

func TestLoweringExtractsReturnFromUnaryBitwiseNot(t *testing.T) {
	program := parseProgram(t, `function main() { return ~5 + 10; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 4 {
		t.Fatalf("expected unary bitwise not return code 4, got %d", value)
	}
}

func TestLoweringExtractsReturnFromShiftExpression(t *testing.T) {
	program := parseProgram(t, `function main() { return (1 << 3) + (16 >> 2) + (16 >>> 3); }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 14 {
		t.Fatalf("expected shift expression return code 14, got %d", value)
	}
}

func TestLoweringUsesPrimitiveNumericCoercionForBitwiseExpression(t *testing.T) {
	program := parseProgram(t, `function main() { return ("7" & true) + ("4" | false) + (null ^ 3); }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 8 {
		t.Fatalf("expected bitwise coercion return code 8, got %d", value)
	}
}

func TestLoweringUsesPrimitiveNumericCoercionForShiftExpression(t *testing.T) {
	program := parseProgram(t, `function main() { return (true << "3") + ("16" >> true) + ("16" >>> "2"); }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 20 {
		t.Fatalf("expected shift coercion return code 20, got %d", value)
	}
}

func TestLoweringAppliesBitwiseAssignmentBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 6; code &= 3; code |= 8; code ^= 1; code <<= 1; code >>= 1; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 11 {
		t.Fatalf("expected bitwise assignment return code 11, got %d", value)
	}
}

func TestLoweringUsesPrimitiveNumericCoercionForBitwiseAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 6; code &= true; code |= "8"; code ^= null; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 8 {
		t.Fatalf("expected bitwise assignment coercion return code 8, got %d", value)
	}
}
