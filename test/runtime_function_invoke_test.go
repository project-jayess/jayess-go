package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeBindFunctionPreservesThisAndPrependedArguments(t *testing.T) {
	target := jayessruntime.NewFunction("join", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		if frame.This().Text() != "bound-this" {
			t.Fatalf("unexpected bound this: %#v", frame.This())
		}
		return jayessruntime.NewString(frame.Argument(0).Text() + "-" + frame.Argument(1).Text())
	})

	boundValue := jayessruntime.BindFunction(
		jayessruntime.NewFunctionValue(target),
		jayessruntime.NewString("bound-this"),
		jayessruntime.NewString("first"),
	)
	result := jayessruntime.CallFunction(boundValue, jayessruntime.NewString("ignored"), jayessruntime.NewString("second"))

	if result.Text() != "first-second" {
		t.Fatalf("unexpected bound call result: %#v", result)
	}
}

func TestRuntimeCallMethodInvokesWithExplicitThis(t *testing.T) {
	target := jayessruntime.NewFunction("thisText", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		return frame.This()
	})

	result := jayessruntime.CallMethod(
		jayessruntime.NewFunctionValue(target),
		jayessruntime.NewString("call-this"),
	)

	if result.Text() != "call-this" {
		t.Fatalf("unexpected call result: %#v", result)
	}
}

func TestRuntimeApplyFunctionUsesArrayArguments(t *testing.T) {
	target := jayessruntime.NewFunction("sum", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		return jayessruntime.NewNumber(frame.Argument(0).Number() + frame.Argument(1).Number())
	})
	args := jayessruntime.NewArray(jayessruntime.NewNumber(4), jayessruntime.NewNumber(6))

	result := jayessruntime.ApplyFunction(
		jayessruntime.NewFunctionValue(target),
		jayessruntime.Undefined(),
		jayessruntime.NewArrayValue(args),
	)

	if result.Number() != 10 {
		t.Fatalf("unexpected apply result: %#v", result)
	}
}

func TestRuntimeBoundArrowStillUsesLexicalThis(t *testing.T) {
	arrow := jayessruntime.NewArrowFunction("arrow", jayessruntime.NewString("lexical"), nil, func(frame jayessruntime.CallFrame) jayessruntime.Value {
		return frame.This()
	})

	bound := jayessruntime.BindFunction(
		jayessruntime.NewFunctionValue(arrow),
		jayessruntime.NewString("bound"),
	)
	result := jayessruntime.CallFunction(bound, jayessruntime.NewString("call"))

	if result.Text() != "lexical" {
		t.Fatalf("expected lexical this for bound arrow, got %#v", result)
	}
}

func TestRuntimeBindCallApplyIgnoreNonFunctions(t *testing.T) {
	if value := jayessruntime.BindFunction(jayessruntime.NewNumber(1), jayessruntime.Undefined()); value.Kind() != jayessruntime.UndefinedValue {
		t.Fatalf("expected undefined bind result, got %#v", value)
	}
	if value := jayessruntime.CallMethod(jayessruntime.NewNumber(1), jayessruntime.Undefined()); value.Kind() != jayessruntime.UndefinedValue {
		t.Fatalf("expected undefined call result, got %#v", value)
	}
	if value := jayessruntime.ApplyFunction(jayessruntime.NewNumber(1), jayessruntime.Undefined(), jayessruntime.Undefined()); value.Kind() != jayessruntime.UndefinedValue {
		t.Fatalf("expected undefined apply result, got %#v", value)
	}
}
