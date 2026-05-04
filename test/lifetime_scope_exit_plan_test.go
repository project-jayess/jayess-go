package test

import (
	"testing"

	"jayess-go/lifetime"
)

func TestLifetimePlansCleanupForNonEscapingLocalByDefault(t *testing.T) {
	program := parseProgram(t, "function make() {\nconst value = {};\nreturn 1;\n}")

	plan := lifetime.BuildScopeExitPlan(program)
	cleanup := findScopeExitCleanup(t, plan, "value")
	if cleanup.Line != 2 || cleanup.Column != 1 {
		t.Fatalf("expected cleanup at declaration 2:1, got %d:%d", cleanup.Line, cleanup.Column)
	}
	if cleanup.ScopeDepth != 1 {
		t.Fatalf("expected function-local cleanup depth 1, got %d", cleanup.ScopeDepth)
	}
}

func TestLifetimeDoesNotPlanScopeExitCleanupForEscapingLocal(t *testing.T) {
	program := parseProgram(t, "function make() {\nconst value = {};\nreturn value;\n}")

	plan := lifetime.BuildScopeExitPlan(program)
	if hasScopeExitCleanup(plan, "value") {
		t.Fatalf("did not expect escaping value to be cleaned up at scope exit")
	}
}

func TestLifetimeExtendsReturnedLocal(t *testing.T) {
	program := parseProgram(t, "function make() {\nconst value = {};\nreturn value;\n}")

	plan := lifetime.BuildScopeExitPlan(program)
	extended := findExtendedLifetime(t, plan, "value")
	if extended.ScopeDepth != 1 {
		t.Fatalf("expected returned value lifetime depth 1, got %d", extended.ScopeDepth)
	}
}

func TestLifetimeExtendsClosureCapturedLocal(t *testing.T) {
	program := parseProgram(t, "function make() {\nconst value = {};\nreturn () => value;\n}")

	plan := lifetime.BuildScopeExitPlan(program)
	if hasScopeExitCleanup(plan, "value") {
		t.Fatalf("did not expect closure-captured value to be cleaned up at scope exit")
	}
	extended := findExtendedLifetime(t, plan, "value")
	if extended.ScopeDepth != 1 {
		t.Fatalf("expected closure-captured value lifetime depth 1, got %d", extended.ScopeDepth)
	}
}

func TestLifetimePlansCleanupForNestedBlockLocal(t *testing.T) {
	program := parseProgram(t, "function make() {\n{\nconst value = {};\n}\nreturn 1;\n}")

	plan := lifetime.BuildScopeExitPlan(program)
	cleanup := findScopeExitCleanup(t, plan, "value")
	if cleanup.Line != 3 || cleanup.Column != 1 {
		t.Fatalf("expected cleanup at nested declaration 3:1, got %d:%d", cleanup.Line, cleanup.Column)
	}
	if cleanup.ScopeDepth != 2 {
		t.Fatalf("expected nested block cleanup depth 2, got %d", cleanup.ScopeDepth)
	}
}

func TestLifetimePlanUsesLexicalScopeDepthNotUseReachability(t *testing.T) {
	program := parseProgram(t, "function make(condition) {\nconst outer = {};\nif (condition) {\nconst inner = {};\nouter;\ninner;\n}\nreturn 1;\n}")

	plan := lifetime.BuildScopeExitPlan(program)
	outer := findScopeExitCleanup(t, plan, "outer")
	inner := findScopeExitCleanup(t, plan, "inner")
	if outer.ScopeDepth != 1 {
		t.Fatalf("expected outer cleanup depth 1, got %d", outer.ScopeDepth)
	}
	if inner.ScopeDepth != 2 {
		t.Fatalf("expected inner cleanup depth 2, got %d", inner.ScopeDepth)
	}
}

func findScopeExitCleanup(t *testing.T, plan lifetime.Plan, binding string) lifetime.Cleanup {
	t.Helper()
	for _, cleanup := range plan.ScopeExitCleanups {
		if cleanup.Binding == binding {
			return cleanup
		}
	}
	t.Fatalf("expected scope-exit cleanup for %s", binding)
	return lifetime.Cleanup{}
}

func hasScopeExitCleanup(plan lifetime.Plan, binding string) bool {
	for _, cleanup := range plan.ScopeExitCleanups {
		if cleanup.Binding == binding {
			return true
		}
	}
	return false
}

func findExtendedLifetime(t *testing.T, plan lifetime.Plan, binding string) lifetime.ExtendedLifetime {
	t.Helper()
	for _, extended := range plan.ExtendedLifetimes {
		if extended.Binding == binding {
			return extended
		}
	}
	t.Fatalf("expected extended lifetime for %s", binding)
	return lifetime.ExtendedLifetime{}
}
