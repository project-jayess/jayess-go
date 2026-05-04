package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFiltersModulesAtDependencyDepth(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	modules, err := graph.ModulesAtDependencyDepth(2)
	if err != nil {
		t.Fatalf("ModulesAtDependencyDepth returned error: %v", err)
	}
	expected := []string{"main.js", "worker.js"}
	if !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected modules at depth 2 %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphFiltersLeafModulesAtDependencyDepthZero(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", nil)
	graph.AddModule("worker.js", nil)

	modules, err := graph.ModulesAtDependencyDepth(0)
	if err != nil {
		t.Fatalf("ModulesAtDependencyDepth returned error: %v", err)
	}
	expected := []string{"app.js", "config.js", "worker.js"}
	if !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected modules at depth 0 %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphModulesAtNegativeDependencyDepthAreEmpty(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	modules, err := graph.ModulesAtDependencyDepth(-1)
	if err != nil {
		t.Fatalf("ModulesAtDependencyDepth returned error: %v", err)
	}
	if modules != nil {
		t.Fatalf("expected nil modules for negative depth, got %#v", modules)
	}
}

func TestResolverModuleGraphModulesAtDependencyDepthReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ModulesAtDependencyDepth(1)
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
