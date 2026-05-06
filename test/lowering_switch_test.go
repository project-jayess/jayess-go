package test

import (
	"testing"

	"jayess-go/lowering"
)

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
