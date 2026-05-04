package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeClassExecutableInheritanceAndDispatch(t *testing.T) {
	base := jayessruntime.NewClass("Animal")
	base.DefineField("kind", jayessruntime.NewString("animal"))
	base.DefineConstructor(jayessruntime.NewFunction("Animal", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, _ := frame.This().Object()
		instance.SetNamedProperty("name", frame.Argument(0))
		return jayessruntime.Undefined()
	}))
	base.DefineMethod("describe", jayessruntime.NewFunction("Animal.describe", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, _ := frame.This().Object()
		name, _ := instance.GetNamedProperty("name")
		kind, _ := instance.GetNamedProperty("kind")
		return jayessruntime.NewString(kind.Text() + ":" + name.Text())
	}))

	dog := jayessruntime.NewClassExtends("Dog", base)
	dog.DefineField("kind", jayessruntime.NewString("dog"))
	dog.DefineConstructor(jayessruntime.NewFunction("Dog", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, _ := frame.This().Object()
		dog.ConstructSuper(instance, frame.Argument(0))
		instance.SetNamedProperty("sound", jayessruntime.NewString("woof"))
		return jayessruntime.Undefined()
	}))
	dog.DefineMethod("describe", jayessruntime.NewFunction("Dog.describe", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, _ := frame.This().Object()
		baseDescription := instance.CallSuperMethod("describe")
		sound, _ := instance.GetNamedProperty("sound")
		return jayessruntime.NewString(baseDescription.Text() + ":" + sound.Text())
	}))

	instance := dog.Construct(jayessruntime.NewString("Mochi"))
	result := instance.CallMethod("describe")

	if result.Text() != "dog:Mochi:woof" {
		t.Fatalf("unexpected derived dispatch result: %#v", result)
	}
}

func TestRuntimeClassExecutableInheritedAccessorAndPrivateState(t *testing.T) {
	base := jayessruntime.NewClass("BaseCounter")
	base.DefinePrivateField("count", jayessruntime.NewNumber(1))
	base.DefineAccessor("value", jayessruntime.NewFunction("get value", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, _ := frame.This().Object()
		value, _ := instance.GetCheckedPrivateField(base, "count")
		return value
	}), jayessruntime.NewFunction("set value", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, _ := frame.This().Object()
		instance.SetCheckedPrivateField(base, "count", frame.Argument(0))
		return jayessruntime.Undefined()
	}))
	base.DefineMethod("inc", jayessruntime.NewFunction("inc", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, _ := frame.This().Object()
		current, _ := instance.GetAccessorProperty("value")
		next := jayessruntime.NewNumber(current.Number() + 1)
		instance.SetAccessorProperty("value", next)
		return next
	}))

	derived := jayessruntime.NewClassExtends("DerivedCounter", base)
	derived.DefineMethod("incTwice", jayessruntime.NewFunction("incTwice", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		instance, _ := frame.This().Object()
		instance.CallMethod("inc")
		return instance.CallMethod("inc")
	}))

	instance := derived.Construct()
	result := instance.CallMethod("incTwice")
	final, ok := instance.GetAccessorProperty("value")

	if result.Number() != 3 || !ok || final.Number() != 3 {
		t.Fatalf("unexpected inherited accessor/private result: result=%#v final=%#v exists=%v", result, final, ok)
	}
}
