package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeValueKindsCoverDynamicJayessValues(t *testing.T) {
	for _, kind := range []jayessruntime.ValueKind{
		jayessruntime.UndefinedValue,
		jayessruntime.NullValue,
		jayessruntime.BooleanValue,
		jayessruntime.NumberValue,
		jayessruntime.StringValue,
		jayessruntime.ObjectValue,
		jayessruntime.ArrayValue,
		jayessruntime.FunctionValue,
		jayessruntime.NativeValue,
	} {
		if !jayessruntime.IsValueKind(kind) {
			t.Fatalf("expected value kind %s to be registered", kind)
		}
	}
}

func TestRuntimeDynamicValueLayoutsAreValid(t *testing.T) {
	if diagnostics := jayessruntime.ValidateDynamicValueLayouts(); len(diagnostics) != 0 {
		t.Fatalf("unexpected layout diagnostics: %#v", diagnostics)
	}
}

func TestRuntimeValueLayoutUsesImmediateAndManagedPayloads(t *testing.T) {
	number, ok := jayessruntime.LayoutForKind(jayessruntime.NumberValue)
	if !ok {
		t.Fatal("missing number layout")
	}
	if number.Payload != jayessruntime.Float64Payload || number.HeapAllocated {
		t.Fatalf("number should be immediate float64 payload, got %#v", number)
	}
	object, ok := jayessruntime.LayoutForKind(jayessruntime.ObjectValue)
	if !ok {
		t.Fatal("missing object layout")
	}
	if object.Payload != jayessruntime.PointerPayload || !object.HeapAllocated || !object.Managed {
		t.Fatalf("object should be managed pointer payload, got %#v", object)
	}
}

func TestRuntimeValueConstructorSymbolsAreModeled(t *testing.T) {
	for _, name := range []string{
		"jayess_value_undefined",
		"jayess_value_null",
		"jayess_value_from_boolean",
		"jayess_value_from_number",
		"jayess_value_from_string_copy",
		"jayess_object_new",
		"jayess_array_new",
		"jayess_function_new",
		"jayess_value_from_native_handle",
	} {
		if !jayessruntime.HasValueRuntimeSymbol(name) {
			t.Fatalf("expected value runtime symbol %s", name)
		}
	}
}

func TestRuntimeValueConstructorsPreservePrimitivePayloads(t *testing.T) {
	if value := jayessruntime.Undefined(); value.Kind() != jayessruntime.UndefinedValue {
		t.Fatalf("expected undefined value, got %#v", value.Kind())
	}
	if value := jayessruntime.Null(); value.Kind() != jayessruntime.NullValue {
		t.Fatalf("expected null value, got %#v", value.Kind())
	}
	if value := jayessruntime.NewBoolean(true); value.Kind() != jayessruntime.BooleanValue || !value.Bool() {
		t.Fatalf("expected true boolean value, got kind=%s bool=%v", value.Kind(), value.Bool())
	}
	if value := jayessruntime.NewNumber(42.5); value.Kind() != jayessruntime.NumberValue || value.Number() != 42.5 {
		t.Fatalf("expected numeric value, got kind=%s number=%v", value.Kind(), value.Number())
	}
	if value := jayessruntime.NewString("jayess"); value.Kind() != jayessruntime.StringValue || value.Text() != "jayess" {
		t.Fatalf("expected string value, got kind=%s text=%q", value.Kind(), value.Text())
	}
}

func TestRuntimeValueConstructorsNormalizeNilManagedPayloads(t *testing.T) {
	objectValue := jayessruntime.NewObjectValue(nil)
	if object, ok := objectValue.Object(); !ok || object == nil {
		t.Fatalf("expected nil object constructor to allocate object, got %#v ok=%v", object, ok)
	}
	arrayValue := jayessruntime.NewArrayValue(nil)
	if array, ok := arrayValue.Array(); !ok || array == nil {
		t.Fatalf("expected nil array constructor to allocate array, got %#v ok=%v", array, ok)
	}
	functionValue := jayessruntime.NewFunctionValue(nil)
	if function, ok := functionValue.Function(); !ok || function == nil {
		t.Fatalf("expected nil function constructor to allocate function, got %#v ok=%v", function, ok)
	}
}

func TestRuntimeValueAccessorsRejectMismatchedKinds(t *testing.T) {
	number := jayessruntime.NewNumber(1)
	if object, ok := number.Object(); ok || object != nil {
		t.Fatalf("number should not expose object payload, got %#v ok=%v", object, ok)
	}
	if array, ok := number.Array(); ok || array != nil {
		t.Fatalf("number should not expose array payload, got %#v ok=%v", array, ok)
	}
	if function, ok := number.Function(); ok || function != nil {
		t.Fatalf("number should not expose function payload, got %#v ok=%v", function, ok)
	}
	native := jayessruntime.NewNativeValue("handle")
	if native.Kind() != jayessruntime.NativeValue || native.Native() != "handle" {
		t.Fatalf("expected native payload, got kind=%s native=%#v", native.Kind(), native.Native())
	}
}
