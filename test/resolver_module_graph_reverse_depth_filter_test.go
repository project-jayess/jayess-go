package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFiltersModulesAtDependentDepth(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	modules, err := graph.ModulesAtDependentDepth(2)
	if err != nil {
		t.Fatalf("ModulesAtDependentDepth returned error: %v", err)
	}
	expected := []string{"model.js"}
	if !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected modules at dependent depth 2 %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphFiltersRootModulesAtDependentDepthZero(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("worker.js", nil)
	graph.AddModule("app.js", nil)

	modules, err := graph.ModulesAtDependentDepth(0)
	if err != nil {
		t.Fatalf("ModulesAtDependentDepth returned error: %v", err)
	}
	expected := []string{"main.js", "worker.js"}
	if !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected modules at dependent depth 0 %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphModulesAtNegativeDependentDepthAreEmpty(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	modules, err := graph.ModulesAtDependentDepth(-1)
	if err != nil {
		t.Fatalf("ModulesAtDependentDepth returned error: %v", err)
	}
	if modules != nil {
		t.Fatalf("expected nil modules for negative depth, got %#v", modules)
	}
}

func TestResolverModuleGraphModulesAtDependentDepthReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ModulesAtDependentDepth(1)
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
