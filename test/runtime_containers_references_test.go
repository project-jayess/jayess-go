package test

import (
	"strings"
	"testing"

	"jayess-go/escape"
)

func TestRuntimeContainerStorageExtendsLifetime(t *testing.T) {
	plan := runtimeLifetimePlan(t, "function make() {\nconst objectValue = {};\nconst arrayValue = {};\nreturn { objectValue, arrayValue: [arrayValue] };\n}")

	assertPreservedNotCleaned(t, plan, "objectValue")
	assertPreservedNotCleaned(t, plan, "arrayValue")
}

func TestRuntimeNestedContainersPropagateEscapeBehavior(t *testing.T) {
	report := escape.Analyze(parseProgram(t, "function make() {\nconst value = {};\nreturn { nested: [{ value }] };\n}"))

	if !report.Escapes("value") {
		t.Fatalf("expected nested container value to escape")
	}
}

func TestRuntimeContainerRemovalAndSharedReferencesAreDocumented(t *testing.T) {
	doc := readRuntimeOwnershipDoc(t)
	required := []string{
		"Removing a value from an object or array must release only the container's reference",
		"still reference the same value must keep it valid",
	}

	for _, text := range required {
		if !strings.Contains(doc, text) {
			t.Fatalf("runtime ownership docs missing %q", text)
		}
	}
}

func TestRuntimeSharedReferencesAndAliasesPreserveEscapingValue(t *testing.T) {
	plan := runtimeLifetimePlan(t, "function make() {\nconst value = {};\nconst alias = value;\nreturn { first: value, second: alias };\n}")

	assertPreservedNotCleaned(t, plan, "value")
	assertPreservedNotCleaned(t, plan, "alias")
}
