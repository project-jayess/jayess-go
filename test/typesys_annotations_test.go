package test

import (
	"testing"

	"jayess-go/typesys"
)

func TestTypeSystemAnnotationTargets(t *testing.T) {
	annotations := []typesys.Annotation{
		typesys.VariableType("total", "number"),
		typesys.ParameterType("name", "string", false),
		typesys.ReturnType("void"),
		typesys.PropertyType("enabled", "boolean", true, true),
	}
	for _, annotation := range annotations {
		if !typesys.ValidateAnnotation(annotation) {
			t.Fatalf("expected valid annotation: %#v", annotation)
		}
	}
	if !hasAnnotationTarget(typesys.AnnotationTargets(), typesys.PropertyAnnotation) {
		t.Fatal("expected property annotation target")
	}
}

func TestTypeSystemRejectsIncompleteAnnotations(t *testing.T) {
	if typesys.ValidateAnnotation(typesys.Annotation{Target: typesys.VariableAnnotation, TypeName: "number"}) {
		t.Fatal("expected variable annotation without name to be invalid")
	}
	if typesys.ValidateAnnotation(typesys.Annotation{Target: typesys.ReturnAnnotation}) {
		t.Fatal("expected return annotation without type to be invalid")
	}
}

func hasAnnotationTarget(values []typesys.AnnotationTarget, want typesys.AnnotationTarget) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
