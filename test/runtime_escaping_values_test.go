package test

import (
	"strings"
	"testing"

	"jayess-go/lifetime"
	"jayess-go/lowering"
)

func TestRuntimeEscapingReturnedValueIsPreserved(t *testing.T) {
	plan := runtimeLifetimePlan(t, "function make() {\nconst value = {};\nreturn value;\n}")

	assertPreservedNotCleaned(t, plan, "value")
}

func TestRuntimeEscapingContainerStoredValuesArePreserved(t *testing.T) {
	plan := runtimeLifetimePlan(t, "function make() {\nconst objectValue = {};\nconst arrayValue = {};\nreturn { objectValue, arrayValue: [arrayValue] };\n}")

	assertPreservedNotCleaned(t, plan, "objectValue")
	assertPreservedNotCleaned(t, plan, "arrayValue")
}

func TestRuntimeEscapingClosureCaptureIsPreserved(t *testing.T) {
	plan := runtimeLifetimePlan(t, "function make() {\nconst value = {};\nreturn () => value;\n}")

	assertPreservedNotCleaned(t, plan, "value")
}

func TestRuntimeEscapingModuleStateAssignmentIsPreserved(t *testing.T) {
	plan := runtimeLifetimePlan(t, "var moduleState = null;\nfunction write() {\nconst value = {};\nmoduleState = value;\n}")

	assertPreservedNotCleaned(t, plan, "value")
}

func TestRuntimeEscapingNativeHandleValuesRequireOwnedData(t *testing.T) {
	plan := runtimeLifetimePlan(t, "function attach(nativeHandle) {\nconst value = {};\nnativeHandle.store(value);\n}")

	assertPreservedNotCleaned(t, plan, "value")
	doc := readRuntimeOwnershipDoc(t)
	for _, text := range []string{"long-lived native state", "copied string buffer", "copied byte buffer"} {
		if !strings.Contains(doc, text) {
			t.Fatalf("expected runtime ownership docs to mention %q", text)
		}
	}
}

func runtimeLifetimePlan(t *testing.T, source string) lifetime.Plan {
	t.Helper()
	return lifetime.BuildScopeExitPlan(parseProgram(t, source))
}

func assertPreservedNotCleaned(t *testing.T, plan lifetime.Plan, binding string) {
	t.Helper()
	if hasScopeExitCleanup(plan, binding) {
		t.Fatalf("did not expect escaping value %s to use scope cleanup", binding)
	}
	if !hasRuntimePreserveOp(lowering.LowerPreserveOps(plan), binding) {
		t.Fatalf("expected escaping value %s to be preserved", binding)
	}
}
