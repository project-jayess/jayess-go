package test

import (
	"testing"

	"jayess-go/lowering"
)

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
