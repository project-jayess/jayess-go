package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/lowering"
)

func TestLoweringExtractsMainReturnCodeFromNumericExpression(t *testing.T) {
	program := parseProgram(t, `function main() { return 1 + 2 * 3; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected folded return code 7, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromLocalBinding(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 4; return value + 3; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected folded local return code 7, got %d", value)
	}
}

func TestLoweringIgnoresUnresolvedIdentifierReturn(t *testing.T) {
	program := parseProgram(t, `function main() { return value; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected unresolved identifier return to stay unresolved")
	}
}

func TestLoweringDoesNotFoldReturnAfterThrow(t *testing.T) {
	program := parseProgram(t, `function main() { throw value; return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after throw to stay unresolved")
	}
}

func TestLoweringDoesNotFoldReturnAfterBlockThrow(t *testing.T) {
	program := parseProgram(t, `function main() { { throw value; } return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after block throw to stay unresolved")
	}
}

func TestLoweringDoesNotFoldReturnAfterTryMayThrow(t *testing.T) {
	program := parseProgram(t, `function main() { try { throw value; } finally { cleanup(); } return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after try throw to stay unresolved")
	}
}

func TestLoweringDoesNotFoldReturnAfterCatchMayReturn(t *testing.T) {
	program := parseProgram(t, `function main() { try { run(); } catch (err) { return value; } return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after catch return to stay unresolved")
	}
}

func TestLoweringDoesNotFoldReturnAfterFinallyMayReturn(t *testing.T) {
	program := parseProgram(t, `function main() { try { run(); } finally { return value; } return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after finally return to stay unresolved")
	}
}

func TestLoweringAppliesTrySideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { code = 41; } catch (err) { code = 2; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 41 {
		t.Fatalf("expected try side-effect return code 41, got %d", value)
	}
}

func TestLoweringAppliesFinallySideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { code = 42; } finally { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 43 {
		t.Fatalf("expected finally side-effect return code 43, got %d", value)
	}
}

func TestLoweringStopsTrySideEffectsAtBreakBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; while (true) { try { code = 44; break; code = 2; } finally { code++; } } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 45 {
		t.Fatalf("expected try break/finally side-effect return code 45, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { code = 46; throw value; code = 2; } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 47 {
		t.Fatalf("expected catch side-effect return code 47, got %d", value)
	}
}

func TestLoweringAppliesCatchAndFinallySideEffectsAfterDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { code = 48; throw value; code = 2; } catch (err) { code++; } finally { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 50 {
		t.Fatalf("expected catch and finally side-effect return code 50, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterNestedBlockDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { { code = 53; throw value; code = 2; } } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 54 {
		t.Fatalf("expected nested block throw catch side-effect return code 54, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterConstantIfDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { if (true) { code = 55; throw value; code = 2; } else { code = 3; } } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 56 {
		t.Fatalf("expected constant if throw catch side-effect return code 56, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterConstantElseDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { if (false) { code = 3; } else { code = 57; throw value; code = 2; } } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 58 {
		t.Fatalf("expected constant else throw catch side-effect return code 58, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterConstantSwitchDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { switch (2) { case 1: code = 3; break; case 2: code = 59; throw value; code = 2; default: code = 4; } } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 60 {
		t.Fatalf("expected constant switch throw catch side-effect return code 60, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterConstantWhileDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { while (true) { code = 61; throw value; code = 2; } } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 62 {
		t.Fatalf("expected constant while throw catch side-effect return code 62, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterDoWhileDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { do { code = 63; throw value; code = 2; } while (false); } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 64 {
		t.Fatalf("expected do while throw catch side-effect return code 64, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterConstantForDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { for (code = 65; true; code++) { throw value; code = 2; } } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 66 {
		t.Fatalf("expected constant for throw catch side-effect return code 66, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterInfiniteForDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { for (code = 67;; code++) { throw value; code = 2; } } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 68 {
		t.Fatalf("expected infinite for throw catch side-effect return code 68, got %d", value)
	}
}

func TestLoweringDoesNotTreatUnreachableLoopThrowAsDefinite(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { while (true) { code = 69; break; throw value; } } catch (err) { code++; } return code; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected unreachable loop throw catch path to stay unresolved")
	}
}

func TestLoweringAppliesForInitializerBeforeLaterDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { for (code = 70; false; code++) { code = 2; } throw value; } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 71 {
		t.Fatalf("expected for initializer before later throw return code 71, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterLabeledBlockDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { entry: { code = 72; throw value; code = 2; } } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 73 {
		t.Fatalf("expected labeled block throw catch side-effect return code 73, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterNestedTryFinallyDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { try { code = 74; throw value; code = 2; } finally { code++; } } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 76 {
		t.Fatalf("expected nested try/finally throw catch side-effect return code 76, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterNestedFinallyDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { try { code = 80; } finally { code++; throw value; } } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 82 {
		t.Fatalf("expected nested finally throw side-effect return code 82, got %d", value)
	}
}

func TestLoweringAppliesCatchSideEffectsAfterNestedTryCatchRethrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { try { code = 77; throw value; } catch (err) { code++; throw value; } } catch (err) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 79 {
		t.Fatalf("expected nested try/catch rethrow side-effect return code 79, got %d", value)
	}
}

func TestLoweringDoesNotFoldReturnAfterDefiniteThrowCatchMayReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { code = 51; throw value; } catch (err) { return value; } return code; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after definite throw catch return to stay unresolved")
	}
}

func TestLoweringDoesNotFoldReturnAfterDefiniteThrowFinallyMayReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { code = 52; throw value; } catch (err) { code++; } finally { return value; } return code; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after definite throw finally return to stay unresolved")
	}
}

func TestLoweringSelectsMainReturnCodeFromConstantIfCondition(t *testing.T) {
	program := parseProgram(t, `function main() { if (false) { return 1; } else { return 2; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 2 {
		t.Fatalf("expected folded conditional return code 2, got %d", value)
	}
}

func TestLoweringDoesNotFoldReturnAfterUnknownIfMayReturn(t *testing.T) {
	program := parseProgram(t, `function main() { if (ready) { return value; } return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after unknown if return to stay unresolved")
	}
}

func TestLoweringDoesNotFoldReturnAfterUnknownIfMayThrow(t *testing.T) {
	program := parseProgram(t, `function main() { if (ready) { throw value; } return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after unknown if throw to stay unresolved")
	}
}

func TestLoweringUsesPostfixUpdateTruthinessInCondition(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 0; if (code++) { return 9; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 1 {
		t.Fatalf("expected postfix update condition return code 1, got %d", value)
	}
}

func TestLoweringUsesPrefixUpdateTruthinessInCondition(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 0; if (++code) { return code; } return 9; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 1 {
		t.Fatalf("expected prefix update condition return code 1, got %d", value)
	}
}

func TestLoweringEvaluatesNonNullishCoalesceLeftUpdateOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = code++ ?? 9; return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated coalesce update return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesIntCommaCoalesceLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = (code++, 2) ?? 9; return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 22 {
		t.Fatalf("expected single-evaluated int comma coalesce return code 22, got %d", value)
	}
}

func TestLoweringEvaluatesNullishCoalesceVoidUpdateOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = void code++ ?? 9; return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 29 {
		t.Fatalf("expected nullish coalesce void update return code 29, got %d", value)
	}
}

func TestLoweringEvaluatesStringCoalesceLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = (code++, "ready") ?? "fallback"; if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated string coalesce return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesNullishStringCoalesceProbeOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = (code++, null) ?? "ready"; if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated nullish string coalesce return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesBooleanCoalesceLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = (code++, true) ?? false; if (value) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated boolean coalesce return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesNullishBooleanCoalesceLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = (code++, null) ?? true; if (value) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated nullish boolean coalesce return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesObjectCoalesceLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, []) ?? {}; if (typeof chosen === "object") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated object coalesce return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesFunctionCoalesceLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, () => 1) ?? (() => 2); if (typeof chosen === "function") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated function coalesce return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesObjectOrLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, []) || {}; if (typeof chosen === "object") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated object or return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesFunctionOrLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, () => 1) || (() => 2); if (typeof chosen === "function") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated function or return code 21, got %d", value)
	}
}

