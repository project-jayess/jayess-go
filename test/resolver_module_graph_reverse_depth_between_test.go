package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFiltersModulesBetweenDependentDepth(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	modules, err := graph.ModulesBetweenDependentDepth(1, 2)
	if err != nil {
		t.Fatalf("ModulesBetweenDependentDepth returned error: %v", err)
	}
	expected := []string{"app.js", "model.js"}
	if !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected modules between dependent depths 1 and 2 %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphModulesBetweenInvalidDependentDepthRangeAreEmpty(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	modules, err := graph.ModulesBetweenDependentDepth(2, 1)
	if err != nil {
		t.Fatalf("ModulesBetweenDependentDepth returned error: %v", err)
	}
	if modules != nil {
		t.Fatalf("expected nil modules for invalid range, got %#v", modules)
	}
}

func TestResolverModuleGraphModulesBetweenDependentDepthReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ModulesBetweenDependentDepth(0, 2)
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
