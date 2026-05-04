package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeClassAllocatesInstanceFields(t *testing.T) {
	class := jayessruntime.NewClass("Point")
	class.DefineField("x", jayessruntime.NewNumber(1))
	class.DefineField("y", jayessruntime.NewNumber(2))

	instance := class.NewInstance()

	value, ok := instance.GetNamedProperty("x")
	if !ok || value.Number() != 1 {
		t.Fatalf("unexpected x field: %#v exists=%v", value, ok)
	}
	owner, ok := instance.Class()
	if !ok || owner.Name() != "Point" {
		t.Fatalf("unexpected instance class: %#v exists=%v", owner, ok)
	}
}

func TestRuntimeClassAllocatesPrivateFields(t *testing.T) {
	class := jayessruntime.NewClass("Counter")
	class.DefinePrivateField("count", jayessruntime.NewNumber(0))

	instance := class.NewInstance()

	if instance.HasNamedProperty("count") {
		t.Fatal("private field should not be a public property")
	}
	value, ok := instance.GetPrivateField("count")
	if !ok || value.Number() != 0 {
		t.Fatalf("unexpected private count: %#v exists=%v", value, ok)
	}
}

func TestRuntimeClassStoresMethodsAndPrivateMethods(t *testing.T) {
	class := jayessruntime.NewClass("Methods")
	public := jayessruntime.NewFunction("read", nil)
	private := jayessruntime.NewFunction("secret", nil)
	class.DefineMethod("read", public)
	class.DefinePrivateMethod("secret", private)

	method, ok := class.Method("read")
	if !ok || method.Name() != "read" {
		t.Fatalf("unexpected public method: %#v exists=%v", method, ok)
	}
	privateMethod, ok := class.PrivateMethod("secret")
	if !ok || privateMethod.Name() != "secret" {
		t.Fatalf("unexpected private method: %#v exists=%v", privateMethod, ok)
	}
}

func TestRuntimeClassStoresAccessors(t *testing.T) {
	class := jayessruntime.NewClass("Accessors")
	getter := jayessruntime.NewFunction("get value", nil)
	setter := jayessruntime.NewFunction("set value", nil)
	class.DefineAccessor("value", getter, setter)

	accessor, ok := class.Accessor("value")
	if !ok || accessor.Getter.Name() != "get value" || accessor.Setter.Name() != "set value" {
		t.Fatalf("unexpected accessor: %#v exists=%v", accessor, ok)
	}
}

func TestRuntimeClassInstanceCloneKeepsClassAndPrivateFields(t *testing.T) {
	class := jayessruntime.NewClass("Cloneable")
	class.DefinePrivateField("id", jayessruntime.NewString("one"))
	instance := class.NewInstance()

	clone := instance.Clone()

	owner, ok := clone.Class()
	if !ok || owner.Name() != "Cloneable" {
		t.Fatalf("unexpected cloned class: %#v exists=%v", owner, ok)
	}
	value, ok := clone.GetPrivateField("id")
	if !ok || value.Text() != "one" {
		t.Fatalf("unexpected cloned private field: %#v exists=%v", value, ok)
	}
}
