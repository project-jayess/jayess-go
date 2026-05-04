package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeClassConstructPassesInstanceAsThis(t *testing.T) {
	class := jayessruntime.NewClass("Point")
	class.DefineConstructor(jayessruntime.NewFunction("constructor", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, ok := frame.This().Object()
		if !ok {
			t.Fatal("expected constructor this object")
		}
		instance.SetNamedProperty("x", frame.Argument(0))
		return jayessruntime.Undefined()
	}))

	instance := class.Construct(jayessruntime.NewNumber(7))

	value, ok := instance.GetNamedProperty("x")
	if !ok || value.Number() != 7 {
		t.Fatalf("unexpected constructed field: %#v exists=%v", value, ok)
	}
}

func TestRuntimeClassConstructInitializesInheritedFields(t *testing.T) {
	base := jayessruntime.NewClass("Base")
	base.DefineField("baseField", jayessruntime.NewString("base"))
	derived := jayessruntime.NewClassExtends("Derived", base)
	derived.DefineField("derivedField", jayessruntime.NewString("derived"))

	instance := derived.Construct()

	if value, ok := instance.GetNamedProperty("baseField"); !ok || value.Text() != "base" {
		t.Fatalf("unexpected base field: %#v exists=%v", value, ok)
	}
	if value, ok := instance.GetNamedProperty("derivedField"); !ok || value.Text() != "derived" {
		t.Fatalf("unexpected derived field: %#v exists=%v", value, ok)
	}
	owner, ok := instance.Class()
	if !ok || owner.Name() != "Derived" {
		t.Fatalf("unexpected derived instance class: %#v exists=%v", owner, ok)
	}
}

func TestRuntimeClassConstructSuperInvokesParentConstructor(t *testing.T) {
	base := jayessruntime.NewClass("Base")
	base.DefineConstructor(jayessruntime.NewFunction("base constructor", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, _ := frame.This().Object()
		instance.SetNamedProperty("baseValue", frame.Argument(0))
		return jayessruntime.Undefined()
	}))
	derived := jayessruntime.NewClassExtends("Derived", base)
	derived.DefineConstructor(jayessruntime.NewFunction("derived constructor", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, _ := frame.This().Object()
		derived.ConstructSuper(instance, frame.Argument(0))
		instance.SetNamedProperty("derivedValue", jayessruntime.NewString("ready"))
		return jayessruntime.Undefined()
	}))

	instance := derived.Construct(jayessruntime.NewString("from-super"))

	if value, ok := instance.GetNamedProperty("baseValue"); !ok || value.Text() != "from-super" {
		t.Fatalf("unexpected super field: %#v exists=%v", value, ok)
	}
	if value, ok := instance.GetNamedProperty("derivedValue"); !ok || value.Text() != "ready" {
		t.Fatalf("unexpected derived constructor field: %#v exists=%v", value, ok)
	}
}

func TestRuntimeClassConstructorMetadataIsAvailable(t *testing.T) {
	base := jayessruntime.NewClass("Base")
	derived := jayessruntime.NewClassExtends("Derived", base)
	constructor := jayessruntime.NewFunction("constructor", nil)
	derived.DefineConstructor(constructor)

	parent, ok := derived.Parent()
	if !ok || parent.Name() != "Base" {
		t.Fatalf("unexpected parent: %#v exists=%v", parent, ok)
	}
	actual, ok := derived.Constructor()
	if !ok || actual.Name() != "constructor" {
		t.Fatalf("unexpected constructor: %#v exists=%v", actual, ok)
	}
}
