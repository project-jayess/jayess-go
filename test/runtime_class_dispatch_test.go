package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeClassDispatchCallsInstanceMethodWithThis(t *testing.T) {
	class := jayessruntime.NewClass("Greeter")
	class.DefineField("name", jayessruntime.NewString("Jayess"))
	class.DefineMethod("greet", jayessruntime.NewFunction("greet", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, ok := frame.This().Object()
		if !ok {
			t.Fatal("expected method this object")
		}
		name, _ := instance.GetNamedProperty("name")
		return jayessruntime.NewString("hello " + name.Text())
	}))

	instance := class.Construct()
	result := instance.CallMethod("greet")

	if result.Text() != "hello Jayess" {
		t.Fatalf("unexpected method result: %#v", result)
	}
}

func TestRuntimeClassDispatchFindsInheritedMethods(t *testing.T) {
	base := jayessruntime.NewClass("Base")
	base.DefineMethod("kind", jayessruntime.NewFunction("kind", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		return jayessruntime.NewString("base")
	}))
	derived := jayessruntime.NewClassExtends("Derived", base)

	instance := derived.Construct()
	result := instance.CallMethod("kind")

	if result.Text() != "base" {
		t.Fatalf("unexpected inherited method result: %#v", result)
	}
}

func TestRuntimeClassDispatchUsesOverrideBeforeParent(t *testing.T) {
	base := jayessruntime.NewClass("Base")
	base.DefineMethod("kind", jayessruntime.NewFunction("base kind", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		return jayessruntime.NewString("base")
	}))
	derived := jayessruntime.NewClassExtends("Derived", base)
	derived.DefineMethod("kind", jayessruntime.NewFunction("derived kind", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		return jayessruntime.NewString("derived")
	}))

	instance := derived.Construct()
	result := instance.CallMethod("kind")

	if result.Text() != "derived" {
		t.Fatalf("unexpected override result: %#v", result)
	}
}

func TestRuntimeClassDispatchCallsSuperMethod(t *testing.T) {
	base := jayessruntime.NewClass("Base")
	base.DefineMethod("describe", jayessruntime.NewFunction("base describe", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		return jayessruntime.NewString("base")
	}))
	derived := jayessruntime.NewClassExtends("Derived", base)
	derived.DefineMethod("describe", jayessruntime.NewFunction("derived describe", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, _ := frame.This().Object()
		baseValue := instance.CallSuperMethod("describe")
		return jayessruntime.NewString(baseValue.Text() + "+derived")
	}))

	instance := derived.Construct()
	result := instance.CallMethod("describe")

	if result.Text() != "base+derived" {
		t.Fatalf("unexpected super dispatch result: %#v", result)
	}
}

func TestRuntimeClassDispatchMissingMethodReturnsUndefined(t *testing.T) {
	class := jayessruntime.NewClass("Empty")
	instance := class.Construct()

	result := instance.CallMethod("missing")

	if result.Kind() != jayessruntime.UndefinedValue {
		t.Fatalf("expected undefined for missing method, got %#v", result)
	}
}
