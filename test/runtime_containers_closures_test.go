package test

import (
	"strings"
	"testing"

	"jayess-go/lifetime"
)

func TestRuntimeContainerReferencesPreserveStoredValues(t *testing.T) {
	plan := runtimeLifetimePlan(t, "function make() {\nconst objectRef = {};\nconst arrayRef = [];\nreturn { objectRef, arrayRef };\n}")

	assertPreservedNotCleaned(t, plan, "objectRef")
	assertPreservedNotCleaned(t, plan, "arrayRef")
}

func TestRuntimeClosureEnvironmentRemainsValid(t *testing.T) {
	plan := lifetime.BuildScopeExitPlan(parseProgram(t, "function make() {\nconst value = {};\nreturn () => value;\n}"))

	environment := findClosureEnvironment(t, plan, "value")
	capture := findClosureCapture(t, environment, "value")
	if environment.Allocation != "heap" {
		t.Fatalf("expected heap closure environment, got %q", environment.Allocation)
	}
	if !capture.LifetimeExtended || !capture.NonDangling {
		t.Fatalf("expected closure capture to remain valid, got %#v", capture)
	}
}

func TestRuntimeContainerAndClosureCleanupRulesAreDocumented(t *testing.T) {
	doc := readRuntimeOwnershipDoc(t)
	required := []string{
		"Object property insertion and array element insertion must retain",
		"Replacing an object property or array element must release",
		"Closure environment cleanup must release captured values",
		"exactly once",
	}

	for _, text := range required {
		if !strings.Contains(doc, text) {
			t.Fatalf("runtime ownership docs missing %q", text)
		}
	}
}
