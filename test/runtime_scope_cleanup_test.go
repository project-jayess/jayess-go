package test

import (
	"testing"

	"jayess-go/lifetime"
	"jayess-go/lowering"
)

func TestRuntimeScopeCleanupKeepsModuleValuesOutOfLocalCleanup(t *testing.T) {
	program := parseProgram(t, "const moduleValue = {};\nfunction run() {\nconst localValue = {};\nreturn 1;\n}")

	plan := lifetime.BuildScopeExitPlan(program)
	if hasScopeExitCleanup(plan, "moduleValue") {
		t.Fatalf("did not expect module value to be cleaned as a local")
	}
	if !hasRuntimePreserveOp(lowering.LowerPreserveOps(plan), "moduleValue") {
		t.Fatalf("expected module value to be preserved")
	}
	if !hasScopeExitCleanup(plan, "localValue") {
		t.Fatalf("expected function local value to use scope cleanup")
	}
}

func TestRuntimeScopeCleanupRulesHaveLoweringCoverage(t *testing.T) {
	program := parseProgram(t, "function run(active) {\nconst returned = {};\nif (active) {\nconst temporary = {};\nreturn returned;\n}\nwhile (active) {\nconst loopValue = {};\nbreak;\n}\nconst errorValue = {};\nthrow returned;\n}")

	plan := lifetime.BuildScopeExitPlan(program)
	ops := lowering.LowerControlFlowCleanupOps(program, plan)
	findControlFlowCleanup(t, ops, "temporary", lowering.CleanupPathReturn)
	findControlFlowCleanup(t, ops, "loopValue", lowering.CleanupPathBreak)
	findControlFlowCleanup(t, ops, "errorValue", lowering.CleanupPathThrow)
	if hasCleanupOp(lowering.LowerCleanupOps(plan), "returned") {
		t.Fatalf("did not expect returned value to be cleaned before caller receives it")
	}
	if !hasRuntimePreserveOp(lowering.LowerPreserveOps(plan), "returned") {
		t.Fatalf("expected returned value to be preserved")
	}
}

func hasRuntimePreserveOp(ops []lowering.PreserveOp, binding string) bool {
	for _, op := range ops {
		if op.Binding == binding {
			return true
		}
	}
	return false
}
