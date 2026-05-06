package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringRestoresCatchBindingShadowAfterDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var err = 81; try { throw value; } catch (err) { err = 2; } return err; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 81 {
		t.Fatalf("expected catch binding shadow restore return code 81, got %d", value)
	}
}

func TestLoweringRestoresDestructuredCatchBindingShadowAfterDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 82; try { throw value; } catch ({ code }) { code = 2; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 82 {
		t.Fatalf("expected destructured catch binding shadow restore return code 82, got %d", value)
	}
}

func TestLoweringClearsCatchBindingAfterDefiniteThrow(t *testing.T) {
	program := parseProgram(t, `function main() { try { throw value; } catch (err) { err = 2; } if (typeof err === "undefined") { return 83; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 83 {
		t.Fatalf("expected catch binding cleanup return code 83, got %d", value)
	}
}

func TestLoweringAppliesFinallyThrowAfterNestedCatchHandlesThrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; try { try { code = 86; throw value; } catch (err) { code++; } finally { code++; throw value; } } catch (outer) { code++; } return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 89 {
		t.Fatalf("expected nested handled-catch finally throw return code 89, got %d", value)
	}
}

func TestLoweringRestoresNestedCatchBindingBeforeFinally(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 90; try { try { throw value; } catch ({ code }) { code = 2; } finally { code++; throw value; } } catch (outer) {} return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 91 {
		t.Fatalf("expected nested catch binding restore before finally return code 91, got %d", value)
	}
}

func TestLoweringRestoresNestedCatchBindingShadowAfterRethrow(t *testing.T) {
	program := parseProgram(t, `function main() { var err = 84; try { try { throw value; } catch (err) { err = 2; throw value; } } catch (outer) {} return err; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 84 {
		t.Fatalf("expected nested catch binding shadow restore return code 84, got %d", value)
	}
}

func TestLoweringRestoresNestedDestructuredCatchBindingShadowAfterRethrow(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 85; try { try { throw value; } catch ({ code }) { code = 2; throw value; } } catch (outer) {} return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 85 {
		t.Fatalf("expected nested destructured catch binding shadow restore return code 85, got %d", value)
	}
}
