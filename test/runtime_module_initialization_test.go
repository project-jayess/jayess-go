package test

import (
	"strings"
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeModuleRegistryInitializesInProvidedOrderOnce(t *testing.T) {
	registry := jayessruntime.NewModuleRegistry()
	var calls []string
	registry.Register("main.js", func(bindings *jayessruntime.ModuleBindings) jayessruntime.Value {
		calls = append(calls, "main.js")
		bindings.DefineLocal("ready", jayessruntime.NewBoolean(true))
		return jayessruntime.Undefined()
	})
	registry.Register("dep.js", func(bindings *jayessruntime.ModuleBindings) jayessruntime.Value {
		calls = append(calls, "dep.js")
		return jayessruntime.Undefined()
	})

	first := registry.InitializeModules([]string{"dep.js", "main.js", "dep.js"})
	second := registry.InitializeModules([]string{"main.js", "dep.js"})

	if strings.Join(first, ",") != "dep.js,main.js" {
		t.Fatalf("expected first initialization order dep.js,main.js, got %#v", first)
	}
	if len(second) != 0 {
		t.Fatalf("expected no modules initialized on second pass, got %#v", second)
	}
	if strings.Join(calls, ",") != "dep.js,main.js" {
		t.Fatalf("expected initializer calls once in order, got %#v", calls)
	}

	mainModule, ok := registry.Module("main.js")
	if !ok || !mainModule.Initialized() {
		t.Fatalf("expected initialized main module")
	}
	ready, ok := mainModule.Bindings().Local("ready")
	if !ok {
		t.Fatalf("expected main local binding")
	}
	value, ok := ready.Value()
	if !ok || !value.Bool() {
		t.Fatalf("expected initialized binding value, got %#v ok=%v", value, ok)
	}
}

func TestRuntimeModuleRegistrySkipsMissingAndReentrantModules(t *testing.T) {
	registry := jayessruntime.NewModuleRegistry()
	var calls int
	registry.Register("self.js", func(bindings *jayessruntime.ModuleBindings) jayessruntime.Value {
		calls++
		if registry.InitializeModule("self.js") {
			t.Fatal("expected reentrant initialization to be skipped")
		}
		return jayessruntime.Undefined()
	})

	initialized := registry.InitializeModules([]string{"missing.js", "self.js"})
	if strings.Join(initialized, ",") != "self.js" {
		t.Fatalf("expected only self.js initialized, got %#v", initialized)
	}
	if calls != 1 {
		t.Fatalf("expected one initializer call, got %d", calls)
	}
}
