package test

import "testing"

func TestLoweringUsesLooseStringBigIntEquality(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if (12n == "12" && "13" != 12n) { return 197; } return 1; }`, 197, "loose string-BigInt equality")
}

func TestLoweringUsesLooseNumberBigIntEquality(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if (12n == 12 && 13 != 12n) { return 198; } return 1; }`, 198, "loose number-BigInt equality")
}

func TestLoweringUsesLooseBooleanBigIntEquality(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if (1n == true && false == 0n) { return 199; } return 1; }`, 199, "loose boolean-BigInt equality")
}

func TestLoweringUsesLooseBigIntEqualitySideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; if (value++ == "12" && value === 13n) { return 200; } return 1; }`, 200, "loose BigInt equality side-effect")
}

func TestLoweringUsesStringBigIntRelationalComparison(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if ("12" < 13n && 13n > "12") { return 201; } return 1; }`, 201, "string-BigInt relational")
}

func TestLoweringUsesNumberBigIntRelationalComparison(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if (12 < 13n && 13n >= 13) { return 202; } return 1; }`, 202, "number-BigInt relational")
}

func TestLoweringUsesBooleanBigIntRelationalComparison(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if (true <= 1n && false < 1n) { return 203; } return 1; }`, 203, "boolean-BigInt relational")
}

func TestLoweringUsesMixedBigIntRelationalSideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; if (value++ < "13" && value === 13n) { return 204; } return 1; }`, 204, "mixed BigInt relational side-effect")
}

func TestLoweringEvaluatesBigIntLooseEqualityRightSideForInvalidString(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; if ("not-bigint" != value++ && value === 13n) { return 214; } return 1; }`, 214, "invalid string loose BigInt side-effect")
}

func TestLoweringEvaluatesBigIntRelationalRightSideForInvalidString(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; if (!("not-bigint" < value++) && value === 13n) { return 215; } return 1; }`, 215, "invalid string relational BigInt side-effect")
}

func TestLoweringUsesNegativeBigIntRelationalOrder(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if (-13n < -12n && -12n > -13n) { return 219; } return 1; }`, 219, "negative BigInt relational order")
}