func TestLoweringExtractsReturnFromLabeledStatement(t *testing.T) {
	program := parseProgram(t, `function main() { entry: return 27; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 27 {
		t.Fatalf("expected labeled return code 27, got %d", value)
	}
}

func TestLoweringAppliesLabeledBlockSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { code = 28; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 28 {
		t.Fatalf("expected labeled block side-effect return code 28, got %d", value)
	}
}

func TestLoweringDoesNotFoldReturnAfterLabeledThrow(t *testing.T) {
	program := parseProgram(t, `function main() { entry: { throw value; } return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after labeled throw to stay unresolved")
	}
}

func TestLoweringAppliesLabeledBreakSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { code = 29; break entry; code = 2; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 29 {
		t.Fatalf("expected labeled break side-effect return code 29, got %d", value)
	}
}

func TestLoweringAppliesLabeledBreakSideEffectsOnceBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { code++; break entry; code = 2; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 2 {
		t.Fatalf("expected labeled break side-effect return code 2, got %d", value)
	}
}

func TestLoweringAppliesNonMatchingBlockSideEffectsOnceBeforeLabeledBreak(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { { code++; } break entry; code = 4; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 2 {
		t.Fatalf("expected non-matching block side-effect return code 2, got %d", value)
	}
}

func TestLoweringAppliesNonMatchingNestedLabelSideEffectsOnceBeforeLabeledBreak(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { other: { code++; } break entry; code = 4; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 2 {
		t.Fatalf("expected non-matching nested label side-effect return code 2, got %d", value)
	}
}

func TestLoweringAppliesNestedLabeledBreakSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { if (true) { code = 30; break entry; } code = 2; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 30 {
		t.Fatalf("expected nested labeled break side-effect return code 30, got %d", value)
	}
}

func TestLoweringAppliesNonMatchingIfSideEffectsOnceBeforeLabeledBreak(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { if (true) { code++; } break entry; code = 4; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 2 {
		t.Fatalf("expected non-matching if side-effect return code 2, got %d", value)
	}
}

func TestLoweringAppliesSwitchLabeledBreakSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { switch (1) { case 1: code = 31; break entry; default: code = 2; } code = 3; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected switch labeled break side-effect return code 31, got %d", value)
	}
}

func TestLoweringAppliesNonMatchingSwitchSideEffectsOnceBeforeLabeledBreak(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { switch (1) { case 1: code++; break; default: code = 3; } break entry; code = 4; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 2 {
		t.Fatalf("expected non-matching switch side-effect return code 2, got %d", value)
	}
}

func TestLoweringAppliesSwitchFallthroughLabeledBreakSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { switch (1) { case 1: code = 32; case 2: break entry; default: code = 2; } code = 3; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 32 {
		t.Fatalf("expected switch fallthrough labeled break side-effect return code 32, got %d", value)
	}
}

func TestLoweringAppliesWhileLabeledBreakSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { while (true) { code = 33; break entry; code = 2; } code = 3; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 33 {
		t.Fatalf("expected while labeled break side-effect return code 33, got %d", value)
	}
}

func TestLoweringAppliesForLabeledBreakSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { for (code = 34; true; code++) { break entry; code = 2; } code = 3; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 34 {
		t.Fatalf("expected for labeled break side-effect return code 34, got %d", value)
	}
}

func TestLoweringAppliesDoWhileLabeledBreakSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: { do { code = 35; break entry; code = 2; } while (true); code = 3; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 35 {
		t.Fatalf("expected do-while labeled break side-effect return code 35, got %d", value)
	}
}

func TestLoweringAppliesDoWhileLabeledContinueSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: do { code = 36; continue entry; code = 2; } while (false); return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 36 {
		t.Fatalf("expected do-while labeled continue side-effect return code 36, got %d", value)
	}
}

func TestLoweringPropagatesNestedLabeledContinueSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; entry: do { while (true) { code = 37; continue entry; code = 2; } code = 3; } while (false); return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 37 {
		t.Fatalf("expected nested labeled continue side-effect return code 37, got %d", value)
	}
}

func TestLoweringAppliesBlockSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; { code = 21; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected block side-effect return code 21, got %d", value)
	}
}

func TestLoweringStopsBlockSideEffectsAtBreakInLoop(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; while (true) { { code = 22; break; code = 2; } } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 22 {
		t.Fatalf("expected block break side-effect return code 22, got %d", value)
	}
}

func TestLoweringAppliesNestedForSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; { for (code = 25; false; code++) { code = 2; } } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 25 {
		t.Fatalf("expected nested for side-effect return code 25, got %d", value)
	}
}

