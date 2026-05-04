package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeArrowFunctionUsesLexicalThis(t *testing.T) {
	lexicalThis := jayessruntime.NewObjectValue(nil)
	callThis := jayessruntime.NewString("call-site")
	arrow := jayessruntime.NewArrowFunction("readThis", lexicalThis, nil, func(frame jayessruntime.CallFrame) jayessruntime.Value {
		return frame.This()
	})

	value := arrow.Call(callThis)

	if value.Kind() != jayessruntime.ObjectValue {
		t.Fatalf("expected lexical object this, got %#v", value)
	}
}

func TestRuntimeArrowFunctionKeepsClosureEnvironment(t *testing.T) {
	environment := jayessruntime.NewClosureEnvironment()
	environment.Set("value", jayessruntime.NewString("captured"))
	arrow := jayessruntime.NewArrowFunction("read", jayessruntime.Undefined(), environment, func(frame jayessruntime.CallFrame) jayessruntime.Value {
		closure, ok := frame.Closure()
		if !ok {
			t.Fatal("expected closure in arrow frame")
		}
		value, _ := closure.Get("value")
		return value
	})

	value := arrow.Call(jayessruntime.NewObjectValue(nil))

	if value.Text() != "captured" {
		t.Fatalf("unexpected arrow closure value: %#v", value)
	}
}

func TestRuntimeArrowMetadataExposesLexicalThis(t *testing.T) {
	lexicalThis := jayessruntime.NewString("outer")
	arrow := jayessruntime.NewArrowFunction("arrow", lexicalThis, nil, nil)

	this, ok := arrow.LexicalThis()
	if !arrow.IsArrow() || !ok || this.Text() != "outer" {
		t.Fatalf("unexpected arrow metadata: isArrow=%v this=%#v ok=%v", arrow.IsArrow(), this, ok)
	}
	regular := jayessruntime.NewFunction("regular", nil)
	if regular.IsArrow() {
		t.Fatal("regular function should not report arrow metadata")
	}
}
