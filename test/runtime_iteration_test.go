package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeForInKeysEnumeratesObjectProperties(t *testing.T) {
	object := jayessruntime.NewObject()
	object.SetNamedProperty("first", jayessruntime.NewNumber(1))
	object.SetNamedProperty("second", jayessruntime.NewNumber(2))

	keys := jayessruntime.ForInKeys(jayessruntime.NewObjectValue(object))

	if len(keys) != 2 || keys[0] != "first" || keys[1] != "second" {
		t.Fatalf("unexpected object for-in keys: %#v", keys)
	}
}

func TestRuntimeForInKeysEnumeratesArrayIndexesAndProperties(t *testing.T) {
	array := jayessruntime.NewArray()
	array.SetIndex(2, jayessruntime.NewString("two"))
	array.SetNamedProperty("label", jayessruntime.NewString("items"))

	keys := jayessruntime.ForInKeys(jayessruntime.NewArrayValue(array))

	if len(keys) != 2 || keys[0] != "2" || keys[1] != "label" {
		t.Fatalf("unexpected array for-in keys: %#v", keys)
	}
}

func TestRuntimeForInKeyValuesWrapsKeysAsStrings(t *testing.T) {
	object := jayessruntime.NewObject()
	object.SetNamedProperty("name", jayessruntime.NewString("jayess"))

	values := jayessruntime.ForInKeyValues(jayessruntime.NewObjectValue(object))

	if len(values) != 1 || values[0].Kind() != jayessruntime.StringValue || values[0].Text() != "name" {
		t.Fatalf("unexpected for-in key values: %#v", values)
	}
}

func TestRuntimeForOfValuesEnumeratesArraySlots(t *testing.T) {
	array := jayessruntime.NewArray(jayessruntime.NewString("a"))
	array.SetIndex(2, jayessruntime.NewString("c"))

	values := jayessruntime.ForOfValues(jayessruntime.NewArrayValue(array))

	if len(values) != 3 {
		t.Fatalf("expected three values including sparse hole, got %#v", values)
	}
	if values[0].Text() != "a" || values[1].Kind() != jayessruntime.UndefinedValue || values[2].Text() != "c" {
		t.Fatalf("unexpected array for-of values: %#v", values)
	}
}

func TestRuntimeForOfValuesEnumeratesObjectValues(t *testing.T) {
	object := jayessruntime.NewObject()
	object.SetNamedProperty("first", jayessruntime.NewNumber(1))
	object.SetNamedProperty("second", jayessruntime.NewNumber(2))

	values := jayessruntime.ForOfValues(jayessruntime.NewObjectValue(object))

	if len(values) != 2 || values[0].Number() != 1 || values[1].Number() != 2 {
		t.Fatalf("unexpected object for-of values: %#v", values)
	}
}

func TestRuntimeIterationIgnoresUnsupportedValues(t *testing.T) {
	if keys := jayessruntime.ForInKeys(jayessruntime.NewNumber(1)); len(keys) != 0 {
		t.Fatalf("expected no keys for number, got %#v", keys)
	}
	if values := jayessruntime.ForOfValues(jayessruntime.NewString("text")); len(values) != 0 {
		t.Fatalf("expected no values for string, got %#v", values)
	}
}
