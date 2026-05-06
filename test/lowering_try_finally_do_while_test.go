package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringAppliesFinallySideEffectsBeforeDoWhileContinue(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; do { try { code = 92; continue; code = 2; } finally { code++; } } while (false); return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 93 {
		t.Fatalf("expected do-while continue/finally side-effect return code 93, got %d", value)
	}
}

func TestLoweringUsesFinallyContinueOverDoWhileBreak(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; do { try { code = 94; break; code = 2; } finally { code++; continue; } } while (false); return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 95 {
		t.Fatalf("expected finally continue over break return code 95, got %d", value)
	}
}

func TestLoweringUsesFinallyBreakOverDoWhileContinue(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; do { try { code = 96; continue; code = 2; } finally { code++; break; } } while (true); return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 97 {
		t.Fatalf("expected finally break over continue return code 97, got %d", value)
	}
}
