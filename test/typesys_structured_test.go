package test

import (
	"testing"

	"jayess-go/typesys"
)

func TestTypeSystemStructuredInterfaceAndAlias(t *testing.T) {
	user := typesys.InterfaceType("User", []typesys.Property{
		{Name: "id", TypeName: "number", Readonly: true},
		{Name: "name", TypeName: "string", Optional: true},
	})
	if user.Kind != typesys.StructuredInterface {
		t.Fatalf("expected interface kind, got %s", user.Kind)
	}
	if len(user.Properties) != 2 {
		t.Fatalf("expected interface properties, got %#v", user.Properties)
	}
	if !user.Properties[0].Readonly {
		t.Fatalf("expected readonly property metadata")
	}
	if !user.Properties[1].Optional {
		t.Fatalf("expected optional property metadata")
	}

	alias := typesys.AliasType("UserId", "number")
	if alias.Kind != typesys.StructuredAlias || alias.Target != "number" {
		t.Fatalf("expected alias metadata, got %#v", alias)
	}
}

func TestTypeSystemStructuredFunctionAndCallable(t *testing.T) {
	params := []typesys.Parameter{
		{Name: "value", TypeName: "string"},
		{Name: "limit", TypeName: "number", Optional: true},
	}
	fn := typesys.FunctionType(params, "boolean")
	if fn.Kind != typesys.StructuredFunction {
		t.Fatalf("expected function kind, got %s", fn.Kind)
	}
	if fn.ReturnType != "boolean" || len(fn.Parameters) != 2 {
		t.Fatalf("expected function signature metadata, got %#v", fn)
	}

	callable := typesys.CallableType(params, "void")
	if callable.Kind != typesys.StructuredCallable {
		t.Fatalf("expected callable kind, got %s", callable.Kind)
	}
	if callable.ReturnType != "void" {
		t.Fatalf("expected callable return type, got %q", callable.ReturnType)
	}
}

func TestTypeSystemStructuredObjectIndexSignatures(t *testing.T) {
	object := typesys.ObjectType(
		[]typesys.Property{{Name: "length", TypeName: "number", Readonly: true}},
		[]typesys.IndexSignature{{
			KeyName:   "key",
			KeyType:   "string",
			ValueType: "unknown",
			Readonly:  true,
		}},
	)
	if object.Kind != typesys.StructuredObject {
		t.Fatalf("expected object kind, got %s", object.Kind)
	}
	if len(object.IndexSignatures) != 1 {
		t.Fatalf("expected index signature metadata, got %#v", object.IndexSignatures)
	}
	if !object.IndexSignatures[0].Readonly {
		t.Fatalf("expected readonly index signature metadata")
	}
}
