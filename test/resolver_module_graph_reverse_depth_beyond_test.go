package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFiltersModulesBeyondDependentDepth(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	modules, err := graph.ModulesBeyondDependentDepth(1)
	if err != nil {
		t.Fatalf("ModulesBeyondDependentDepth returned error: %v", err)
	}
	expected := []string{"model.js"}
	if !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected modules beyond dependent depth 1 %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphModulesBeyondNegativeDependentDepthIncludesAllModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", nil)

	modules, err := graph.ModulesBeyondDependentDepth(-1)
	if err != nil {
		t.Fatalf("ModulesBeyondDependentDepth returned error: %v", err)
	}
	expected := []string{"app.js", "main.js", "worker.js"}
	if !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected modules beyond negative dependent depth %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphModulesBeyondDependentDepthReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ModulesBeyondDependentDepth(1)
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
