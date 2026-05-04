package test

import (
	"testing"

	"jayess-go/typesys"
)

func TestTypeSystemAdvancedUnionIntersectionAndLiteral(t *testing.T) {
	union := typesys.UnionType("string", "number")
	if union.Kind != typesys.AdvancedUnion || len(union.Types) != 2 {
		t.Fatalf("expected union metadata, got %#v", union)
	}

	intersection := typesys.IntersectionType("Named", "Timestamped")
	if intersection.Kind != typesys.AdvancedIntersection || len(intersection.Types) != 2 {
		t.Fatalf("expected intersection metadata, got %#v", intersection)
	}

	literal := typesys.LiteralType("string", "ready")
	if literal.Kind != typesys.AdvancedLiteral {
		t.Fatalf("expected literal kind, got %s", literal.Kind)
	}
	if literal.Literal.TypeName != "string" || literal.Literal.Value != "ready" {
		t.Fatalf("expected literal metadata, got %#v", literal.Literal)
	}
}

func TestTypeSystemAdvancedDiscriminatedUnion(t *testing.T) {
	variants := []typesys.StructuredType{
		typesys.ObjectType([]typesys.Property{{Name: "kind", TypeName: `"ok"`}}, nil),
		typesys.ObjectType([]typesys.Property{{Name: "kind", TypeName: `"err"`}}, nil),
	}
	union := typesys.DiscriminatedUnionType("kind", variants)
	if union.Kind != typesys.AdvancedDiscriminatedUnion {
		t.Fatalf("expected discriminated union kind, got %s", union.Kind)
	}
	if union.Discriminator != "kind" || len(union.Variants) != 2 {
		t.Fatalf("expected discriminated union metadata, got %#v", union)
	}
}

func TestTypeSystemAdvancedGenericsAndConstraints(t *testing.T) {
	params := []typesys.GenericParameter{
		{Name: "T", Constraint: "object"},
		{Name: "K", Constraint: "keyof T", Default: "string"},
	}
	body := typesys.ObjectType([]typesys.Property{{Name: "value", TypeName: "T"}}, nil)
	generic := typesys.GenericType("Box", params, body)
	if generic.Kind != typesys.AdvancedGeneric {
		t.Fatalf("expected generic kind, got %s", generic.Kind)
	}
	if generic.Name != "Box" || len(generic.Parameters) != 2 || generic.Body.Kind != typesys.StructuredObject {
		t.Fatalf("expected generic metadata, got %#v", generic)
	}

	constraint := typesys.GenericConstraint("T", "object")
	if constraint.Kind != typesys.AdvancedConstraint {
		t.Fatalf("expected constraint kind, got %s", constraint.Kind)
	}
	if constraint.Parameters[0].Constraint != "object" {
		t.Fatalf("expected constraint metadata, got %#v", constraint.Parameters)
	}
}

func TestTypeSystemAdvancedEnumMetadata(t *testing.T) {
	enumType := typesys.EnumType("Color", []typesys.EnumMember{
		{Name: "Red", Value: "0"},
		{Name: "Blue", Value: "1"},
	})
	if enumType.Kind != typesys.AdvancedEnum {
		t.Fatalf("expected enum kind, got %s", enumType.Kind)
	}
	if enumType.Name != "Color" || len(enumType.EnumMembers) != 2 {
		t.Fatalf("expected enum metadata, got %#v", enumType)
	}
}
