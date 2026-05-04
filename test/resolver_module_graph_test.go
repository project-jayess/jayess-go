package test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphOrdersDependenciesBeforeImporter(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js", "app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	order, err := graph.InitializationOrder("main.js")
	if err != nil {
		t.Fatalf("InitializationOrder returned error: %v", err)
	}
	expected := []string{"config.js", "model.js", "app.js", "main.js"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("expected initialization order %#v, got %#v", expected, order)
	}
}

func TestResolverModuleGraphIncludesImplicitLeafImports(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"dep.js"})

	order, err := graph.InitializationOrder("main.js")
	if err != nil {
		t.Fatalf("InitializationOrder returned error: %v", err)
	}
	expected := []string{"dep.js", "main.js"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("expected initialization order %#v, got %#v", expected, order)
	}
}

func TestResolverModuleGraphReportsImportCycle(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationOrder("main.js")
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
	expected := []string{"a.js", "b.js", "a.js"}
	if !reflect.DeepEqual(cycleErr.Cycle, expected) {
		t.Fatalf("expected cycle %#v, got %#v", expected, cycleErr.Cycle)
	}
	if !strings.Contains(err.Error(), "a.js -> b.js -> a.js") {
		t.Fatalf("expected readable cycle diagnostic, got %v", err)
	}
}
