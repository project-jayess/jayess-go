package test

import (
	"testing"

	"jayess-go/lowering"
)

func expectTryReturnCode(t *testing.T, source string, want int, context string) {
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

func expectNoTryReturnCode(t *testing.T, source string, context string) {
	t.Helper()

	program := parseProgram(t, source)
	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatalf("expected %s to stay unresolved", context)
	}
}

func TestLoweringDoesNotFoldReturnAfterThrow(t *testing.T) {
	expectNoTryReturnCode(t, `function main() { throw value; return 2; }`, "return after throw")
}

func TestLoweringDoesNotFoldReturnAfterBlockThrow(t *testing.T) {
	expectNoTryReturnCode(t, `function main() { { throw value; } return 2; }`, "return after block throw")
}

func TestLoweringDoesNotFoldReturnAfterTryMayThrow(t *testing.T) {
	expectNoTryReturnCode(t, `function main() { try { throw value; } finally { cleanup(); } return 2; }`, "return after try throw")
}

func TestLoweringDoesNotFoldReturnAfterCatchMayReturn(t *testing.T) {
	expectNoTryReturnCode(t, `function main() { try { run(); } catch (err) { return value; } return 2; }`, "return after catch return")
}

func TestLoweringDoesNotFoldReturnAfterFinallyMayReturn(t *testing.T) {
	expectNoTryReturnCode(t, `function main() { try { run(); } finally { return value; } return 2; }`, "return after finally return")
}

func TestLoweringAppliesTrySideEffectsBeforeReturn(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { code = 41; } catch (err) { code = 2; } return code; }`, 41, "try side-effect")
}

func TestLoweringAppliesFinallySideEffectsBeforeReturn(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { code = 42; } finally { code++; } return code; }`, 43, "finally side-effect")
}

func TestLoweringStopsTrySideEffectsAtBreakBeforeReturn(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; while (true) { try { code = 44; break; code = 2; } finally { code++; } } return code; }`, 45, "try break/finally side-effect")
}

func TestLoweringAppliesCatchSideEffectsAfterDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { code = 46; throw value; code = 2; } catch (err) { code++; } return code; }`, 47, "catch side-effect")
}

func TestLoweringAppliesCatchAndFinallySideEffectsAfterDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { code = 48; throw value; code = 2; } catch (err) { code++; } finally { code++; } return code; }`, 50, "catch and finally side-effect")
}

func TestLoweringAppliesCatchSideEffectsAfterNestedBlockDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { { code = 53; throw value; code = 2; } } catch (err) { code++; } return code; }`, 54, "nested block throw catch side-effect")
}

func TestLoweringAppliesCatchSideEffectsAfterConstantIfDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { if (true) { code = 55; throw value; code = 2; } else { code = 3; } } catch (err) { code++; } return code; }`, 56, "constant if throw catch side-effect")
}

func TestLoweringAppliesCatchSideEffectsAfterConstantElseDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { if (false) { code = 3; } else { code = 57; throw value; code = 2; } } catch (err) { code++; } return code; }`, 58, "constant else throw catch side-effect")
}

func TestLoweringAppliesCatchSideEffectsAfterConstantSwitchDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { switch (2) { case 1: code = 3; break; case 2: code = 59; throw value; code = 2; default: code = 4; } } catch (err) { code++; } return code; }`, 60, "constant switch throw catch side-effect")
}

func TestLoweringAppliesCatchSideEffectsAfterConstantWhileDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { while (true) { code = 61; throw value; code = 2; } } catch (err) { code++; } return code; }`, 62, "constant while throw catch side-effect")
}

func TestLoweringAppliesCatchSideEffectsAfterDoWhileDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { do { code = 63; throw value; code = 2; } while (false); } catch (err) { code++; } return code; }`, 64, "do while throw catch side-effect")
}

func TestLoweringAppliesCatchSideEffectsAfterConstantForDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { for (code = 65; true; code++) { throw value; code = 2; } } catch (err) { code++; } return code; }`, 66, "constant for throw catch side-effect")
}

func TestLoweringAppliesCatchSideEffectsAfterInfiniteForDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { for (code = 67;; code++) { throw value; code = 2; } } catch (err) { code++; } return code; }`, 68, "infinite for throw catch side-effect")
}

func TestLoweringDoesNotTreatUnreachableLoopThrowAsDefinite(t *testing.T) {
	expectNoTryReturnCode(t, `function main() { var code = 1; try { while (true) { code = 69; break; throw value; } } catch (err) { code++; } return code; }`, "unreachable loop throw catch path")
}

func TestLoweringAppliesForInitializerBeforeLaterDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { for (code = 70; false; code++) { code = 2; } throw value; } catch (err) { code++; } return code; }`, 71, "for initializer before later throw")
}

func TestLoweringAppliesCatchSideEffectsAfterLabeledBlockDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { entry: { code = 72; throw value; code = 2; } } catch (err) { code++; } return code; }`, 73, "labeled block throw catch side-effect")
}

func TestLoweringAppliesCatchSideEffectsAfterNestedTryFinallyDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { try { code = 74; throw value; code = 2; } finally { code++; } } catch (err) { code++; } return code; }`, 76, "nested try/finally throw catch side-effect")
}

func TestLoweringAppliesCatchSideEffectsAfterNestedFinallyDefiniteThrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { try { code = 80; } finally { code++; throw value; } } catch (err) { code++; } return code; }`, 82, "nested finally throw side-effect")
}

func TestLoweringAppliesCatchSideEffectsAfterNestedTryCatchRethrow(t *testing.T) {
	expectTryReturnCode(t, `function main() { var code = 1; try { try { code = 77; throw value; } catch (err) { code++; throw value; } } catch (err) { code++; } return code; }`, 79, "nested try/catch rethrow side-effect")
}

func TestLoweringDoesNotFoldReturnAfterDefiniteThrowCatchMayReturn(t *testing.T) {
	expectNoTryReturnCode(t, `function main() { var code = 1; try { code = 51; throw value; } catch (err) { return value; } return code; }`, "return after definite throw catch return")
}

func TestLoweringDoesNotFoldReturnAfterDefiniteThrowFinallyMayReturn(t *testing.T) {
	expectNoTryReturnCode(t, `function main() { var code = 1; try { code = 52; throw value; } catch (err) { code++; } finally { return value; } return code; }`, "return after definite throw finally return")
}
