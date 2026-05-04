package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeFunctionSupportsRecursiveExecution(t *testing.T) {
	environment := jayessruntime.NewClosureEnvironment()
	var factorial *jayessruntime.Function
	factorial = jayessruntime.NewClosureFunction("factorial", environment, func(frame jayessruntime.CallFrame) jayessruntime.Value {
		n := frame.Argument(0).Number()
		if n <= 1 {
			return jayessruntime.NewNumber(1)
		}
		closure, ok := frame.Closure()
		if !ok {
			t.Fatal("expected recursive function closure")
		}
		self, ok := closure.Get("factorial")
		if !ok {
			t.Fatal("expected recursive self binding")
		}
		next := jayessruntime.CallFunction(self, jayessruntime.Undefined(), jayessruntime.NewNumber(n-1))
		return jayessruntime.NewNumber(n * next.Number())
	})
	environment.Set("factorial", jayessruntime.NewFunctionValue(factorial))

	result := factorial.Call(jayessruntime.Undefined(), jayessruntime.NewNumber(5))

	if result.Number() != 120 {
		t.Fatalf("unexpected recursive result: %#v", result)
	}
}

func TestRuntimeFunctionSupportsHigherOrderExecution(t *testing.T) {
	double := jayessruntime.NewFunction("double", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		return jayessruntime.NewNumber(frame.Argument(0).Number() * 2)
	})
	applyTwice := jayessruntime.NewFunction("applyTwice", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		fn := frame.Argument(0)
		first := jayessruntime.CallFunction(fn, jayessruntime.Undefined(), frame.Argument(1))
		return jayessruntime.CallFunction(fn, jayessruntime.Undefined(), first)
	})

	result := applyTwice.Call(
		jayessruntime.Undefined(),
		jayessruntime.NewFunctionValue(double),
		jayessruntime.NewNumber(3),
	)

	if result.Number() != 12 {
		t.Fatalf("unexpected higher-order result: %#v", result)
	}
}

func TestRuntimeFunctionCanReturnFunctionValues(t *testing.T) {
	makeAdder := jayessruntime.NewFunction("makeAdder", func(frame jayessruntime.CallFrame) jayessruntime.Value {
		amount := frame.Argument(0)
		environment := jayessruntime.NewClosureEnvironment()
		environment.Set("amount", amount)
		adder := jayessruntime.NewClosureFunction("adder", environment, func(frame jayessruntime.CallFrame) jayessruntime.Value {
			closure, ok := frame.Closure()
			if !ok {
				t.Fatal("expected adder closure")
			}
			amount, _ := closure.Get("amount")
			return jayessruntime.NewNumber(frame.Argument(0).Number() + amount.Number())
		})
		return jayessruntime.NewFunctionValue(adder)
	})

	adder := makeAdder.Call(jayessruntime.Undefined(), jayessruntime.NewNumber(7))
	result := jayessruntime.CallFunction(adder, jayessruntime.Undefined(), jayessruntime.NewNumber(5))

	if result.Number() != 12 {
		t.Fatalf("unexpected returned function result: %#v", result)
	}
}
