package test

import (
	"testing"

	"jayess-go/lowering"
)

func expectBigIntReturnCode(t *testing.T, source string, want int, context string) {
	t.Helper()

	program := parseProgram(t, source)
	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != want {
		t.Fatalf("expected %s return code %d, got %d", context, want, value)
	}
}

func TestLoweringUsesBigIntTruthyLiteral(t *testing.T) {
	program := parseProgram(t, `function main() { if (12n) { return 146; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 146 {
		t.Fatalf("expected BigInt truthy return code 146, got %d", value)
	}
}

func TestLoweringUsesZeroBigIntFalsyLiteral(t *testing.T) {
	program := parseProgram(t, `function main() { if (0n) { return 1; } return 147; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 147 {
		t.Fatalf("expected zero BigInt falsy return code 147, got %d", value)
	}
}

func TestLoweringUsesTypeofBigIntLiteral(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof 12n === "bigint") { return 148; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 148 {
		t.Fatalf("expected typeof BigInt return code 148, got %d", value)
	}
}

func TestLoweringUsesBigIntTemplateInterpolation(t *testing.T) {
	program := parseProgram(t, "function main() { if (`value ${12n}` === \"value 12\") { return 149; } return 1; }")

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 149 {
		t.Fatalf("expected BigInt template interpolation return code 149, got %d", value)
	}
}

func TestLoweringUsesBigIntStrictEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (12n === 12n) { return 150; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 150 {
		t.Fatalf("expected BigInt strict equality return code 150, got %d", value)
	}
}

func TestLoweringUsesBigIntStrictInequality(t *testing.T) {
	program := parseProgram(t, `function main() { if (12n !== 13n) { return 151; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 151 {
		t.Fatalf("expected BigInt strict inequality return code 151, got %d", value)
	}
}

func TestLoweringUsesBigIntNumberStrictMismatch(t *testing.T) {
	program := parseProgram(t, `function main() { if (12n !== 12) { return 152; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 152 {
		t.Fatalf("expected BigInt number strict mismatch return code 152, got %d", value)
	}
}

func TestLoweringUsesBigIntRelationalComparison(t *testing.T) {
	program := parseProgram(t, `function main() { if (12n < 13n) { return 153; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 153 {
		t.Fatalf("expected BigInt relational comparison return code 153, got %d", value)
	}
}

func TestLoweringUsesBigIntRelationalComparisonByMagnitude(t *testing.T) {
	program := parseProgram(t, `function main() { if (100000000000000000000n > 99999999999999999999n) { return 154; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 154 {
		t.Fatalf("expected BigInt magnitude comparison return code 154, got %d", value)
	}
}

func TestLoweringUsesBigIntVariableEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 12n; if (value === 12n) { return 155; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 155 {
		t.Fatalf("expected BigInt variable equality return code 155, got %d", value)
	}
}

func TestLoweringUsesBigIntVariableTypeof(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 12n; if (typeof value === "bigint") { return 156; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 156 {
		t.Fatalf("expected BigInt variable typeof return code 156, got %d", value)
	}
}

func TestLoweringUsesBigIntVariableStringCoercion(t *testing.T) {
	program := parseProgram(t, "function main() { const value = 12n; if (`value ${value}` === \"value 12\") { return 157; } return 1; }")

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 157 {
		t.Fatalf("expected BigInt variable string coercion return code 157, got %d", value)
	}
}

func TestLoweringUsesNegativeBigIntEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (-12n === -12n) { return 173; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 173 {
		t.Fatalf("expected negative BigInt equality return code 173, got %d", value)
	}
}

func TestLoweringUsesNegativeBigIntRelationalComparison(t *testing.T) {
	program := parseProgram(t, `function main() { if (-13n < -12n) { return 174; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 174 {
		t.Fatalf("expected negative BigInt relational return code 174, got %d", value)
	}
}

func TestLoweringUsesNegativeBigIntTypeofAndCoercion(t *testing.T) {
	program := parseProgram(t, "function main() { if (typeof -12n === \"bigint\" && `${-12n}` === \"-12\") { return 175; } return 1; }")

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 175 {
		t.Fatalf("expected negative BigInt typeof/coercion return code 175, got %d", value)
	}
}

func TestLoweringUsesNegativeBigIntTruthiness(t *testing.T) {
	program := parseProgram(t, `function main() { if (-12n) { return 176; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 176 {
		t.Fatalf("expected negative BigInt truthiness return code 176, got %d", value)
	}
}
