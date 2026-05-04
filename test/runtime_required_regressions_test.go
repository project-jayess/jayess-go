package test

import (
	"strings"
	"testing"

	"jayess-go/lifetime"
	"jayess-go/lowering"
)

func TestRuntimeRegressionReturningLocalObjectAndArrayRemainValid(t *testing.T) {
	objectPlan := runtimeLifetimePlan(t, "function makeObject() {\nconst value = {};\nreturn value;\n}")
	arrayPlan := runtimeLifetimePlan(t, "function makeArray() {\nconst value = [];\nreturn value;\n}")

	assertPreservedNotCleaned(t, objectPlan, "value")
	assertPreservedNotCleaned(t, arrayPlan, "value")
}

func TestRuntimeRegressionStoringLocalValueIntoObjectAndArrayRemainValid(t *testing.T) {
	plan := runtimeLifetimePlan(t, "function make() {\nconst objectValue = {};\nconst arrayValue = {};\nreturn { item: objectValue, values: [arrayValue] };\n}")

	assertPreservedNotCleaned(t, plan, "objectValue")
	assertPreservedNotCleaned(t, plan, "arrayValue")
}

func TestRuntimeRegressionClosureCaptureAfterOuterReturnRemainsValid(t *testing.T) {
	plan := lifetime.BuildScopeExitPlan(parseProgram(t, "function make() {\nconst value = {};\nreturn () => value;\n}"))

	environment := findClosureEnvironment(t, plan, "value")
	capture := findClosureCapture(t, environment, "value")
	if !capture.LifetimeExtended || !capture.NonDangling {
		t.Fatalf("expected closure capture to remain valid after outer return, got %#v", capture)
	}
}

func TestRuntimeRegressionScopeCleanupAndAbruptCleanupStillRun(t *testing.T) {
	program := parseProgram(t, "function run(active) {\n{\nconst scoped = {};\n}\nif (active) {\nconst returned = {};\nreturn 1;\n}\nwhile (active) {\nconst looped = {};\nbreak;\n}\nwhile (active) {\nconst continued = {};\ncontinue;\n}\n}")
	plan := lifetime.BuildScopeExitPlan(program)
	ops := lowering.LowerControlFlowCleanupOps(program, plan)

	findControlFlowCleanup(t, ops, "scoped", lowering.CleanupPathNormal)
	findControlFlowCleanup(t, ops, "returned", lowering.CleanupPathReturn)
	findControlFlowCleanup(t, ops, "looped", lowering.CleanupPathBreak)
	findControlFlowCleanup(t, ops, "continued", lowering.CleanupPathContinue)
}

func TestRuntimeRegressionReplacementAndManagedHandleSafetyAreDocumented(t *testing.T) {
	doc := readRuntimeOwnershipDoc(t)
	required := []string{
		"Replacing an object property or array element must release",
		"Repeated close on a managed native handle is safe",
		"Using a closed managed native handle must report a runtime error",
	}

	for _, text := range required {
		if !strings.Contains(doc, text) {
			t.Fatalf("runtime ownership docs missing %q", text)
		}
	}
}
