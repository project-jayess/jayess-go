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

func TestLoweringUsesBigIntAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 0n; value = 12n; if (value === 12n) { return 158; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 158 {
		t.Fatalf("expected BigInt assignment return code 158, got %d", value)
	}
}

func TestLoweringUsesBigIntOrAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 0n; value ||= 12n; if (value === 12n) { return 159; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 159 {
		t.Fatalf("expected BigInt or-assignment return code 159, got %d", value)
	}
}

func TestLoweringKeepsTruthyBigIntOrAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; value ||= 13n; if (value === 12n) { return 160; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 160 {
		t.Fatalf("expected truthy BigInt or-assignment return code 160, got %d", value)
	}
}

func TestLoweringUsesTruthyBigIntAndAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; value &&= 13n; if (value === 13n) { return 161; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 161 {
		t.Fatalf("expected truthy BigInt and-assignment return code 161, got %d", value)
	}
}

func TestLoweringKeepsFalsyBigIntAndAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 0n; value &&= 13n; if (value === 0n) { return 162; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 162 {
		t.Fatalf("expected falsy BigInt and-assignment return code 162, got %d", value)
	}
}

func TestLoweringKeepsBigIntNullishAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; value ??= 13n; if (value === 12n) { return 163; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 163 {
		t.Fatalf("expected BigInt nullish-assignment keep return code 163, got %d", value)
	}
}

func TestLoweringUsesNullishToBigIntAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var value = undefined; value ??= 13n; if (value === 13n) { return 164; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 164 {
		t.Fatalf("expected nullish-to-BigInt assignment return code 164, got %d", value)
	}
}

func TestLoweringUsesConditionalBigIntValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = true ? 12n : 13n; if (value === 12n) { return 165; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 165 {
		t.Fatalf("expected conditional BigInt return code 165, got %d", value)
	}
}

func TestLoweringUsesCommaBigIntValueSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = (code++, 12n); if (value === 12n) { return code * 100 + 66; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 266 {
		t.Fatalf("expected comma BigInt side-effect return code 266, got %d", value)
	}
}

func TestLoweringUsesNullishCoalesceBigIntValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = undefined ?? 12n; if (value === 12n) { return 167; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 167 {
		t.Fatalf("expected nullish coalesce BigInt return code 167, got %d", value)
	}
}

func TestLoweringKeepsNonNullishBigIntValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 12n ?? 13n; if (value === 12n) { return 168; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 168 {
		t.Fatalf("expected non-nullish BigInt return code 168, got %d", value)
	}
}

func TestLoweringUsesTruthyBigIntLogicalAndValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 12n && 13n; if (value === 13n) { return 169; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 169 {
		t.Fatalf("expected truthy BigInt logical-and return code 169, got %d", value)
	}
}

func TestLoweringKeepsFalsyBigIntLogicalAndValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 0n && 13n; if (value === 0n) { return 170; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 170 {
		t.Fatalf("expected falsy BigInt logical-and return code 170, got %d", value)
	}
}

func TestLoweringKeepsTruthyBigIntLogicalOrValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 12n || 13n; if (value === 12n) { return 171; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 171 {
		t.Fatalf("expected truthy BigInt logical-or return code 171, got %d", value)
	}
}

