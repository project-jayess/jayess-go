package test

import (
	"testing"

	"jayess-go/lifetime"
	"jayess-go/lowering"
)

func TestLoweringInsertsCleanupForNonEscapingScopeExitValue(t *testing.T) {
	program := parseProgram(t, "function make() {\nconst value = {};\nreturn 1;\n}")

	ops := lowering.LowerCleanupOps(lifetime.BuildScopeExitPlan(program))
	op := findCleanupOp(t, ops, "value")
	if op.Line != 2 || op.Column != 1 {
		t.Fatalf("expected cleanup op at 2:1, got %d:%d", op.Line, op.Column)
	}
	if op.ScopeDepth != 1 {
		t.Fatalf("expected cleanup op depth 1, got %d", op.ScopeDepth)
	}
}

func TestLoweringDoesNotInsertCleanupForReturnedValue(t *testing.T) {
	program := parseProgram(t, "function make() {\nconst value = {};\nreturn value;\n}")

	plan := lifetime.BuildScopeExitPlan(program)
	ops := lowering.LowerCleanupOps(plan)
	if hasCleanupOp(ops, "value") {
		t.Fatalf("did not expect cleanup op for returned value")
	}
	if !hasPreserveOp(lowering.LowerPreserveOps(plan), "value") {
		t.Fatalf("expected preserve op for returned value")
	}
}

func TestLoweringDoesNotInsertCleanupForClosureCapturedValue(t *testing.T) {
	program := parseProgram(t, "function make() {\nconst value = {};\nreturn () => value;\n}")

	plan := lifetime.BuildScopeExitPlan(program)
	if hasCleanupOp(lowering.LowerCleanupOps(plan), "value") {
		t.Fatalf("did not expect cleanup op for closure-captured value")
	}
	if !hasPreserveOp(lowering.LowerPreserveOps(plan), "value") {
		t.Fatalf("expected preserve op for closure-captured value")
	}
}

func TestLoweringDoesNotInsertCleanupForUnknownCallArgument(t *testing.T) {
	program := parseProgram(t, "function run() {\nconst value = {};\nexternal(value);\n}")

	plan := lifetime.BuildScopeExitPlan(program)
	if hasCleanupOp(lowering.LowerCleanupOps(plan), "value") {
		t.Fatalf("did not expect cleanup op for unknown call argument")
	}
	if !hasPreserveOp(lowering.LowerPreserveOps(plan), "value") {
		t.Fatalf("expected preserve op for unknown call argument")
	}
}

func findCleanupOp(t *testing.T, ops []lowering.CleanupOp, binding string) lowering.CleanupOp {
	t.Helper()
	for _, op := range ops {
		if op.Binding == binding {
			return op
		}
	}
	t.Fatalf("expected cleanup op for %s", binding)
	return lowering.CleanupOp{}
}

func hasCleanupOp(ops []lowering.CleanupOp, binding string) bool {
	for _, op := range ops {
		if op.Binding == binding {
			return true
		}
	}
	return false
}

func hasPreserveOp(ops []lowering.PreserveOp, binding string) bool {
	for _, op := range ops {
		if op.Binding == binding {
			return true
		}
	}
	return false
}
