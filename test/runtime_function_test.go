package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeFunctionValueWrapsCallableFunction(t *testing.T) {
	function := jayessruntime.NewFunction("add", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		left := frame.Argument(0).Number()
		right := frame.Argument(1).Number()
		return jayessruntime.NewNumber(left + right)
	})

	value := jayessruntime.NewFunctionValue(function)
	callable, ok := value.Function()
	if !ok || callable.Name() != "add" {
		t.Fatalf("expected function value, got %#v ok=%v", callable, ok)
	}

	result := callable.Call(jayessruntime.Undefined(), jayessruntime.NewNumber(2), jayessruntime.NewNumber(3))
	if result.Kind() != jayessruntime.NumberValue || result.Number() != 5 {
		t.Fatalf("unexpected call result: %#v", result)
	}
}

func TestRuntimeFunctionCallPreservesThisAndArguments(t *testing.T) {
	this := jayessruntime.NewObjectValue(nil)
	function := jayessruntime.NewFunction("inspect", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		if frame.This().Kind() != jayessruntime.ObjectValue {
			t.Fatalf("expected this object, got %#v", frame.This())
		}
		if frame.ArgumentCount() != 2 {
			t.Fatalf("expected two arguments, got %d", frame.ArgumentCount())
		}
		return frame.Argument(1)
	})

	result := jayessruntime.CallFunction(
		jayessruntime.NewFunctionValue(function),
		this,
		jayessruntime.NewString("skip"),
		jayessruntime.NewString("use"),
	)

	if result.Kind() != jayessruntime.StringValue || result.Text() != "use" {
		t.Fatalf("unexpected inspected result: %#v", result)
	}
}

func TestRuntimeCallFrameCopiesArguments(t *testing.T) {
	arguments := []jayessruntime.Value{jayessruntime.NewString("original")}
	frame := jayessruntime.NewCallFrame(jayessruntime.Undefined(), arguments...)
	arguments[0] = jayessruntime.NewString("changed")

	if value := frame.Argument(0); value.Text() != "original" {
		t.Fatalf("expected copied argument, got %#v", value)
	}
	copied := frame.Arguments()
	copied[0] = jayessruntime.NewString("mutated")
	if value := frame.Argument(0); value.Text() != "original" {
		t.Fatalf("expected frame arguments to remain immutable, got %#v", value)
	}
}

func TestRuntimeCallFrameSupportsDefaultArguments(t *testing.T) {
	frame := jayessruntime.NewCallFrame(
		jayessruntime.Undefined(),
		jayessruntime.NewString("provided"),
		jayessruntime.Undefined(),
	)

	if !frame.HasArgument(0) {
		t.Fatal("expected first argument to be present")
	}
	if frame.HasArgument(1) {
		t.Fatal("expected explicit undefined to use default fallback")
	}
	if value := frame.ArgumentOrDefault(0, jayessruntime.NewString("fallback")); value.Text() != "provided" {
		t.Fatalf("expected provided argument, got %#v", value)
	}
	if value := frame.ArgumentOrDefault(1, jayessruntime.NewString("fallback")); value.Text() != "fallback" {
		t.Fatalf("expected fallback for undefined argument, got %#v", value)
	}
	if value := frame.ArgumentOrDefault(4, jayessruntime.NewNumber(4)); value.Number() != 4 {
		t.Fatalf("expected fallback for missing argument, got %#v", value)
	}
}

func TestRuntimeCallFrameMaterializesRestArguments(t *testing.T) {
	frame := jayessruntime.NewCallFrame(
		jayessruntime.Undefined(),
		jayessruntime.NewString("first"),
		jayessruntime.NewString("rest-a"),
		jayessruntime.NewString("rest-b"),
	)

	rest := frame.RestArguments(1)
	values := rest.Values()
	if len(values) != 2 || values[0].Text() != "rest-a" || values[1].Text() != "rest-b" {
		t.Fatalf("unexpected rest arguments: %#v", values)
	}
	empty := frame.RestArguments(5)
	if empty.Length() != 0 {
		t.Fatalf("expected empty rest arguments, got %d", empty.Length())
	}
}

func TestRuntimeCallFunctionIgnoresNonFunctions(t *testing.T) {
	result := jayessruntime.CallFunction(jayessruntime.NewNumber(1), jayessruntime.Undefined())
	if result.Kind() != jayessruntime.UndefinedValue {
		t.Fatalf("expected undefined for non-function call, got %#v", result)
	}
}
