package test

import "testing"

func TestLoweringUsesBigIntAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 0n; value = 12n; if (value === 12n) { return 158; } return 1; }`, 158, "BigInt assignment")
}

func TestLoweringUsesBigIntOrAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 0n; value ||= 12n; if (value === 12n) { return 159; } return 1; }`, 159, "BigInt or-assignment")
}

func TestLoweringKeepsTruthyBigIntOrAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; value ||= 13n; if (value === 12n) { return 160; } return 1; }`, 160, "truthy BigInt or-assignment")
}

func TestLoweringUsesTruthyBigIntAndAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; value &&= 13n; if (value === 13n) { return 161; } return 1; }`, 161, "truthy BigInt and-assignment")
}

func TestLoweringKeepsFalsyBigIntAndAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 0n; value &&= 13n; if (value === 0n) { return 162; } return 1; }`, 162, "falsy BigInt and-assignment")
}

func TestLoweringKeepsBigIntNullishAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; value ??= 13n; if (value === 12n) { return 163; } return 1; }`, 163, "BigInt nullish-assignment keep")
}

func TestLoweringUsesNullishToBigIntAssignment(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = undefined; value ??= 13n; if (value === 13n) { return 164; } return 1; }`, 164, "nullish-to-BigInt assignment")
}

func TestLoweringUsesConditionalBigIntValue(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = true ? 12n : 13n; if (value === 12n) { return 165; } return 1; }`, 165, "conditional BigInt")
}

func TestLoweringUsesCommaBigIntValueSideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var code = 1; const value = (code++, 12n); if (value === 12n) { return code * 100 + 66; } return 1; }`, 266, "comma BigInt side-effect")
}

func TestLoweringUsesNullishCoalesceBigIntValue(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = undefined ?? 12n; if (value === 12n) { return 167; } return 1; }`, 167, "nullish coalesce BigInt")
}

func TestLoweringKeepsNonNullishBigIntValue(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = 12n ?? 13n; if (value === 12n) { return 168; } return 1; }`, 168, "non-nullish BigInt")
}

func TestLoweringUsesTruthyBigIntLogicalAndValue(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = 12n && 13n; if (value === 13n) { return 169; } return 1; }`, 169, "truthy BigInt logical-and")
}

func TestLoweringKeepsFalsyBigIntLogicalAndValue(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = 0n && 13n; if (value === 0n) { return 170; } return 1; }`, 170, "falsy BigInt logical-and")
}

func TestLoweringKeepsTruthyBigIntLogicalOrValue(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = 12n || 13n; if (value === 12n) { return 171; } return 1; }`, 171, "truthy BigInt logical-or")
}

func TestLoweringUsesFalsyBigIntLogicalOrValue(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = 0n || 13n; if (value === 13n) { return 172; } return 1; }`, 172, "falsy BigInt logical-or")
}
