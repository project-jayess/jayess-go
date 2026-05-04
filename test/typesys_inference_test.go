package test

import (
	"testing"

	"jayess-go/typesys"
)

func TestTypeSystemInfersLocalLiteralTypes(t *testing.T) {
	cases := map[string]string{
		"42":        "number",
		"-3.5":      "number",
		"\"hello\"": "string",
		"true":      "boolean",
		"null":      "null",
		"undefined": "undefined",
		"[1, 2]":    "array",
		"{ a: 1 }":  "object",
	}
	for source, want := range cases {
		inference := typesys.InferLocalLiteral(source)
		if inference.TypeName != want || !inference.Confident {
			t.Fatalf("expected %q to infer %s confidently, got %#v", source, want, inference)
		}
	}
}

func TestTypeSystemInferenceFallsBackToUnknown(t *testing.T) {
	inference := typesys.InferLocalLiteral("callSomething()")
	if inference.TypeName != "unknown" || inference.Confident {
		t.Fatalf("expected unknown non-confident inference, got %#v", inference)
	}
}
