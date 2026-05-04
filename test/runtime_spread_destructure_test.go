package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeObjectSpreadCopiesObjectAndArrayProperties(t *testing.T) {
	object := jayessruntime.NewObject()
	object.SetNamedProperty("name", jayessruntime.NewString("jayess"))
	array := jayessruntime.NewArray(jayessruntime.NewString("first"))
	array.SetNamedProperty("label", jayessruntime.NewString("items"))

	spread := jayessruntime.NewObjectFromSpread(
		jayessruntime.NewObjectValue(object),
		jayessruntime.NewArrayValue(array),
	)

	if value, ok := spread.GetNamedProperty("name"); !ok || value.Text() != "jayess" {
		t.Fatalf("expected object property in spread, got %#v exists=%v", value, ok)
	}
	if value, ok := spread.GetNamedProperty("0"); !ok || value.Text() != "first" {
		t.Fatalf("expected array index property in spread, got %#v exists=%v", value, ok)
	}
	if value, ok := spread.GetNamedProperty("label"); !ok || value.Text() != "items" {
		t.Fatalf("expected array named property in spread, got %#v exists=%v", value, ok)
	}
}

func TestRuntimeObjectRestExcludesSelectedKeys(t *testing.T) {
	object := jayessruntime.NewObject()
	object.SetNamedProperty("keep", jayessruntime.NewString("yes"))
	object.SetNamedProperty("drop", jayessruntime.NewString("no"))

	rest := jayessruntime.ObjectRest(jayessruntime.NewObjectValue(object), "drop")

	if rest.HasNamedProperty("drop") {
		t.Fatal("expected excluded property to be absent")
	}
	if value, ok := rest.GetNamedProperty("keep"); !ok || value.Text() != "yes" {
		t.Fatalf("expected kept property, got %#v exists=%v", value, ok)
	}
}

func TestRuntimeArraySpreadAppendsArrayValuesAndElements(t *testing.T) {
	left := jayessruntime.NewArray(jayessruntime.NewString("a"), jayessruntime.NewString("b"))
	spread := jayessruntime.NewArrayFromSpread(
		jayessruntime.NewArrayValue(left),
		jayessruntime.NewString("c"),
	)

	values := spread.Values()
	if len(values) != 3 || values[0].Text() != "a" || values[1].Text() != "b" || values[2].Text() != "c" {
		t.Fatalf("unexpected spread array values: %#v", values)
	}
}

func TestRuntimeArrayRestCopiesTailValues(t *testing.T) {
	array := jayessruntime.NewArray(
		jayessruntime.NewNumber(1),
		jayessruntime.NewNumber(2),
		jayessruntime.NewNumber(3),
	)

	rest := jayessruntime.ArrayRest(jayessruntime.NewArrayValue(array), 1)

	values := rest.Values()
	if len(values) != 2 || values[0].Number() != 2 || values[1].Number() != 3 {
		t.Fatalf("unexpected rest values: %#v", values)
	}
}

func TestRuntimeDestructuringUsesDefaultsForMissingValues(t *testing.T) {
	object := jayessruntime.NewObject()
	object.SetNamedProperty("name", jayessruntime.NewString("jayess"))
	array := jayessruntime.NewArray(jayessruntime.NewString("first"))

	name := jayessruntime.DestructureObjectProperty(
		jayessruntime.NewObjectValue(object),
		"name",
		jayessruntime.NewString("fallback"),
	)
	missing := jayessruntime.DestructureObjectProperty(
		jayessruntime.NewObjectValue(object),
		"missing",
		jayessruntime.NewString("fallback"),
	)
	first := jayessruntime.DestructureArrayIndex(
		jayessruntime.NewArrayValue(array),
		0,
		jayessruntime.NewString("fallback"),
	)
	empty := jayessruntime.DestructureArrayIndex(
		jayessruntime.NewArrayValue(array),
		2,
		jayessruntime.NewString("fallback"),
	)

	if name.Text() != "jayess" || missing.Text() != "fallback" {
		t.Fatalf("unexpected object destructuring values: name=%#v missing=%#v", name, missing)
	}
	if first.Text() != "first" || empty.Text() != "fallback" {
		t.Fatalf("unexpected array destructuring values: first=%#v empty=%#v", first, empty)
	}
}
