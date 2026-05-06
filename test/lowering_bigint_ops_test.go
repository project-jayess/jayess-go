package test

import "testing"

func TestLoweringUsesBigIntAddition(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = 12n + 13n; if (value === 25n) { return 177; } return 1; }`, 177, "BigInt addition")
}

func TestLoweringUsesBigIntSubtraction(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = 12n - 13n; if (value === -1n) { return 178; } return 1; }`, 178, "BigInt subtraction")
}

func TestLoweringUsesBigIntMultiplication(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = 12n * 13n; if (value === 156n) { return 179; } return 1; }`, 179, "BigInt multiplication")
}

func TestLoweringUsesBigIntDivisionAndRemainder(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const div = 13n / 5n; const rem = 13n % 5n; if (div === 2n && rem === 3n) { return 180; } return 1; }`, 180, "BigInt division/remainder")
}

func TestLoweringUsesBigIntExponentiation(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = 2n ** 8n; if (value === 256n) { return 181; } return 1; }`, 181, "BigInt exponentiation")
}

func TestLoweringUsesBigIntAddAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; value += 13n; if (value === 25n) { return 182; } return 1; }`, 182, "BigInt add-assignment")
}

func TestLoweringUsesBigIntMultiplyAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; value *= 13n; if (value === 156n) { return 183; } return 1; }`, 183, "BigInt multiply-assignment")
}

func TestLoweringUsesBigIntDivideRemainderAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var div = 13n; var rem = 13n; div /= 5n; rem %= 5n; if (div === 2n && rem === 3n) { return 184; } return 1; }`, 184, "BigInt divide/remainder-assignment")
}

func TestLoweringUsesBigIntPowerAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 2n; value **= 8n; if (value === 256n) { return 185; } return 1; }`, 185, "BigInt power-assignment")
}

func TestLoweringUsesBigIntBitwiseOperators(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = (12n & 10n) | 1n; if (value === 9n) { return 186; } return 1; }`, 186, "BigInt bitwise")
}

func TestLoweringUsesBigIntXorAndShiftOperators(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = (5n ^ 3n) << 2n; if (value === 24n) { return 187; } return 1; }`, 187, "BigInt xor/shift")
}

func TestLoweringUsesBigIntRightShiftOperator(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = 64n >> 3n; if (value === 8n) { return 188; } return 1; }`, 188, "BigInt right-shift")
}

func TestLoweringUsesBigIntBitwiseAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; value &= 10n; value |= 1n; if (value === 9n) { return 189; } return 1; }`, 189, "BigInt bitwise-assignment")
}

func TestLoweringUsesBigIntShiftAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 5n; value ^= 3n; value <<= 2n; value >>= 1n; if (value === 12n) { return 190; } return 1; }`, 190, "BigInt shift-assignment")
}

func TestLoweringUsesBigIntBitwiseNotLiteral(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if (~12n === -13n) { return 191; } return 1; }`, 191, "BigInt bitwise-not literal")
}

func TestLoweringUsesBigIntBitwiseNotBinding(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = 12n; if (~value === -13n) { return 192; } return 1; }`, 192, "BigInt bitwise-not binding")
}

func TestLoweringUsesBigIntPostfixUpdateStatement(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; value++; if (value === 13n) { return 193; } return 1; }`, 193, "BigInt postfix update")
}

func TestLoweringUsesBigIntPrefixUpdateStatement(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; ++value; if (value === 13n) { return 194; } return 1; }`, 194, "BigInt prefix update")
}

func TestLoweringUsesBigIntPostfixUpdateValue(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; const old = value++; if (old === 12n && value === 13n) { return 195; } return 1; }`, 195, "BigInt postfix update value")
}

func TestLoweringUsesBigIntPrefixDecrementValue(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; const next = --value; if (next === 11n && value === 11n) { return 196; } return 1; }`, 196, "BigInt prefix decrement value")
}
