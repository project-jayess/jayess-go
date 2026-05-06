package test

import (
	"testing"

	"jayess-go/lowering"
)

func expectControlReturnCode(t *testing.T, source string, want int, context string) {
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

func expectNoControlReturnCode(t *testing.T, source string, context string) {
	t.Helper()

	program := parseProgram(t, source)
	_, ok := lowering.MainReturnCode(program)
	if ok {
		t.Fatalf("expected %s to stay unresolved", context)
	}
}

func TestLoweringSelectsMainReturnCodeFromConstantIfCondition(t *testing.T) {
	expectControlReturnCode(t, `function main() { if (false) { return 1; } else { return 2; } }`, 2, "folded conditional")
}

func TestLoweringDoesNotFoldReturnAfterUnknownIfMayReturn(t *testing.T) {
	expectNoControlReturnCode(t, `function main() { if (ready) { return value; } return 2; }`, "return after unknown if return")
}

func TestLoweringDoesNotFoldReturnAfterUnknownIfMayThrow(t *testing.T) {
	expectNoControlReturnCode(t, `function main() { if (ready) { throw value; } return 2; }`, "return after unknown if throw")
}

func TestLoweringUsesPostfixUpdateTruthinessInCondition(t *testing.T) {
	expectControlReturnCode(t, `function main() { var code = 0; if (code++) { return 9; } return code; }`, 1, "postfix update condition")
}

func TestLoweringUsesPrefixUpdateTruthinessInCondition(t *testing.T) {
	expectControlReturnCode(t, `function main() { var code = 0; if (++code) { return code; } return 9; }`, 1, "prefix update condition")
}

func TestLoweringAppliesBlockSideEffectsBeforeReturn(t *testing.T) {
	expectControlReturnCode(t, `function main() { var code = 1; { code = 21; } return code; }`, 21, "block side-effect")
}

func TestLoweringStopsBlockSideEffectsAtBreakInLoop(t *testing.T) {
	expectControlReturnCode(t, `function main() { var code = 1; while (true) { { code = 22; break; code = 2; } } return code; }`, 22, "block break side-effect")
}

func TestLoweringAppliesNestedForSideEffectsBeforeReturn(t *testing.T) {
	expectControlReturnCode(t, `function main() { var code = 1; { for (code = 25; false; code++) { code = 2; } } return code; }`, 25, "nested for side-effect")
}

func TestLoweringAppliesForInitBeforeUnknownCondition(t *testing.T) {
	expectControlReturnCode(t, `function main() { var code = 1; for (code = 38; ready; code++) { code = 2; } return code; }`, 38, "unknown-condition for init")
}

func TestLoweringAppliesNestedForInitBeforeUnknownCondition(t *testing.T) {
	expectControlReturnCode(t, `function main() { var code = 1; { for (code = 39; ready; code++) { code = 2; } } return code; }`, 39, "nested unknown-condition for init")
}

func TestLoweringAppliesNestedDoWhileSideEffectsBeforeReturn(t *testing.T) {
	expectControlReturnCode(t, `function main() { var code = 1; if (true) { do { code = 26; } while (false); } return code; }`, 26, "nested do while side-effect")
}

func TestLoweringAppliesConstantIfSideEffectsBeforeReturn(t *testing.T) {
	expectControlReturnCode(t, `function main() { var code = 1; if (true) { code = 7; } return code; }`, 7, "constant if side-effect")
}

func TestLoweringAppliesNestedSwitchSideEffectsBeforeReturn(t *testing.T) {
	expectControlReturnCode(t, `function main() { var code = 1; if (true) { switch (1) { case 1: code = 19; break; default: code = 2; } } return code; }`, 19, "nested switch side-effect")
}

func TestLoweringAppliesConstantElseSideEffectsBeforeReturn(t *testing.T) {
	expectControlReturnCode(t, `function main() { var code = 1; if (false) { code = 2; } else { code = 8; } return code; }`, 8, "constant else side-effect")
}
