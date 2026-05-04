package test

import (
	"testing"

	"jayess-go/escape"
	"jayess-go/lifetime"
	"jayess-go/lowering"
)

func TestRuntimeValidationCoversScopeExitEscapeClosureContainerAndReturn(t *testing.T) {
	program := parseProgram(t, "function make(active) {\nconst returned = {};\nconst stored = {};\nconst captured = {};\nconst closure = () => captured;\nif (active) {\nconst temporary = {};\nreturn { returned, stored: [stored], closure };\n}\nreturn returned;\n}")
	plan := lifetime.BuildScopeExitPlan(program)

	assertPreservedNotCleaned(t, plan, "returned")
	assertPreservedNotCleaned(t, plan, "stored")
	assertPreservedNotCleaned(t, plan, "captured")
	findControlFlowCleanup(t, lowering.LowerControlFlowCleanupOps(program, plan), "temporary", lowering.CleanupPathReturn)
}

func TestRuntimeValidationCoversCrossFunctionLifetimeBehavior(t *testing.T) {
	plan := runtimeLifetimePlan(t, "function make() {\nconst value = {};\nreturn value;\n}\nfunction run() {\nconst kept = make();\nreturn kept;\n}")

	assertPreservedNotCleaned(t, plan, "value")
	assertPreservedNotCleaned(t, plan, "kept")
}

func TestRuntimeValidationStressNestedScopesAndClosures(t *testing.T) {
	program := parseProgram(t, "function make() {\nconst root = {};\n{\nconst nested = {};\nreturn () => [root, nested];\n}\n}")
	report := escape.Analyze(program)
	plan := lifetime.BuildScopeExitPlan(program)

	for _, binding := range []string{"root", "nested"} {
		if !report.Escapes(binding) {
			t.Fatalf("expected nested closure binding %s to escape", binding)
		}
		assertPreservedNotCleaned(t, plan, binding)
		findClosureEnvironment(t, plan, binding)
	}
}
