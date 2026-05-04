package test

import (
	"testing"

	"jayess-go/lifetime"
	"jayess-go/lowering"
)

func TestLoweringEmitsCleanupOnNormalBlockExit(t *testing.T) {
	ops := lowerControlFlowCleanups(t, "function run() {\n{\nconst value = {};\n}\n}")

	op := findControlFlowCleanup(t, ops, "value", lowering.CleanupPathNormal)
	if op.Line != 3 || op.Column != 1 {
		t.Fatalf("expected declaration at 3:1, got %d:%d", op.Line, op.Column)
	}
}

func TestLoweringEmitsCleanupBeforeEarlyReturn(t *testing.T) {
	ops := lowerControlFlowCleanups(t, "function run() {\nconst value = {};\nreturn 1;\n}")

	op := findControlFlowCleanup(t, ops, "value", lowering.CleanupPathReturn)
	if op.ExitLine != 3 || op.ExitColumn != 1 {
		t.Fatalf("expected return exit at 3:1, got %d:%d", op.ExitLine, op.ExitColumn)
	}
}

func TestLoweringEmitsCleanupBeforeBreakAndContinue(t *testing.T) {
	ops := lowerControlFlowCleanups(t, "function run(active) {\nwhile (active) {\nconst broken = {};\nbreak;\n}\nwhile (active) {\nconst continued = {};\ncontinue;\n}\n}")

	findControlFlowCleanup(t, ops, "broken", lowering.CleanupPathBreak)
	findControlFlowCleanup(t, ops, "continued", lowering.CleanupPathContinue)
}

func TestLoweringEmitsCleanupBeforeThrow(t *testing.T) {
	ops := lowerControlFlowCleanups(t, "function run() {\nconst value = {};\nthrow value;\n}")

	op := findControlFlowCleanup(t, ops, "value", lowering.CleanupPathThrow)
	if op.ExitLine != 3 || op.ExitColumn != 1 {
		t.Fatalf("expected throw exit at 3:1, got %d:%d", op.ExitLine, op.ExitColumn)
	}
}

func TestLoweringDoesNotSkipCleanupThroughNestedControlFlow(t *testing.T) {
	ops := lowerControlFlowCleanups(t, "function run(active, ready) {\nwhile (active) {\nconst outer = {};\nif (ready) {\nconst inner = {};\nbreak;\n}\n}\n}")

	outer := findControlFlowCleanup(t, ops, "outer", lowering.CleanupPathBreak)
	inner := findControlFlowCleanup(t, ops, "inner", lowering.CleanupPathBreak)
	if inner.Line <= outer.Line {
		t.Fatalf("expected inner cleanup to reference nested declaration after outer, got inner line %d outer line %d", inner.Line, outer.Line)
	}
}

func TestLoweringDoesNotDuplicateCleanupForShadowedBindings(t *testing.T) {
	ops := lowerControlFlowCleanups(t, "function run() {\nconst value = {};\n{\nconst value = {};\nreturn 1;\n}\n}")

	assertUniqueControlFlowCleanups(t, ops)
	if countControlFlowCleanups(ops, "value", lowering.CleanupPathReturn) != 2 {
		t.Fatalf("expected cleanup for both shadowed values before return, got %#v", ops)
	}
}

func lowerControlFlowCleanups(t *testing.T, source string) []lowering.ControlFlowCleanupOp {
	t.Helper()
	program := parseProgram(t, source)
	return lowering.LowerControlFlowCleanupOps(program, lifetime.BuildScopeExitPlan(program))
}

func findControlFlowCleanup(t *testing.T, ops []lowering.ControlFlowCleanupOp, binding string, path lowering.CleanupPath) lowering.ControlFlowCleanupOp {
	t.Helper()
	for _, op := range ops {
		if op.Binding == binding && op.Path == path {
			return op
		}
	}
	t.Fatalf("expected %s cleanup for %s in %#v", path, binding, ops)
	return lowering.ControlFlowCleanupOp{}
}

func countControlFlowCleanups(ops []lowering.ControlFlowCleanupOp, binding string, path lowering.CleanupPath) int {
	count := 0
	for _, op := range ops {
		if op.Binding == binding && op.Path == path {
			count++
		}
	}
	return count
}

func assertUniqueControlFlowCleanups(t *testing.T, ops []lowering.ControlFlowCleanupOp) {
	t.Helper()
	seen := map[lowering.ControlFlowCleanupOp]bool{}
	for _, op := range ops {
		if seen[op] {
			t.Fatalf("duplicate cleanup op: %#v in %#v", op, ops)
		}
		seen[op] = true
	}
}
