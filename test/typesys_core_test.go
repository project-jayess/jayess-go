package test

import (
	"testing"

	"jayess-go/typesys"
)

func TestTypeSystemCoreTypesAreDeclared(t *testing.T) {
	expected := []string{
		"number",
		"string",
		"boolean",
		"bigint",
		"void",
		"null",
		"undefined",
		"any",
		"unknown",
		"never",
		"object",
		"array",
		"tuple",
	}
	for _, name := range expected {
		if !typesys.HasCoreType(name) {
			t.Fatalf("expected core type %s", name)
		}
	}
}

func TestTypeSystemCoreTypeKinds(t *testing.T) {
	cases := map[string]typesys.Kind{
		"number":    typesys.KindPrimitive,
		"string":    typesys.KindPrimitive,
		"boolean":   typesys.KindPrimitive,
		"bigint":    typesys.KindPrimitive,
		"void":      typesys.KindVoid,
		"null":      typesys.KindNullish,
		"undefined": typesys.KindNullish,
		"any":       typesys.KindTop,
		"unknown":   typesys.KindTop,
		"never":     typesys.KindBottom,
		"object":    typesys.KindObject,
		"array":     typesys.KindArray,
		"tuple":     typesys.KindTuple,
	}
	for name, kind := range cases {
		coreType, ok := typesys.LookupCoreType(name)
		if !ok {
			t.Fatalf("expected core type %s", name)
		}
		if coreType.Kind != kind {
			t.Fatalf("expected core type %s to have kind %s, got %s", name, kind, coreType.Kind)
		}
	}
}

func TestTypeSystemRejectsUnknownCoreTypeLookup(t *testing.T) {
	if typesys.HasCoreType("Promise") {
		t.Fatal("did not expect Promise to be a core type")
	}
	if coreType, ok := typesys.LookupCoreType("Promise"); ok || coreType.Name != "" {
		t.Fatalf("expected empty lookup result for unknown core type, got %#v", coreType)
	}
}