func TestLoweringAppliesForInitBeforeUnknownCondition(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; for (code = 38; ready; code++) { code = 2; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 38 {
		t.Fatalf("expected unknown-condition for init return code 38, got %d", value)
	}
}

func TestLoweringAppliesNestedForInitBeforeUnknownCondition(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; { for (code = 39; ready; code++) { code = 2; } } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 39 {
		t.Fatalf("expected nested unknown-condition for init return code 39, got %d", value)
	}
}

func TestLoweringAppliesNestedDoWhileSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (true) { do { code = 26; } while (false); } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 26 {
		t.Fatalf("expected nested do while side-effect return code 26, got %d", value)
	}
}

func TestLoweringAppliesConstantIfSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (true) { code = 7; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected constant if side-effect return code 7, got %d", value)
	}
}

func TestLoweringAppliesNestedSwitchSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (true) { switch (1) { case 1: code = 19; break; default: code = 2; } } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 19 {
		t.Fatalf("expected nested switch side-effect return code 19, got %d", value)
	}
}

func TestLoweringAppliesConstantElseSideEffectsBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (false) { code = 2; } else { code = 8; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 8 {
		t.Fatalf("expected constant else side-effect return code 8, got %d", value)
	}
}

func TestLoweringSelectsMainReturnCodeFromBooleanBinding(t *testing.T) {
	program := parseProgram(t, `function main() { const enabled = true; if (!enabled) { return 1; } return 3; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 3 {
		t.Fatalf("expected folded boolean binding return code 3, got %d", value)
	}
}

func TestLoweringSelectsMainReturnCodeFromComparisonCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const count = 3; if (count >= 3) { return 8; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 8 {
		t.Fatalf("expected folded comparison return code 8, got %d", value)
	}
}

func TestLoweringSelectsMainReturnCodeFromLogicalCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const count = 2; const ready = true; if (ready && count < 2) { return 1; } return 6; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 6 {
		t.Fatalf("expected folded logical return code 6, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromLogicalAndValue(t *testing.T) {
	program := parseProgram(t, `function main() { return true && 5; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 5 {
		t.Fatalf("expected logical and value return code 5, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromLogicalOrValue(t *testing.T) {
	program := parseProgram(t, `function main() { return 0 || 7; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected logical or value return code 7, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromShortCircuitLogicalValue(t *testing.T) {
	program := parseProgram(t, `function main() { return 6 || 9; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 6 {
		t.Fatalf("expected short circuit logical value return code 6, got %d", value)
	}
}

func TestLoweringEvaluatesIntLogicalOrLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = code++ || 9; return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated int logical or return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesIntLogicalAndLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 0; var value = code++ && 9; return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 10 {
		t.Fatalf("expected single-evaluated int logical and return code 10, got %d", value)
	}
}

func TestLoweringUsesStringLogicalAndValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = "ready" && "go"; if (value === "go") { return 8; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 8 {
		t.Fatalf("expected string logical and return code 8, got %d", value)
	}
}

func TestLoweringUsesStringLogicalOrValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = "" || "fallback"; if (value === "fallback") { return 9; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 9 {
		t.Fatalf("expected string logical or return code 9, got %d", value)
	}
}

func TestLoweringEvaluatesStringLogicalOrLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = (code++, "ready") || "fallback"; if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated string logical or return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesStringLogicalAndFallbackLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = (code++, 1) && "ready"; if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated string logical and return code 21, got %d", value)
	}
}

func TestLoweringUsesStringConcatenationInCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const value = "jay" + "ess"; if (value === "jayess") { return 27; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 27 {
		t.Fatalf("expected string concatenation return code 27, got %d", value)
	}
}

func TestLoweringEvaluatesNumericAddLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = (code++, 2) + 3; return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 25 {
		t.Fatalf("expected single-evaluated numeric add return code 25, got %d", value)
	}
}

func TestLoweringEvaluatesNumericAddAssignmentRightOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = 2; value += (code++, 3); return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 25 {
		t.Fatalf("expected single-evaluated numeric add-assignment return code 25, got %d", value)
	}
}

func TestLoweringEvaluatesScalarAssignmentProbeOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = 0; value = (code++, "ready"); if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated scalar assignment return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesLogicalScalarAssignmentProbeOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = ""; value ||= (code++, "ready"); if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated logical scalar assignment return code 21, got %d", value)
	}
}

func TestLoweringEvaluatesNumericStringCoercionOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; var value = +(code++, "2"); return code * 10 + value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 22 {
		t.Fatalf("expected single-evaluated numeric string coercion return code 22, got %d", value)
	}
}

func TestLoweringDoesNotCommitFailedNumericStringCoercion(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; +(code++, "x"); return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 1 {
		t.Fatalf("expected failed numeric coercion to keep return code 1, got %d", value)
	}
}

func TestLoweringDoesNotCommitFailedNumericConditionalBoolProbe(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; +((code++, true) ? "x" : false); return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 1 {
		t.Fatalf("expected failed numeric conditional probe to keep return code 1, got %d", value)
	}
}

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

