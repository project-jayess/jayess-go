package test

import (
	"os"
	"strings"
	"testing"
)

func TestRuntimeOwnershipDocsCoverValueHelperFamilies(t *testing.T) {
	doc := readRuntimeOwnershipDoc(t)
	required := []string{
		"`jayess_value_from_*`",
		"`jayess_value_from_bytes_copy(...)`",
		"`jayess_value_as_object(...)`",
		"`jayess_value_as_array(...)`",
		"`jayess_value_as_string(...)`",
		"`jayess_value_to_string_copy(...)`",
		"`jayess_value_to_bytes_copy(...)`",
		"`jayess_value_from_native_handle(...)`",
		"`jayess_value_as_native_handle(...)`",
		"`jayess_value_from_managed_native_handle(...)`",
		"`jayess_value_close_native_handle(...)`",
	}

	for _, text := range required {
		if !strings.Contains(doc, text) {
			t.Fatalf("runtime ownership docs missing %s", text)
		}
	}
}

func TestRuntimeOwnershipDocsDefineOwnershipTerms(t *testing.T) {
	doc := readRuntimeOwnershipDoc(t)
	required := []string{
		"Owned value",
		"Borrowed value",
		"Copied buffer",
		"Retained value",
		"Closed value",
		"valid only during the current native call",
		"Release it with `jayess_string_free(...)`",
		"Release it with `jayess_bytes_free(...)`",
	}

	for _, text := range required {
		if !strings.Contains(doc, text) {
			t.Fatalf("runtime ownership docs missing %q", text)
		}
	}
}

func TestRuntimeOwnershipDocsCoverDynamicObjectsAndArrays(t *testing.T) {
	doc := readRuntimeOwnershipDoc(t)
	required := []string{
		"## Dynamic Objects",
		"ordered property table",
		"computed keys",
		"must not move it in enumeration order",
		"copies every enumerable property except the excluded keys",
		"## Dynamic Arrays",
		"indexed slots plus an ordinary named-property table",
		"The `length` property reflects the current slot count",
		"holes that read as `undefined`",
		"Array `for...in` enumeration yields present numeric indexes first",
		"materialize holes as `undefined` values",
	}

	for _, text := range required {
		if !strings.Contains(doc, text) {
			t.Fatalf("runtime ownership docs missing %q", text)
		}
	}
}

func readRuntimeOwnershipDoc(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile("../docs/runtime_ownership.md")
	if err != nil {
		t.Fatalf("read runtime ownership docs: %v", err)
	}
	return string(content)
}
