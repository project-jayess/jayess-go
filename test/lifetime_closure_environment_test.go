package test

import (
	"testing"

	"jayess-go/lifetime"
)

func TestLifetimeClosureEnvironmentCapturesVariableByReference(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			return () => value;
		}
	`)

	plan := lifetime.BuildScopeExitPlan(program)
	environment := findClosureEnvironment(t, plan, "value")
	if environment.Allocation != "heap" {
		t.Fatalf("expected closure environment allocation heap, got %q", environment.Allocation)
	}
	capture := findClosureCapture(t, environment, "value")
	if !capture.ByReference {
		t.Fatalf("expected closure capture to reference the binding")
	}
}

func TestLifetimeClosureEnvironmentKeepsCapturedVariableValidAfterOuterExit(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			return () => value;
		}
	`)

	plan := lifetime.BuildScopeExitPlan(program)
	environment := findClosureEnvironment(t, plan, "value")
	capture := findClosureCapture(t, environment, "value")
	if !capture.LifetimeExtended {
		t.Fatalf("expected captured value lifetime to extend after outer scope exit")
	}
	if !capture.NonDangling {
		t.Fatalf("expected captured value reference to be non-dangling")
	}
	if hasScopeExitCleanup(plan, "value") {
		t.Fatalf("did not expect captured value to be cleaned up at outer scope exit")
	}
}

func TestLifetimeClosureEnvironmentRejectsDanglingCapturedReferences(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			return () => value;
		}
	`)

	plan := lifetime.BuildScopeExitPlan(program)
	environment := findClosureEnvironment(t, plan, "value")
	capture := findClosureCapture(t, environment, "value")
	if capture.ByReference && !capture.NonDangling {
		t.Fatalf("closure capture references must not dangle")
	}
}

func TestLifetimeClosureEnvironmentUsesExtendedAllocation(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			return () => value;
		}
	`)

	plan := lifetime.BuildScopeExitPlan(program)
	environment := findClosureEnvironment(t, plan, "value")
	if environment.Allocation != "heap" {
		t.Fatalf("expected captured closure environment to be heap allocated, got %q", environment.Allocation)
	}
}

func TestLifetimeMultipleClosuresShareCapturedVariableSlot(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			const value = {};
			const first = () => value;
			const second = () => value;
			return first;
		}
	`)

	plan := lifetime.BuildScopeExitPlan(program)
	first := findClosureEnvironment(t, plan, "value")
	second := findSecondClosureEnvironment(t, plan, "value")
	firstCapture := findClosureCapture(t, first, "value")
	secondCapture := findClosureCapture(t, second, "value")
	if firstCapture.SharedSlot != secondCapture.SharedSlot {
		t.Fatalf("expected shared capture slot, got %d and %d", firstCapture.SharedSlot, secondCapture.SharedSlot)
	}
}

func TestLifetimeCapturedMutationUsesSharedReference(t *testing.T) {
	program := parseProgram(t, `
		function make() {
			var value = 0;
			const increment = () => {
				value = value + 1;
				return value;
			};
			const read = () => value;
			return read;
		}
	`)

	plan := lifetime.BuildScopeExitPlan(program)
	increment := findClosureEnvironment(t, plan, "value")
	read := findSecondClosureEnvironment(t, plan, "value")
	incrementCapture := findClosureCapture(t, increment, "value")
	readCapture := findClosureCapture(t, read, "value")
	if !incrementCapture.Mutated {
		t.Fatalf("expected mutating closure capture to be marked mutated")
	}
	if incrementCapture.SharedSlot != readCapture.SharedSlot {
		t.Fatalf("expected mutation to share slot with reader, got %d and %d", incrementCapture.SharedSlot, readCapture.SharedSlot)
	}
	if !incrementCapture.ByReference || !readCapture.ByReference {
		t.Fatalf("expected mutation and read captures to use references")
	}
}

func TestLifetimeClosureEnvironmentDoesNotCaptureLocalCopy(t *testing.T) {
	program := parseProgram(t, `
		function make(value) {
			return () => {
				const local = {};
				return local;
			};
		}
	`)

	plan := lifetime.BuildScopeExitPlan(program)
	for _, environment := range plan.ClosureEnvironments {
		for _, capture := range environment.Captures {
			if capture.Binding == "local" {
				t.Fatalf("did not expect closure-local binding to be captured")
			}
		}
	}
}

func findClosureEnvironment(t *testing.T, plan lifetime.Plan, binding string) lifetime.ClosureEnvironment {
	t.Helper()
	for _, environment := range plan.ClosureEnvironments {
		for _, capture := range environment.Captures {
			if capture.Binding == binding {
				return environment
			}
		}
	}
	t.Fatalf("expected closure environment capturing %s", binding)
	return lifetime.ClosureEnvironment{}
}

func findClosureCapture(t *testing.T, environment lifetime.ClosureEnvironment, binding string) lifetime.ClosureCapture {
	t.Helper()
	for _, capture := range environment.Captures {
		if capture.Binding == binding {
			return capture
		}
	}
	t.Fatalf("expected closure capture for %s", binding)
	return lifetime.ClosureCapture{}
}

func findSecondClosureEnvironment(t *testing.T, plan lifetime.Plan, binding string) lifetime.ClosureEnvironment {
	t.Helper()
	foundFirst := false
	for _, environment := range plan.ClosureEnvironments {
		for _, capture := range environment.Captures {
			if capture.Binding == binding {
				if foundFirst {
					return environment
				}
				foundFirst = true
			}
		}
	}
	t.Fatalf("expected second closure environment capturing %s", binding)
	return lifetime.ClosureEnvironment{}
}
