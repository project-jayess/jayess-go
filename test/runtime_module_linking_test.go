package test

import (
	"strings"
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeModuleLinkingExecutesLocalDefaultReExportAndNamespaceImports(t *testing.T) {
	registry := jayessruntime.NewModuleRegistry()
	var calls []string

	registry.Register("math.js", func(bindings *jayessruntime.ModuleBindings) jayessruntime.Value {
		calls = append(calls, "math.js")
		total := bindings.DefineLocal("total", jayessruntime.NewNumber(2))
		main := bindings.DefineLocal("main", jayessruntime.NewString("math-main"))
		bindings.BindExport("total", total)
		bindings.BindExport("default", main)
		return jayessruntime.Undefined()
	})
	registry.Register("barrel.js", func(bindings *jayessruntime.ModuleBindings) jayessruntime.Value {
		calls = append(calls, "barrel.js")
		mathModule, _ := registry.Module("math.js")
		if !bindings.BindReExportFrom("sum", mathModule.Bindings(), "total") {
			t.Fatal("expected named re-export from math.js")
		}
		if !bindings.BindReExportFrom("main", mathModule.Bindings(), "default") {
			t.Fatal("expected default re-export from math.js")
		}
		if !bindings.BindNamespaceExport("math", mathModule.Bindings()) {
			t.Fatal("expected namespace re-export from math.js")
		}
		return jayessruntime.Undefined()
	})
	registry.Register("app.js", func(bindings *jayessruntime.ModuleBindings) jayessruntime.Value {
		calls = append(calls, "app.js")
		barrelModule, _ := registry.Module("barrel.js")
		if !bindings.BindImportFrom("sum", barrelModule.Bindings(), "sum") {
			t.Fatal("expected local named import from barrel.js")
		}
		if !bindings.BindImportFrom("main", barrelModule.Bindings(), "main") {
			t.Fatal("expected local default re-export import from barrel.js")
		}
		if !bindings.BindNamespaceImport("tools", barrelModule.Bindings()) {
			t.Fatal("expected namespace import from barrel.js")
		}
		sum, _ := bindings.Import("sum")
		main, _ := bindings.Import("main")
		bindings.BindExport("sum", sum)
		bindings.BindExport("main", main)
		return jayessruntime.Undefined()
	})

	initialized := registry.InitializeModules([]string{"math.js", "barrel.js", "app.js"})
	if strings.Join(initialized, ",") != "math.js,barrel.js,app.js" {
		t.Fatalf("unexpected initialization order: %#v", initialized)
	}
	if strings.Join(calls, ",") != "math.js,barrel.js,app.js" {
		t.Fatalf("unexpected initializer calls: %#v", calls)
	}

	appModule, _ := registry.Module("app.js")
	assertModuleNumberExport(t, appModule.Bindings(), "sum", 2)
	assertModuleStringExport(t, appModule.Bindings(), "main", "math-main")

	toolsBinding, ok := appModule.Bindings().Import("tools")
	if !ok {
		t.Fatal("expected namespace import binding")
	}
	toolsValue, ok := toolsBinding.Value()
	if !ok {
		t.Fatal("expected namespace import value")
	}
	tools, ok := toolsValue.Object()
	if !ok {
		t.Fatalf("expected namespace import object, got %#v", toolsValue)
	}
	mathNamespaceValue, ok := tools.GetNamedProperty("math")
	if !ok {
		t.Fatal("expected namespace re-export property")
	}
	mathNamespace, ok := mathNamespaceValue.Object()
	if !ok {
		t.Fatalf("expected namespace re-export object, got %#v", mathNamespaceValue)
	}
	total, ok := mathNamespace.GetNamedProperty("total")
	if !ok || total.Number() != 2 {
		t.Fatalf("expected namespace total 2, got %#v ok=%v", total, ok)
	}
}

func assertModuleNumberExport(t *testing.T, bindings *jayessruntime.ModuleBindings, name string, expected float64) {
	t.Helper()
	binding, ok := bindings.Export(name)
	if !ok {
		t.Fatalf("expected export %s", name)
	}
	value, ok := binding.Value()
	if !ok || value.Number() != expected {
		t.Fatalf("expected export %s=%v, got %#v ok=%v", name, expected, value, ok)
	}
}

func assertModuleStringExport(t *testing.T, bindings *jayessruntime.ModuleBindings, name string, expected string) {
	t.Helper()
	binding, ok := bindings.Export(name)
	if !ok {
		t.Fatalf("expected export %s", name)
	}
	value, ok := binding.Value()
	if !ok || value.Text() != expected {
		t.Fatalf("expected export %s=%q, got %#v ok=%v", name, expected, value, ok)
	}
}