func TestLoweringUsesFalsyBigIntLogicalOrValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 0n || 13n; if (value === 13n) { return 172; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 172 {
		t.Fatalf("expected falsy BigInt logical-or return code 172, got %d", value)
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

func TestLoweringUsesBigIntAddition(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 12n + 13n; if (value === 25n) { return 177; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 177 {
		t.Fatalf("expected BigInt addition return code 177, got %d", value)
	}
}

func TestLoweringUsesBigIntSubtraction(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 12n - 13n; if (value === -1n) { return 178; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 178 {
		t.Fatalf("expected BigInt subtraction return code 178, got %d", value)
	}
}

func TestLoweringUsesBigIntMultiplication(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 12n * 13n; if (value === 156n) { return 179; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 179 {
		t.Fatalf("expected BigInt multiplication return code 179, got %d", value)
	}
}

func TestLoweringUsesBigIntDivisionAndRemainder(t *testing.T) {
	program := parseProgram(t, `function main() { const div = 13n / 5n; const rem = 13n % 5n; if (div === 2n && rem === 3n) { return 180; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 180 {
		t.Fatalf("expected BigInt division/remainder return code 180, got %d", value)
	}
}

func TestLoweringUsesBigIntExponentiation(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 2n ** 8n; if (value === 256n) { return 181; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 181 {
		t.Fatalf("expected BigInt exponentiation return code 181, got %d", value)
	}
}

func TestLoweringUsesBigIntAddAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; value += 13n; if (value === 25n) { return 182; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 182 {
		t.Fatalf("expected BigInt add-assignment return code 182, got %d", value)
	}
}

func TestLoweringUsesBigIntMultiplyAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; value *= 13n; if (value === 156n) { return 183; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 183 {
		t.Fatalf("expected BigInt multiply-assignment return code 183, got %d", value)
	}
}

func TestLoweringUsesBigIntDivideRemainderAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var div = 13n; var rem = 13n; div /= 5n; rem %= 5n; if (div === 2n && rem === 3n) { return 184; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 184 {
		t.Fatalf("expected BigInt divide/remainder-assignment return code 184, got %d", value)
	}
}

func TestLoweringUsesBigIntPowerAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 2n; value **= 8n; if (value === 256n) { return 185; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 185 {
		t.Fatalf("expected BigInt power-assignment return code 185, got %d", value)
	}
}

func TestLoweringUsesBigIntBitwiseOperators(t *testing.T) {
	program := parseProgram(t, `function main() { const value = (12n & 10n) | 1n; if (value === 9n) { return 186; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 186 {
		t.Fatalf("expected BigInt bitwise return code 186, got %d", value)
	}
}

func TestLoweringUsesBigIntXorAndShiftOperators(t *testing.T) {
	program := parseProgram(t, `function main() { const value = (5n ^ 3n) << 2n; if (value === 24n) { return 187; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 187 {
		t.Fatalf("expected BigInt xor/shift return code 187, got %d", value)
	}
}

func TestLoweringUsesBigIntRightShiftOperator(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 64n >> 3n; if (value === 8n) { return 188; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 188 {
		t.Fatalf("expected BigInt right-shift return code 188, got %d", value)
	}
}

func TestLoweringUsesBigIntBitwiseAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; value &= 10n; value |= 1n; if (value === 9n) { return 189; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 189 {
		t.Fatalf("expected BigInt bitwise-assignment return code 189, got %d", value)
	}
}

func TestLoweringUsesBigIntShiftAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 5n; value ^= 3n; value <<= 2n; value >>= 1n; if (value === 12n) { return 190; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 190 {
		t.Fatalf("expected BigInt shift-assignment return code 190, got %d", value)
	}
}

func TestLoweringUsesBigIntBitwiseNotLiteral(t *testing.T) {
	program := parseProgram(t, `function main() { if (~12n === -13n) { return 191; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 191 {
		t.Fatalf("expected BigInt bitwise-not literal return code 191, got %d", value)
	}
}

func TestLoweringUsesBigIntBitwiseNotBinding(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 12n; if (~value === -13n) { return 192; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 192 {
		t.Fatalf("expected BigInt bitwise-not binding return code 192, got %d", value)
	}
}

func TestLoweringUsesBigIntPostfixUpdateStatement(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; value++; if (value === 13n) { return 193; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 193 {
		t.Fatalf("expected BigInt postfix update return code 193, got %d", value)
	}
}

func TestLoweringUsesBigIntPrefixUpdateStatement(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; ++value; if (value === 13n) { return 194; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 194 {
		t.Fatalf("expected BigInt prefix update return code 194, got %d", value)
	}
}

func TestLoweringUsesBigIntPostfixUpdateValue(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; const old = value++; if (old === 12n && value === 13n) { return 195; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 195 {
		t.Fatalf("expected BigInt postfix update value return code 195, got %d", value)
	}
}

func TestLoweringUsesBigIntPrefixDecrementValue(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; const next = --value; if (next === 11n && value === 11n) { return 196; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 196 {
		t.Fatalf("expected BigInt prefix decrement value return code 196, got %d", value)
	}
}

func TestLoweringUsesLooseStringBigIntEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (12n == "12" && "13" != 12n) { return 197; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 197 {
		t.Fatalf("expected loose string-BigInt equality return code 197, got %d", value)
	}
}

func TestLoweringUsesLooseNumberBigIntEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (12n == 12 && 13 != 12n) { return 198; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 198 {
		t.Fatalf("expected loose number-BigInt equality return code 198, got %d", value)
	}
}

func TestLoweringUsesLooseBooleanBigIntEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (1n == true && false == 0n) { return 199; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 199 {
		t.Fatalf("expected loose boolean-BigInt equality return code 199, got %d", value)
	}
}

func TestLoweringUsesLooseBigIntEqualitySideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; if (value++ == "12" && value === 13n) { return 200; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 200 {
		t.Fatalf("expected loose BigInt equality side-effect return code 200, got %d", value)
	}
}

func TestLoweringUsesStringBigIntRelationalComparison(t *testing.T) {
	program := parseProgram(t, `function main() { if ("12" < 13n && 13n > "12") { return 201; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 201 {
		t.Fatalf("expected string-BigInt relational return code 201, got %d", value)
	}
}

func TestLoweringUsesNumberBigIntRelationalComparison(t *testing.T) {
	program := parseProgram(t, `function main() { if (12 < 13n && 13n >= 13) { return 202; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 202 {
		t.Fatalf("expected number-BigInt relational return code 202, got %d", value)
	}
}

func TestLoweringUsesBooleanBigIntRelationalComparison(t *testing.T) {
	program := parseProgram(t, `function main() { if (true <= 1n && false < 1n) { return 203; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 203 {
		t.Fatalf("expected boolean-BigInt relational return code 203, got %d", value)
	}
}

func TestLoweringUsesMixedBigIntRelationalSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; if (value++ < "13" && value === 13n) { return 204; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 204 {
		t.Fatalf("expected mixed BigInt relational side-effect return code 204, got %d", value)
	}
}

func TestLoweringUsesBigIntSwitchCase(t *testing.T) {
	program := parseProgram(t, `function main() { const kind = 12n; switch (kind) { case 11n: return 1; case 12n: return 205; default: return 2; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 205 {
		t.Fatalf("expected BigInt switch case return code 205, got %d", value)
	}
}

func TestLoweringUsesBigIntSwitchDefault(t *testing.T) {
	program := parseProgram(t, `function main() { const kind = 12n; switch (kind) { case 10n: return 1; case 11n: return 2; default: return 206; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 206 {
		t.Fatalf("expected BigInt switch default return code 206, got %d", value)
	}
}

func TestLoweringUsesBigIntSwitchDiscriminantSideEffect(t *testing.T) {
	program := parseProgram(t, `function main() { var kind = 12n; switch (kind++) { case 12n: if (kind === 13n) { return 207; } return 1; default: return 2; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 207 {
		t.Fatalf("expected BigInt switch side-effect return code 207, got %d", value)
	}
}

func TestLoweringUsesBigIntArrayIndex(t *testing.T) {
	program := parseProgram(t, `function main() { const values = [11n, 12n]; if (values[1] === 12n) { return 208; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 208 {
		t.Fatalf("expected BigInt array index return code 208, got %d", value)
	}
}

func TestLoweringUsesBigIntObjectMember(t *testing.T) {
	program := parseProgram(t, `function main() { const value = { code: 12n }; if (value.code === 12n) { return 209; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 209 {
		t.Fatalf("expected BigInt object member return code 209, got %d", value)
	}
}

func TestLoweringUsesBigIntObjectIndex(t *testing.T) {
	program := parseProgram(t, `function main() { const value = { code: 12n }; if (value["code"] === 12n) { return 210; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 210 {
		t.Fatalf("expected BigInt object index return code 210, got %d", value)
	}
}

func TestLoweringUsesBigIntElementTruthiness(t *testing.T) {
	program := parseProgram(t, `function main() { const values = [0n, 12n]; if (!values[0] && values[1]) { return 211; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 211 {
		t.Fatalf("expected BigInt element truthiness return code 211, got %d", value)
	}
}

func TestLoweringUsesBigIntPrimitiveInstanceof(t *testing.T) {
	program := parseProgram(t, `function main() { if (!(12n instanceof function Value() {})) { return 212; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 212 {
		t.Fatalf("expected BigInt primitive instanceof return code 212, got %d", value)
	}
}

func TestLoweringUsesBigIntInstanceofLeftSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; if (!(value++ instanceof function Value() {}) && value === 13n) { return 213; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 213 {
		t.Fatalf("expected BigInt instanceof side-effect return code 213, got %d", value)
	}
}

func TestLoweringEvaluatesBigIntLooseEqualityRightSideForInvalidString(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; if ("not-bigint" != value++ && value === 13n) { return 214; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 214 {
		t.Fatalf("expected invalid string loose BigInt side-effect return code 214, got %d", value)
	}
}

func TestLoweringEvaluatesBigIntRelationalRightSideForInvalidString(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; if (!("not-bigint" < value++) && value === 13n) { return 215; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 215 {
		t.Fatalf("expected invalid string relational BigInt side-effect return code 215, got %d", value)
	}
}

func TestLoweringUsesBigIntComputedObjectPropertyKey(t *testing.T) {
	program := parseProgram(t, `function main() { const value = { [12n]: 216 }; if (value[12n] === 216) { return 216; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 216 {
		t.Fatalf("expected BigInt computed object key return code 216, got %d", value)
	}
}

func TestLoweringUsesBigIntInOperatorKey(t *testing.T) {
	program := parseProgram(t, `function main() { if (12n in ({ [12n]: "value" })) { return 217; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 217 {
		t.Fatalf("expected BigInt in-operator key return code 217, got %d", value)
	}
}

func TestLoweringEvaluatesBigIntComputedObjectPropertyKeyOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var key = 12n; if (({ [key++]: 218 })[12n] === 218 && key === 13n) { return 218; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 218 {
		t.Fatalf("expected BigInt computed key side-effect return code 218, got %d", value)
	}
}

func TestLoweringUsesNegativeBigIntRelationalOrder(t *testing.T) {
	program := parseProgram(t, `function main() { if (-13n < -12n && -12n > -13n) { return 219; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 219 {
		t.Fatalf("expected negative BigInt relational order return code 219, got %d", value)
	}
}

func TestLoweringUsesBigIntArraySpreadIndex(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const values = [10n, ...[11n, 12n]]; if (values[2] === 12n) { return 220; } return 1; }`, 220, "BigInt array spread index")
}

func TestLoweringUsesBigIntObjectSpreadMember(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = { ...{ code: 12n } }; if (value.code === 12n) { return 221; } return 1; }`, 221, "BigInt object spread member")
}

func TestLoweringUsesBigIntSpreadSideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; const values = [...[value++]]; if (values[0] === 12n && value === 13n) { return 222; } return 1; }`, 222, "BigInt spread side-effect")
}

func TestLoweringUsesBigIntVoidSideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; const ignored = void value++; if (ignored === undefined && value === 13n) { return 223; } return 1; }`, 223, "BigInt void side-effect")
}

func TestLoweringUsesBigIntDeleteTargetSideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; if (delete value++ && value === 13n) { return 224; } return 1; }`, 224, "BigInt delete target side-effect")
}

func TestLoweringUsesBigIntDeleteIndexKeySideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var key = 12n; if (delete ({ [12n]: "value" })[key++] && key === 13n) { return 225; } return 1; }`, 225, "BigInt delete index-key side-effect")
}

func TestLoweringUsesBigIntArrayIndexKey(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const values = [10n, 11n]; if (values[1n] === 11n) { return 226; } return 1; }`, 226, "BigInt array index key")
}

func TestLoweringUsesBigIntArrayIndexSideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var index = 1n; if ([10n, 11n][index++] === 11n && index === 2n) { return 227; } return 1; }`, 227, "BigInt array index side-effect")
}

func TestLoweringUsesBigIntStringIndexKey(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if ("ab"[1n] === "b") { return 228; } return 1; }`, 228, "BigInt string index key")
}

func TestLoweringUsesHugeBigIntArrayIndexAsUndefined(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if ([10n][999999999999999999999999999999n] === undefined) { return 229; } return 1; }`, 229, "huge BigInt array index")
}

func TestLoweringUsesHugeBigIntStringIndexAsUndefined(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if ("ab"[999999999999999999999999999999n] === undefined) { return 230; } return 1; }`, 230, "huge BigInt string index")
}
