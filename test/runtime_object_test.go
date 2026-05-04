package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeObjectAllocatesAndStoresNamedProperties(t *testing.T) {
	object := jayessruntime.NewObject()
	object.SetNamedProperty("name", jayessruntime.NewString("jayess"))
	object.SetNamedProperty("count", jayessruntime.NewNumber(2))

	name, ok := object.GetNamedProperty("name")
	if !ok || name.Kind() != jayessruntime.StringValue || name.Text() != "jayess" {
		t.Fatalf("unexpected name property: %#v exists=%v", name, ok)
	}
	count, ok := object.GetNamedProperty("count")
	if !ok || count.Kind() != jayessruntime.NumberValue || count.Number() != 2 {
		t.Fatalf("unexpected count property: %#v exists=%v", count, ok)
	}
}

func TestRuntimeObjectSupportsComputedPropertyKeys(t *testing.T) {
	object := jayessruntime.NewObject()
	object.SetProperty(jayessruntime.NewString("dynamic"), jayessruntime.NewBoolean(true))
	object.SetProperty(jayessruntime.NewNumber(7), jayessruntime.NewString("seven"))
	object.SetProperty(jayessruntime.NewBoolean(false), jayessruntime.Null())

	if !object.HasNamedProperty("dynamic") || !object.HasNamedProperty("7") || !object.HasNamedProperty("false") {
		t.Fatalf("expected computed property keys in %#v", object.Keys())
	}
	value, ok := object.GetProperty(jayessruntime.NewNumber(7))
	if !ok || value.Text() != "seven" {
		t.Fatalf("expected numeric computed property lookup, got %#v exists=%v", value, ok)
	}
}

func TestRuntimeObjectPreservesInsertionOrderForKeys(t *testing.T) {
	object := jayessruntime.NewObject()
	object.SetNamedProperty("first", jayessruntime.NewNumber(1))
	object.SetNamedProperty("second", jayessruntime.NewNumber(2))
	object.SetNamedProperty("first", jayessruntime.NewNumber(3))

	keys := object.Keys()
	if len(keys) != 2 || keys[0] != "first" || keys[1] != "second" {
		t.Fatalf("unexpected key order: %#v", keys)
	}
}

func TestRuntimeObjectDeleteAndClone(t *testing.T) {
	object := jayessruntime.NewObject()
	object.SetNamedProperty("keep", jayessruntime.NewString("yes"))
	object.SetNamedProperty("drop", jayessruntime.NewString("no"))
	if !object.DeleteNamedProperty("drop") {
		t.Fatal("expected delete to remove existing property")
	}
	if object.HasNamedProperty("drop") {
		t.Fatal("expected dropped property to be absent")
	}
	clone := object.Clone()
	clone.SetNamedProperty("keep", jayessruntime.NewString("clone"))
	original, _ := object.GetNamedProperty("keep")
	copied, _ := clone.GetNamedProperty("keep")
	if original.Text() != "yes" || copied.Text() != "clone" {
		t.Fatalf("clone should not mutate original: original=%#v clone=%#v", original, copied)
	}
}

func TestRuntimeObjectValueWrapsAllocatedObject(t *testing.T) {
	value := jayessruntime.NewObjectValue(nil)
	object, ok := value.Object()
	if !ok {
		t.Fatalf("expected object value, got %#v", value)
	}
	object.SetNamedProperty("ready", jayessruntime.NewBoolean(true))
	ready, ok := object.GetNamedProperty("ready")
	if !ok || !ready.Bool() {
		t.Fatalf("expected ready property, got %#v exists=%v", ready, ok)
	}
}
