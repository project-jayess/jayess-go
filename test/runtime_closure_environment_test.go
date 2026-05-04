package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeClosureEnvironmentStoresAndReadsCaptures(t *testing.T) {
	environment := jayessruntime.NewClosureEnvironment()
	environment.Set("count", jayessruntime.NewNumber(1))
	environment.Set("name", jayessruntime.NewString("jayess"))

	count, ok := environment.Get("count")
	if !ok || count.Number() != 1 {
		t.Fatalf("unexpected count capture: %#v exists=%v", count, ok)
	}
	name, ok := environment.Get("name")
	if !ok || name.Text() != "jayess" {
		t.Fatalf("unexpected name capture: %#v exists=%v", name, ok)
	}
}

func TestRuntimeClosureEnvironmentPreservesSharedMutation(t *testing.T) {
	environment := jayessruntime.NewClosureEnvironment()
	environment.Set("count", jayessruntime.NewNumber(0))
	increment := jayessruntime.NewClosureFunction("increment", environment, func(frame jayessruntime.CallFrame) jayessruntime.Value {
		closure, ok := frame.Closure()
		if !ok {
			t.Fatal("expected closure environment")
		}
		count, _ := closure.Get("count")
		next := jayessruntime.NewNumber(count.Number() + 1)
		closure.Set("count", next)
		return next
	})
	read := jayessruntime.NewClosureFunction("read", environment, func(frame jayessruntime.CallFrame) jayessruntime.Value {
		closure, _ := frame.Closure()
		count, _ := closure.Get("count")
		return count
	})

	increment.Call(jayessruntime.Undefined())
	increment.Call(jayessruntime.Undefined())
	value := read.Call(jayessruntime.Undefined())

	if value.Number() != 2 {
		t.Fatalf("expected shared closure mutation result 2, got %#v", value)
	}
}

func TestRuntimeClosureFunctionExposesEnvironmentInCallFrame(t *testing.T) {
	environment := jayessruntime.NewClosureEnvironmentFromCaptures(map[string]jayessruntime.Value{
		"message": jayessruntime.NewString("captured"),
	})
	function := jayessruntime.NewClosureFunction("message", environment, func(frame jayessruntime.CallFrame) jayessruntime.Value {
		closure, ok := frame.Closure()
		if !ok {
			t.Fatal("expected closure environment in call frame")
		}
		value, _ := closure.Get("message")
		return value
	})

	value := function.Call(jayessruntime.Undefined())

	if value.Text() != "captured" {
		t.Fatalf("unexpected closure result: %#v", value)
	}
}

func TestRuntimeClosureEnvironmentCloneDoesNotShareSlots(t *testing.T) {
	environment := jayessruntime.NewClosureEnvironment()
	environment.Set("value", jayessruntime.NewString("original"))
	clone := environment.Clone()
	clone.Set("value", jayessruntime.NewString("clone"))

	original, _ := environment.Get("value")
	copied, _ := clone.Get("value")
	if original.Text() != "original" || copied.Text() != "clone" {
		t.Fatalf("unexpected cloned captures: original=%#v clone=%#v", original, copied)
	}
}
