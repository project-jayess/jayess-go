package test

import (
	"testing"

	"jayess-go/lowering"
)

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
