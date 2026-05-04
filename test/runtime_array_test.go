package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeArrayAllocatesAndReadsIndexes(t *testing.T) {
	array := jayessruntime.NewArray(jayessruntime.NewString("a"))
	array.SetIndex(1, jayessruntime.NewString("b"))

	first, ok := array.GetIndex(0)
	if !ok || first.Text() != "a" {
		t.Fatalf("unexpected first element: %#v exists=%v", first, ok)
	}
	second, ok := array.GetIndex(1)
	if !ok || second.Text() != "b" {
		t.Fatalf("unexpected second element: %#v exists=%v", second, ok)
	}
}

func TestRuntimeArrayGrowsAndReportsLength(t *testing.T) {
	array := jayessruntime.NewArray()
	array.SetIndex(3, jayessruntime.NewNumber(4))

	if array.Length() != 4 {
		t.Fatalf("expected grown length 4, got %d", array.Length())
	}
	if _, ok := array.GetIndex(2); ok {
		t.Fatal("expected unassigned grown slot to be absent")
	}
	length, ok := array.GetNamedProperty("length")
	if !ok || length.Number() != 4 {
		t.Fatalf("unexpected length property: %#v exists=%v", length, ok)
	}
}

func TestRuntimeArraySupportsComputedIndexProperties(t *testing.T) {
	array := jayessruntime.NewArray()
	array.SetProperty(jayessruntime.NewNumber(2), jayessruntime.NewString("two"))
	array.SetNamedProperty("label", jayessruntime.NewString("items"))

	value, ok := array.GetProperty(jayessruntime.NewString("2"))
	if !ok || value.Text() != "two" {
		t.Fatalf("expected computed index lookup, got %#v exists=%v", value, ok)
	}
	label, ok := array.GetNamedProperty("label")
	if !ok || label.Text() != "items" {
		t.Fatalf("expected named property lookup, got %#v exists=%v", label, ok)
	}
}

func TestRuntimeArrayLengthCanTruncateAndGrow(t *testing.T) {
	array := jayessruntime.NewArray(jayessruntime.NewNumber(1), jayessruntime.NewNumber(2))
	array.SetNamedProperty("length", jayessruntime.NewNumber(1))
	if array.Length() != 1 {
		t.Fatalf("expected truncated length 1, got %d", array.Length())
	}
	if _, ok := array.GetIndex(1); ok {
		t.Fatal("expected truncated element to be absent")
	}
	array.SetNamedProperty("length", jayessruntime.NewNumber(3))
	if array.Length() != 3 {
		t.Fatalf("expected grown length 3, got %d", array.Length())
	}
}

func TestRuntimeArrayValueWrapsAllocatedArray(t *testing.T) {
	value := jayessruntime.NewArrayValue(nil)
	array, ok := value.Array()
	if !ok {
		t.Fatalf("expected array value, got %#v", value)
	}
	array.Push(jayessruntime.NewBoolean(true))
	item, ok := array.GetIndex(0)
	if !ok || !item.Bool() {
		t.Fatalf("expected pushed boolean, got %#v exists=%v", item, ok)
	}
}
