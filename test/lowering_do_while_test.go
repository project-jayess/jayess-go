package test

import (
	"testing"

	"jayess-go/lowering"
)

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