func TestLoweringUsesNumericBinaryFalsyCondition(t *testing.T) {
	program := parseProgram(t, `function main() { if (1 - 1) { return 1; } return 81; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 81 {
		t.Fatalf("expected numeric binary falsy return code 81, got %d", value)
	}
}

func TestLoweringUsesPrimitiveNumericBinaryOperators(t *testing.T) {
	program := parseProgram(t, `function main() { return ("7" - true) + (true * 4) + (null + 2); }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 12 {
		t.Fatalf("expected primitive numeric binary return code 12, got %d", value)
	}
}

func TestLoweringKeepsStringPlusAsConcatenation(t *testing.T) {
	program := parseProgram(t, `function main() { const value = "7" + 1; if (value === "71") { return 106; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 106 {
		t.Fatalf("expected string plus concatenation return code 106, got %d", value)
	}
}

func TestLoweringAppliesStringAddAssignmentBeforeCondition(t *testing.T) {
	program := parseProgram(t, `function main() { var value = "jay"; value += "ess"; if (value === "jayess") { return 28; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 28 {
		t.Fatalf("expected string add assignment return code 28, got %d", value)
	}
}

func TestLoweringAppliesStringNumberAddAssignmentBeforeCondition(t *testing.T) {
	program := parseProgram(t, `function main() { var value = "code:"; value += 7; if (value === "code:7") { return 74; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 74 {
		t.Fatalf("expected string number add assignment return code 74, got %d", value)
	}
}

func TestLoweringAppliesStringPrimitiveAddAssignmentBeforeCondition(t *testing.T) {
	program := parseProgram(t, `function main() { var value = "ready:"; value += false; value += ":"; value += undefined; if (value === "ready:false:undefined") { return 75; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 75 {
		t.Fatalf("expected string primitive add assignment return code 75, got %d", value)
	}
}

func TestLoweringUsesNullishLogicalAndValue(t *testing.T) {
	program := parseProgram(t, `function main() { const value = null && undefined; if (value === null) { return 10; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 10 {
		t.Fatalf("expected nullish logical and return code 10, got %d", value)
	}
}

func TestLoweringEvaluatesNullishLogicalAndLeftOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = (code++, null) && undefined; if (value === null) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated nullish logical and return code 21, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromConditionalExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const count = 4; return count > 3 ? 11 : 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 11 {
		t.Fatalf("expected folded conditional expression return code 11, got %d", value)
	}
}

func TestLoweringEvaluatesStringConditionalConditionOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = (code++, true) ? "ready" : "fallback"; if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated string conditional return code 21, got %d", value)
	}
}

func TestLoweringUsesConditionalExpressionForBooleanBinding(t *testing.T) {
	program := parseProgram(t, `function main() { const count = 0; const ready = count ? false : true; if (ready) { return 5; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 5 {
		t.Fatalf("expected folded conditional boolean return code 5, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeAfterNumericAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; code += 4; code *= 2; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 10 {
		t.Fatalf("expected folded assignment return code 10, got %d", value)
	}
}

func TestLoweringAppliesPrimitiveNumericCompoundAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 10; code -= "3"; code *= true; code += null; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected primitive numeric assignment return code 7, got %d", value)
	}
}

func TestLoweringKeepsStringAssignmentAsString(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 1; value = "7"; if (value === "7") { return 107; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 107 {
		t.Fatalf("expected string assignment return code 107, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeAfterBooleanAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var ready = false; ready = true; if (ready) { return 7; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected folded boolean assignment return code 7, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeAfterPostfixUpdate(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; code++; code++; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 3 {
		t.Fatalf("expected folded postfix update return code 3, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeAfterPrefixUpdate(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 4; --code; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 3 {
		t.Fatalf("expected folded prefix update return code 3, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromPostfixUpdateExpression(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; return code++ + code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 3 {
		t.Fatalf("expected postfix update expression return code 3, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromPrefixUpdateExpression(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; return ++code + code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 4 {
		t.Fatalf("expected prefix update expression return code 4, got %d", value)
	}
}

func TestLoweringAppliesCommaExpressionStatementSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; code++, code++; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 3 {
		t.Fatalf("expected comma expression statement return code 3, got %d", value)
	}
}

func TestLoweringEvaluatesDiscardExpressionOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; code++, "done"; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 2 {
		t.Fatalf("expected single-evaluated discard expression return code 2, got %d", value)
	}
}

func TestLoweringEvaluatesStringCommaExpressionProbeOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const value = (code++, "ready"); if (value === "ready") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated string comma expression return code 21, got %d", value)
	}
}

func TestLoweringAppliesConditionalExpressionStatementSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; true ? code++ : code--; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 2 {
		t.Fatalf("expected conditional expression statement return code 2, got %d", value)
	}
}

func TestLoweringSkipsConstantFalseWhileBeforeReturn(t *testing.T) {
	program := parseProgram(t, `function main() { while (false) { return 1; } return 4; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 4 {
		t.Fatalf("expected skipped while return code 4, got %d", value)
	}
}

func TestLoweringExtractsReturnFromConstantTrueWhileBody(t *testing.T) {
	program := parseProgram(t, `function main() { while (true) { return 12; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 12 {
		t.Fatalf("expected constant true while return code 12, got %d", value)
	}
}

func TestLoweringAppliesConstantTrueWhileSideEffectsUntilBreak(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; while (true) { code = 12; break; code = 2; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 12 {
		t.Fatalf("expected while break side-effect return code 12, got %d", value)
	}
}

func TestLoweringDoesNotFoldReturnAfterConstantTrueWhile(t *testing.T) {
	program := parseProgram(t, `function main() { while (true) { } return 1; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after constant true while to stay unresolved")
	}
}

func TestLoweringDoesNotFoldReturnAfterConstantTrueWhileContinue(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; while (true) { code = 40; continue; code = 2; } return code; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after continuing constant true while to stay unresolved")
	}
}

func TestLoweringDoesNotFoldReturnAfterUnknownWhileMayReturn(t *testing.T) {
	program := parseProgram(t, `function main() { while (ready) { return value; } return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after unknown while return to stay unresolved")
	}
}

func TestLoweringAppliesNestedIfSideEffectsUntilLoopBreak(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; while (true) { if (true) { code = 18; break; } code = 2; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 18 {
		t.Fatalf("expected nested if loop break side-effect return code 18, got %d", value)
	}
}

func TestLoweringAppliesNestedSwitchSideEffectsUntilLoopBreak(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; while (true) { switch (1) { case 1: code = 20; break; default: code = 2; } break; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 20 {
		t.Fatalf("expected nested switch loop side-effect return code 20, got %d", value)
	}
}

func TestLoweringAppliesForInitBeforeConstantFalseLoop(t *testing.T) {
	program := parseProgram(t, `function main() { for (var code = 6; false; code++) { return 1; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 6 {
		t.Fatalf("expected for init return code 6, got %d", value)
	}
}

func TestLoweringExtractsReturnFromConstantTrueForBody(t *testing.T) {
	program := parseProgram(t, `function main() { for (var code = 13; true; code++) { return code; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 13 {
		t.Fatalf("expected constant true for return code 13, got %d", value)
	}
}

func TestLoweringAppliesConstantTrueForSideEffectsUntilBreak(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; for (; true; code++) { code = 15; break; code = 2; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 15 {
		t.Fatalf("expected for break side-effect return code 15, got %d", value)
	}
}

func TestLoweringDoesNotFoldReturnAfterConstantTrueFor(t *testing.T) {
	program := parseProgram(t, `function main() { for (; true; ) { } return 1; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after constant true for to stay unresolved")
	}
}

func TestLoweringDoesNotFoldReturnAfterUnknownForMayReturn(t *testing.T) {
	program := parseProgram(t, `function main() { for (; ready; ) { return value; } return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after unknown for return to stay unresolved")
	}
}

func TestLoweringExtractsReturnFromConditionlessForBody(t *testing.T) {
	program := parseProgram(t, `function main() { for (var code = 14;; code++) { return code; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 14 {
		t.Fatalf("expected conditionless for return code 14, got %d", value)
	}
}

func TestLoweringAppliesConditionlessForSideEffectsUntilBreak(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; for (;;) { code = 16; break; code = 2; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 16 {
		t.Fatalf("expected conditionless for break side-effect return code 16, got %d", value)
	}
}

func TestLoweringDoesNotFoldReturnAfterConditionlessFor(t *testing.T) {
	program := parseProgram(t, `function main() { for (;;) { } return 1; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after conditionless for to stay unresolved")
	}
}

func TestLoweringExtractsReturnFromDoWhileBody(t *testing.T) {
	program := parseProgram(t, `function main() { do { return 9; } while (false); return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 9 {
		t.Fatalf("expected do body return code 9, got %d", value)
	}
}

func TestLoweringAppliesDoWhileBodyBeforeConstantFalseCondition(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 2; do { code += 3; } while (false); return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 5 {
		t.Fatalf("expected do body assignment return code 5, got %d", value)
	}
}

func TestLoweringAppliesDoWhileSideEffectsUntilBreak(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 2; do { code = 23; break; code = 1; } while (true); return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 23 {
		t.Fatalf("expected do while break side-effect return code 23, got %d", value)
	}
}

func TestLoweringStopsDoWhileSideEffectsAtContinue(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 2; do { code = 17; continue; code = 1; } while (false); return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 17 {
		t.Fatalf("expected do while continue side-effect return code 17, got %d", value)
	}
}

func TestLoweringDoesNotFoldReturnAfterConstantTrueDoWhile(t *testing.T) {
	program := parseProgram(t, `function main() { do { } while (true); return 1; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after constant true do while to stay unresolved")
	}
}

func TestLoweringDoesNotFoldReturnAfterDoWhileMayReturn(t *testing.T) {
	program := parseProgram(t, `function main() { do { return value; } while (false); return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after do while return to stay unresolved")
	}
}

func TestLoweringExtractsReturnFromConstantSwitchCase(t *testing.T) {
	program := parseProgram(t, `function main() { const kind = 2; switch (kind) { case 1: return 1; case 2: return 8; default: return 3; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 8 {
		t.Fatalf("expected matched switch return code 8, got %d", value)
	}
}

func TestLoweringDoesNotFoldReturnAfterUnknownSwitchMayReturn(t *testing.T) {
	program := parseProgram(t, `function main() { switch (kind) { case 1: return value; } return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after unknown switch return to stay unresolved")
	}
}

func TestLoweringDoesNotFoldReturnAfterUnknownSwitchMayThrow(t *testing.T) {
	program := parseProgram(t, `function main() { switch (kind) { case 1: throw value; } return 2; }`)

	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatal("expected return after unknown switch throw to stay unresolved")
	}
}

func TestLoweringAppliesConstantSwitchDefaultSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; switch (false) { case true: code = 9; break; default: code += 4; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 5 {
		t.Fatalf("expected switch default side-effect return code 5, got %d", value)
	}
}

func TestLoweringExtractsReturnFromStringSwitchCase(t *testing.T) {
	program := parseProgram(t, `function main() { const kind = "audio"; switch (kind) { case "video": return 1; case "audio": return 6; default: return 2; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 6 {
		t.Fatalf("expected string switch return code 6, got %d", value)
	}
}

func TestLoweringAppliesStringAssignmentBeforeSwitch(t *testing.T) {
	program := parseProgram(t, `function main() { var kind = "debug"; kind = "release"; switch (kind) { case "release": return 12; default: return 1; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 12 {
		t.Fatalf("expected string assignment switch return code 12, got %d", value)
	}
}

func TestLoweringStopsSwitchSideEffectsAtBreak(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; switch (1) { case 1: code = 4; break; code = 9; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 4 {
		t.Fatalf("expected switch break return code 4, got %d", value)
	}
}

func TestLoweringFallsThroughEmptySwitchCase(t *testing.T) {
	program := parseProgram(t, `function main() { switch (1) { case 1: case 2: return 7; default: return 3; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 7 {
		t.Fatalf("expected empty switch case fallthrough return code 7, got %d", value)
	}
}

func TestLoweringFallsThroughNonEmptySwitchCase(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; switch (1) { case 1: code = 24; case 2: return code; default: return 3; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 24 {
		t.Fatalf("expected non-empty switch fallthrough return code 24, got %d", value)
	}
}

func TestLoweringTreatsNullAndUndefinedConditionsAsFalse(t *testing.T) {
	program := parseProgram(t, `function main() { const value = null; if (value) { return 1; } const missing = undefined; if (missing) { return 2; } return 6; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 6 {
		t.Fatalf("expected nullish condition return code 6, got %d", value)
	}
}

func TestLoweringExtractsReturnFromNullishSwitchCase(t *testing.T) {
	program := parseProgram(t, `function main() { var value = null; value = undefined; switch (value) { case null: return 1; case undefined: return 9; default: return 2; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 9 {
		t.Fatalf("expected undefined switch return code 9, got %d", value)
	}
}

func TestLoweringSelectsReturnFromStringComparison(t *testing.T) {
	program := parseProgram(t, `function main() { const mode = "release"; if (mode === "release") { return 14; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 14 {
		t.Fatalf("expected string comparison return code 14, got %d", value)
	}
}

func TestLoweringSelectsReturnFromStringRelationalComparison(t *testing.T) {
	program := parseProgram(t, `function main() { if ("beta" > "alpha" && "same" <= "same") { return 43; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 43 {
		t.Fatalf("expected string relational comparison return code 43, got %d", value)
	}
}

func TestLoweringUsesNumericRelationalPrimitiveCoercion(t *testing.T) {
	program := parseProgram(t, `function main() { if ("7" > 3 && true <= 1 && null >= 0) { return 108; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 108 {
		t.Fatalf("expected numeric relational primitive return code 108, got %d", value)
	}
}

func TestLoweringKeepsStringRelationalComparisonLexical(t *testing.T) {
	program := parseProgram(t, `function main() { if ("10" < "2") { return 109; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 109 {
		t.Fatalf("expected lexical string relational return code 109, got %d", value)
	}
}

func TestLoweringSelectsReturnFromBoolAndNullishComparison(t *testing.T) {
	program := parseProgram(t, `function main() { const ready = true; const missing = undefined; if (ready !== false && missing === undefined) { return 15; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 15 {
		t.Fatalf("expected bool/nullish comparison return code 15, got %d", value)
	}
}

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

func TestLoweringExtractsReturnFromCommaExpression(t *testing.T) {
	program := parseProgram(t, `function main() { return (1, 16); }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 16 {
		t.Fatalf("expected comma expression return code 16, got %d", value)
	}
}

func TestLoweringUsesCommaExpressionInConstantCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const mode = ("debug", "release"); if (mode === "release") { return 17; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 17 {
		t.Fatalf("expected comma condition return code 17, got %d", value)
	}
}

func TestLoweringExtractsReturnFromNullishCoalescingExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = 18; return null ?? fallback; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 18 {
		t.Fatalf("expected nullish coalescing return code 18, got %d", value)
	}
}

func TestLoweringPreservesFalsyNonNullishCoalescingLeft(t *testing.T) {
	program := parseProgram(t, `function main() { const value = 0 ?? 19; return value; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 0 {
		t.Fatalf("expected falsy non-nullish coalescing return code 0, got %d", value)
	}
}

func TestLoweringUsesStringNullishCoalescingInCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const mode = undefined ?? "release"; if (mode === "release") { return 20; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 20 {
		t.Fatalf("expected string nullish coalescing return code 20, got %d", value)
	}
}

func TestLoweringAppliesNullishAssignmentForNullishLocal(t *testing.T) {
	program := parseProgram(t, `function main() { var code = undefined; code ??= 21; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected nullish assignment return code 21, got %d", value)
	}
}

func TestLoweringPreservesNonNullishLocalForNullishAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 0; code ??= 22; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 0 {
		t.Fatalf("expected preserved nullish assignment return code 0, got %d", value)
	}
}

func TestLoweringAppliesLogicalOrAndAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 0; code ||= 23; var ready = true; ready &&= false; if (!ready) { return code; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 23 {
		t.Fatalf("expected logical assignment return code 23, got %d", value)
	}
}

func TestLoweringUsesStringBindingTruthiness(t *testing.T) {
	program := parseProgram(t, `function main() { const label = "ready"; if (label) { return 24; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 24 {
		t.Fatalf("expected string truthiness return code 24, got %d", value)
	}
}

func TestLoweringUsesEmptyStringLiteralAsFalsy(t *testing.T) {
	program := parseProgram(t, `function main() { if ("") { return 1; } return 25; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 25 {
		t.Fatalf("expected empty string falsy return code 25, got %d", value)
	}
}

func TestLoweringUsesTypeofForKnownLocalCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const code = 24; if (typeof code === "number") { return code; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 24 {
		t.Fatalf("expected typeof known local return code 24, got %d", value)
	}
}

func TestLoweringEvaluatesTypeofOperandOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (typeof (code++, "x") === "string") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated typeof operand return code 21, got %d", value)
	}
}

func TestLoweringUsesTypeofForMissingIdentifierCondition(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof missing === "undefined") { return 25; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 25 {
		t.Fatalf("expected typeof missing identifier return code 25, got %d", value)
	}
}

func TestLoweringUsesTypeofForFunctionExpressionCondition(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof (() => 1) === "function") { return 44; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 44 {
		t.Fatalf("expected typeof function expression return code 44, got %d", value)
	}
}

func TestLoweringUsesTypeofForFunctionBindingCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (typeof f === "function") { return 45; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 45 {
		t.Fatalf("expected typeof function binding return code 45, got %d", value)
	}
}

func TestLoweringUsesTypeofForAssignedFunctionBinding(t *testing.T) {
	program := parseProgram(t, `function main() { var f = undefined; f = function () { return 1; }; if (typeof f === "function") { return 46; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 46 {
		t.Fatalf("expected typeof assigned function binding return code 46, got %d", value)
	}
}

func TestLoweringUsesTypeofForFunctionConditionalExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const left = () => 1; const right = () => 2; if (typeof (true ? left : right) === "function") { return 86; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 86 {
		t.Fatalf("expected typeof function conditional return code 86, got %d", value)
	}
}

func TestLoweringUsesTypeofForFunctionLogicalExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const left = () => 1; const right = () => 2; if (typeof (left || right) === "function") { return 87; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 87 {
		t.Fatalf("expected typeof function logical return code 87, got %d", value)
	}
}

func TestLoweringEvaluatesTypeofFreshFunctionLogicalOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (typeof ((code++, (() => 1)) || (() => 2)) === "function") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated typeof fresh function logical return code 21, got %d", value)
	}
}

func TestLoweringUsesTypeofForFreshFunctionNullishExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = () => 2; if (typeof ((() => 1) ?? fallback) === "function") { return 98; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 98 {
		t.Fatalf("expected typeof fresh function nullish return code 98, got %d", value)
	}
}

func TestLoweringUsesTypeofForFreshFunctionConditionalExpression(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof (true ? (() => 1) : (() => 2)) === "function") { return 99; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 99 {
		t.Fatalf("expected typeof fresh function conditional return code 99, got %d", value)
	}
}

func TestLoweringEvaluatesTypeofFreshFunctionConditionalOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (typeof ((code++, true) ? (() => 1) : (() => 2)) === "function") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated typeof fresh function conditional return code 21, got %d", value)
	}
}

func TestLoweringTreatsFunctionExpressionAsTruthy(t *testing.T) {
	program := parseProgram(t, `function main() { if (() => 1) { return 47; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 47 {
		t.Fatalf("expected function expression truthy return code 47, got %d", value)
	}
}

func TestLoweringTreatsFunctionBindingAsTruthy(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (!f) { return 1; } return 48; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 48 {
		t.Fatalf("expected function binding truthy return code 48, got %d", value)
	}
}

func TestLoweringComparesSameFunctionBindingAsEqual(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (f === f) { return 58; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 58 {
		t.Fatalf("expected same function binding equality return code 58, got %d", value)
	}
}

func TestLoweringComparesDistinctFunctionBindingsAsNotEqual(t *testing.T) {
	program := parseProgram(t, `function main() { const left = () => 1; const right = () => 1; if (left !== right) { return 59; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 59 {
		t.Fatalf("expected distinct function binding inequality return code 59, got %d", value)
	}
}

func TestLoweringComparesFreshFunctionExpressionsAsNotEqual(t *testing.T) {
	program := parseProgram(t, `function main() { if ((() => 1) !== (() => 1)) { return 60; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 60 {
		t.Fatalf("expected fresh function expression inequality return code 60, got %d", value)
	}
}

func TestLoweringComparesFunctionBindingAgainstNullish(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (f != null && f !== undefined) { return 61; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 61 {
		t.Fatalf("expected function nullish comparison return code 61, got %d", value)
	}
}

func TestLoweringPreservesFunctionIdentityThroughBindingAlias(t *testing.T) {
	program := parseProgram(t, `function main() { const left = () => 1; const right = left; if (left === right) { return 62; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 62 {
		t.Fatalf("expected function alias equality return code 62, got %d", value)
	}
}

func TestLoweringPreservesFunctionIdentityThroughLogicalOr(t *testing.T) {
	program := parseProgram(t, `function main() { const left = () => 1; const right = () => 2; const chosen = left || right; if (chosen === left) { return 82; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 82 {
		t.Fatalf("expected function logical identity return code 82, got %d", value)
	}
}

func TestLoweringMaterializesFreshFunctionThroughLogicalAnd(t *testing.T) {
	program := parseProgram(t, `function main() { const chosen = (() => 1) && (() => 2); if (typeof chosen === "function" && chosen === chosen) { return 102; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 102 {
		t.Fatalf("expected fresh function logical-and materialization return code 102, got %d", value)
	}
}

func TestLoweringMaterializesFreshFunctionThroughNullishCoalescing(t *testing.T) {
	program := parseProgram(t, `function main() { const chosen = (() => 1) ?? (() => 2); if (typeof chosen === "function" && chosen === chosen) { return 94; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 94 {
		t.Fatalf("expected fresh function nullish materialization return code 94, got %d", value)
	}
}

func TestLoweringEvaluatesFreshFunctionNullishMaterializationOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, null) ?? (() => 2); if (typeof chosen === "function" && chosen === chosen) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated fresh function nullish materialization return code 21, got %d", value)
	}
}

func TestLoweringMaterializesFreshFunctionThroughConditional(t *testing.T) {
	program := parseProgram(t, `function main() { const chosen = true ? (() => 1) : (() => 2); if (typeof chosen === "function" && chosen === chosen) { return 95; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 95 {
		t.Fatalf("expected fresh function conditional materialization return code 95, got %d", value)
	}
}

func TestLoweringEvaluatesFunctionConditionalConditionOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, true) ? (() => 1) : (() => 2); if (typeof chosen === "function" && chosen === chosen) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated function conditional return code 21, got %d", value)
	}
}

func TestLoweringPreservesFunctionIdentityThroughComma(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = () => 1; const chosen = (0, fallback); if (chosen === fallback) { return 83; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 83 {
		t.Fatalf("expected function comma identity return code 83, got %d", value)
	}
}

func TestLoweringPreservesFunctionIdentityThroughLogicalAndAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { var chosen = () => 1; const next = () => 2; chosen &&= next; if (chosen === next) { return 90; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 90 {
		t.Fatalf("expected function logical assignment identity return code 90, got %d", value)
	}
}

func TestLoweringPreservesFunctionIdentityThroughLogicalOrAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { const left = () => 1; const right = () => 2; var chosen = left; chosen ||= right; if (chosen === left) { return 91; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 91 {
		t.Fatalf("expected function logical or assignment identity return code 91, got %d", value)
	}
}

func TestLoweringDoesNotUseFunctionTruthinessForLooseBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (f == true) { return 1; } if (f != true) { return 63; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 63 {
		t.Fatalf("expected function loose boolean inequality return code 63, got %d", value)
	}
}

func TestLoweringSeparatesStrictFunctionNumberEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (f === 1) { return 1; } if (f !== 1) { return 67; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 67 {
		t.Fatalf("expected strict function number inequality return code 67, got %d", value)
	}
}

func TestLoweringEvaluatesReferencePrimitiveMismatchOperandOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if ((code++, 1) !== (() => 1)) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated reference primitive mismatch return code 21, got %d", value)
	}
}

func TestLoweringSeparatesStrictFunctionStringEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const f = () => 1; if (f === "fn") { return 1; } if (f !== "fn") { return 68; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 68 {
		t.Fatalf("expected strict function string inequality return code 68, got %d", value)
	}
}

func TestLoweringUsesBooleanValuesForLooseBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { if (true == false) { return 1; } if (true != false) { return 64; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 64 {
		t.Fatalf("expected boolean loose inequality return code 64, got %d", value)
	}
}

func TestLoweringUsesTypeofForObjectAndArrayLiterals(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof ({ value: 1 }) === "object" && typeof [] === "object") { return 49; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 49 {
		t.Fatalf("expected typeof object and array literals return code 49, got %d", value)
	}
}

func TestLoweringUsesTypeofForObjectBindingCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const value = {}; if (typeof value === "object") { return 50; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 50 {
		t.Fatalf("expected typeof object binding return code 50, got %d", value)
	}
}

func TestLoweringTreatsArrayBindingAsTruthy(t *testing.T) {
	program := parseProgram(t, `function main() { const items = []; if (items) { return 51; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 51 {
		t.Fatalf("expected array binding truthy return code 51, got %d", value)
	}
}

func TestLoweringUsesTypeofForAssignedArrayBinding(t *testing.T) {
	program := parseProgram(t, `function main() { var items = undefined; items = []; if (typeof items === "object") { return 52; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 52 {
		t.Fatalf("expected typeof assigned array binding return code 52, got %d", value)
	}
}

func TestLoweringUsesTypeofForObjectNullishExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = []; if (typeof (undefined ?? fallback) === "object") { return 88; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 88 {
		t.Fatalf("expected typeof object nullish return code 88, got %d", value)
	}
}

func TestLoweringUsesTypeofForObjectCommaExpression(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = {}; if (typeof (0, fallback) === "object") { return 89; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 89 {
		t.Fatalf("expected typeof object comma return code 89, got %d", value)
	}
}

func TestLoweringUsesTypeofForFreshObjectNullishExpression(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof ([] ?? {}) === "object") { return 100; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 100 {
		t.Fatalf("expected typeof fresh object nullish return code 100, got %d", value)
	}
}

func TestLoweringUsesTypeofForFreshObjectLogicalExpression(t *testing.T) {
	program := parseProgram(t, `function main() { if (typeof ({} || []) === "object") { return 101; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 101 {
		t.Fatalf("expected typeof fresh object logical return code 101, got %d", value)
	}
}

func TestLoweringEvaluatesTypeofFreshObjectLogicalOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if (typeof ((code++, {}) || []) === "object") { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated typeof fresh object logical return code 21, got %d", value)
	}
}

func TestLoweringComparesSameObjectBindingAsEqual(t *testing.T) {
	program := parseProgram(t, `function main() { const value = {}; if (value === value) { return 53; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 53 {
		t.Fatalf("expected same object binding equality return code 53, got %d", value)
	}
}

func TestLoweringComparesDistinctObjectBindingsAsNotEqual(t *testing.T) {
	program := parseProgram(t, `function main() { const left = {}; const right = {}; if (left !== right) { return 54; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 54 {
		t.Fatalf("expected distinct object binding inequality return code 54, got %d", value)
	}
}

func TestLoweringComparesFreshObjectLiteralsAsNotEqual(t *testing.T) {
	program := parseProgram(t, `function main() { if (({}) !== ({})) { return 55; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 55 {
		t.Fatalf("expected fresh object literal inequality return code 55, got %d", value)
	}
}

func TestLoweringComparesObjectBindingAgainstNullish(t *testing.T) {
	program := parseProgram(t, `function main() { const value = {}; if (value != null && value !== undefined) { return 56; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 56 {
		t.Fatalf("expected object nullish comparison return code 56, got %d", value)
	}
}

func TestLoweringPreservesObjectIdentityThroughBindingAlias(t *testing.T) {
	program := parseProgram(t, `function main() { const left = []; const right = left; if (left === right) { return 57; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 57 {
		t.Fatalf("expected object alias equality return code 57, got %d", value)
	}
}

func TestLoweringPreservesObjectIdentityThroughNullishCoalescing(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = []; const chosen = undefined ?? fallback; if (chosen === fallback) { return 84; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 84 {
		t.Fatalf("expected object nullish identity return code 84, got %d", value)
	}
}

func TestLoweringMaterializesFreshObjectThroughNullishCoalescing(t *testing.T) {
	program := parseProgram(t, `function main() { const chosen = [] ?? {}; if (typeof chosen === "object" && chosen === chosen) { return 96; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 96 {
		t.Fatalf("expected fresh object nullish materialization return code 96, got %d", value)
	}
}

func TestLoweringMaterializesFreshObjectThroughLogicalOr(t *testing.T) {
	program := parseProgram(t, `function main() { const chosen = {} || []; if (typeof chosen === "object" && chosen === chosen) { return 97; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 97 {
		t.Fatalf("expected fresh object logical materialization return code 97, got %d", value)
	}
}

func TestLoweringMaterializesFreshObjectThroughLogicalAnd(t *testing.T) {
	program := parseProgram(t, `function main() { const chosen = {} && []; if (typeof chosen === "object" && chosen === chosen) { return 103; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 103 {
		t.Fatalf("expected fresh object logical-and materialization return code 103, got %d", value)
	}
}

func TestLoweringPreservesObjectIdentityThroughConditional(t *testing.T) {
	program := parseProgram(t, `function main() { const left = []; const right = {}; const chosen = true ? left : right; if (chosen === left) { return 85; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 85 {
		t.Fatalf("expected object conditional identity return code 85, got %d", value)
	}
}

func TestLoweringEvaluatesObjectConditionalConditionOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; const chosen = (code++, true) ? {} : []; if (typeof chosen === "object" && chosen === chosen) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated object conditional return code 21, got %d", value)
	}
}

func TestLoweringPreservesObjectIdentityThroughNullishAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = []; var chosen = null; chosen ??= fallback; if (chosen === fallback) { return 92; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 92 {
		t.Fatalf("expected object nullish assignment identity return code 92, got %d", value)
	}
}

func TestLoweringPreservesObjectIdentityThroughLogicalOrAssignment(t *testing.T) {
	program := parseProgram(t, `function main() { const fallback = {}; var chosen = null; chosen ||= fallback; if (chosen === fallback) { return 93; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 93 {
		t.Fatalf("expected object logical or assignment identity return code 93, got %d", value)
	}
}

func TestLoweringDoesNotUseObjectTruthinessForLooseBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const value = {}; if (value == true) { return 1; } if (value != true) { return 65; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 65 {
		t.Fatalf("expected object loose boolean inequality return code 65, got %d", value)
	}
}

func TestLoweringDoesNotUseArrayTruthinessForLooseBooleanEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const value = []; if (value == true) { return 1; } if (value != true) { return 66; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 66 {
		t.Fatalf("expected array loose boolean inequality return code 66, got %d", value)
	}
}

func TestLoweringSeparatesStrictObjectNumberEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const value = {}; if (value === 1) { return 1; } if (value !== 1) { return 69; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 69 {
		t.Fatalf("expected strict object number inequality return code 69, got %d", value)
	}
}

func TestLoweringEvaluatesObjectPrimitiveMismatchOperandOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; if ((code++, "x") !== ({})) { return code * 10 + 1; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 21 {
		t.Fatalf("expected single-evaluated object primitive mismatch return code 21, got %d", value)
	}
}

func TestLoweringSeparatesStrictArrayStringEquality(t *testing.T) {
	program := parseProgram(t, `function main() { const value = []; if (value === "items") { return 1; } if (value !== "items") { return 70; } return 2; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 70 {
		t.Fatalf("expected strict array string inequality return code 70, got %d", value)
	}
}

func TestLoweringUsesVoidAsUndefinedCondition(t *testing.T) {
	program := parseProgram(t, `function main() { const value = void 0; if (value === undefined) { return 26; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 26 {
		t.Fatalf("expected void undefined return code 26, got %d", value)
	}
}

func TestLoweringUsesVoidExpressionAsFalsyCondition(t *testing.T) {
	program := parseProgram(t, `function main() { if (void 0) { return 1; } return 102; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 102 {
		t.Fatalf("expected void falsy return code 102, got %d", value)
	}
}

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

func TestCLILowersMainReturnExpressionToLLVMReturnCode(t *testing.T) {
	root := cliRepoRoot(t)
	dir := cliTempDir(t, root, "cli-return-code-*")
	input := filepath.Join(dir, "return_code.js")
	output := filepath.Join(dir, "return_code.ll")
	if err := os.WriteFile(input, []byte("function main() { const code = 5; return code * 2; }\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	runJayessCLI(t, root, "compile", "--target=linux-x64", "--emit=llvm", "-o", output, input)
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read LLVM output: %v", err)
	}
	if !strings.Contains(string(content), "ret i32 10") {
		t.Fatalf("expected folded return code in LLVM output, got:\n%s", string(content))
	}
}
