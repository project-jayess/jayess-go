package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeClassGetterInvokesWithInstanceThis(t *testing.T) {
	class := jayessruntime.NewClass("Box")
	class.DefineField("value", jayessruntime.NewString("stored"))
	class.DefineAccessor("computed", jayessruntime.NewFunction("get computed", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, ok := frame.This().Object()
		if !ok {
			t.Fatal("expected getter this object")
		}
		value, _ := instance.GetNamedProperty("value")
		return value
	}), nil)

	instance := class.Construct()
	value, ok := instance.GetAccessorProperty("computed")

	if !ok || value.Text() != "stored" {
		t.Fatalf("unexpected getter value: %#v exists=%v", value, ok)
	}
}

func TestRuntimeClassSetterInvokesWithInstanceThis(t *testing.T) {
	class := jayessruntime.NewClass("Box")
	class.DefineAccessor("computed", nil, jayessruntime.NewFunction("set computed", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, ok := frame.This().Object()
		if !ok {
			t.Fatal("expected setter this object")
		}
		instance.SetNamedProperty("value", frame.Argument(0))
		return jayessruntime.Undefined()
	}))

	instance := class.Construct()
	if !instance.SetAccessorProperty("computed", jayessruntime.NewString("updated")) {
		t.Fatal("expected setter to run")
	}
	value, ok := instance.GetNamedProperty("value")
	if !ok || value.Text() != "updated" {
		t.Fatalf("unexpected setter value: %#v exists=%v", value, ok)
	}
}

func TestRuntimeClassAccessorsResolveFromParent(t *testing.T) {
	base := jayessruntime.NewClass("Base")
	base.DefineAccessor("kind", jayessruntime.NewFunction("get kind", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		return jayessruntime.NewString("base")
	}), nil)
	derived := jayessruntime.NewClassExtends("Derived", base)

	value, ok := derived.Construct().GetAccessorProperty("kind")

	if !ok || value.Text() != "base" {
		t.Fatalf("unexpected inherited accessor value: %#v exists=%v", value, ok)
	}
}

func TestRuntimeClassCheckedPrivateAccessRequiresOwner(t *testing.T) {
	owner := jayessruntime.NewClass("Owner")
	owner.DefinePrivateField("secret", jayessruntime.NewString("value"))
	other := jayessruntime.NewClass("Other")
	instance := owner.Construct()

	value, ok := instance.GetCheckedPrivateField(owner, "secret")
	if !ok || value.Text() != "value" {
		t.Fatalf("unexpected checked private value: %#v exists=%v", value, ok)
	}
	if value, ok := instance.GetCheckedPrivateField(other, "secret"); ok || value.Kind() != jayessruntime.UndefinedValue {
		t.Fatalf("expected rejected private access, got %#v exists=%v", value, ok)
	}
}

func TestRuntimeClassCheckedPrivateMutationRequiresOwner(t *testing.T) {
	owner := jayessruntime.NewClass("Owner")
	owner.DefinePrivateField("secret", jayessruntime.NewString("old"))
	instance := owner.Construct()

	if !instance.SetCheckedPrivateField(owner, "secret", jayessruntime.NewString("new")) {
		t.Fatal("expected checked private mutation")
	}
	value, ok := instance.GetPrivateField("secret")
	if !ok || value.Text() != "new" {
		t.Fatalf("unexpected private mutation: %#v exists=%v", value, ok)
	}
	if instance.SetCheckedPrivateField(jayessruntime.NewClass("Other"), "secret", jayessruntime.NewString("bad")) {
		t.Fatal("expected rejected private mutation")
	}
}
