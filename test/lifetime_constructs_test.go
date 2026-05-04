package test

import (
	"testing"

	"jayess-go/lifetime"
)

func TestLifetimeRulesApplyInsideLoopConstructs(t *testing.T) {
	program := parseProgram(t, `
		function run(values) {
			while (values) {
				const whileValue = {};
			}
			for (const forValue = {}; values; values = null) {
				const bodyValue = {};
			}
		}
	`)

	plan := lifetime.BuildScopeExitPlan(program)
	for _, name := range []string{"whileValue", "forValue", "bodyValue"} {
		if !hasLifetimeCleanup(plan, name) {
			t.Fatalf("expected cleanup for %s", name)
		}
	}
}

func TestLifetimeRulesApplyInsideSwitchConstructs(t *testing.T) {
	program := parseProgram(t, `
		function run(value) {
			switch (value) {
			case 1:
				const caseValue = {};
				break;
			default:
				const defaultValue = {};
			}
		}
	`)

	plan := lifetime.BuildScopeExitPlan(program)
	for _, name := range []string{"caseValue", "defaultValue"} {
		if !hasLifetimeCleanup(plan, name) {
			t.Fatalf("expected cleanup for %s", name)
		}
	}
}

func TestLifetimeRulesApplyInsideTryConstructs(t *testing.T) {
	program := parseProgram(t, `
		function run() {
			try {
				const tryValue = {};
			} catch (error) {
				const catchValue = {};
			} finally {
				const finallyValue = {};
			}
		}
	`)

	plan := lifetime.BuildScopeExitPlan(program)
	for _, name := range []string{"tryValue", "error", "catchValue", "finallyValue"} {
		if !hasLifetimeCleanup(plan, name) {
			t.Fatalf("expected cleanup for %s", name)
		}
	}
}

func hasLifetimeCleanup(plan lifetime.Plan, binding string) bool {
	for _, cleanup := range plan.ScopeExitCleanups {
		if cleanup.Binding == binding {
			return true
		}
	}
	return false
}
