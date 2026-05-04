package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeClassStoresStaticFieldsInOrder(t *testing.T) {
	class := jayessruntime.NewClass("Statics")
	class.DefineStaticField("version", jayessruntime.NewNumber(1))
	class.DefineStaticField("name", jayessruntime.NewString("jayess"))
	class.DefineStaticField("version", jayessruntime.NewNumber(2))

	version, ok := class.StaticField("version")
	if !ok || version.Number() != 2 {
		t.Fatalf("unexpected static version: %#v exists=%v", version, ok)
	}
	names := class.StaticFieldNames()
	if len(names) != 2 || names[0] != "version" || names[1] != "name" {
		t.Fatalf("unexpected static field order: %#v", names)
	}
}

func TestRuntimeClassRunsStaticBlocksOnce(t *testing.T) {
	class := jayessruntime.NewClass("Blocks")
	class.DefineStaticField("count", jayessruntime.NewNumber(0))
	class.DefineStaticBlock(jayessruntime.NewFunction("static", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		current, _ := class.StaticField("count")
		class.DefineStaticField("count", jayessruntime.NewNumber(current.Number()+1))
		return jayessruntime.Undefined()
	}))

	class.RunStaticBlocks()
	class.RunStaticBlocks()

	count, ok := class.StaticField("count")
	if !ok || count.Number() != 1 {
		t.Fatalf("expected static block to run once, got %#v exists=%v", count, ok)
	}
	if !class.StaticBlocksRan() {
		t.Fatal("expected class to report static blocks ran")
	}
}

func TestRuntimeClassStaticBlockReceivesClassAsThis(t *testing.T) {
	class := jayessruntime.NewClass("Receiver")
	class.DefineStaticBlock(jayessruntime.NewFunction("static", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		received, ok := frame.This().Native().(*jayessruntime.Class)
		if !ok || received.Name() != "Receiver" {
			t.Fatalf("unexpected static this: %#v ok=%v", frame.This().Native(), ok)
		}
		return jayessruntime.Undefined()
	}))

	class.RunStaticBlocks()
}
