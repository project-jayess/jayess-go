package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFiltersModulesBetweenDependencyDepth(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	modules, err := graph.ModulesBetweenDependencyDepth(1, 2)
	if err != nil {
		t.Fatalf("ModulesBetweenDependencyDepth returned error: %v", err)
	}
	expected := []string{"app.js", "main.js", "worker.js"}
	if !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected modules between depths 1 and 2 %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphModulesBetweenInvalidDependencyDepthRangeAreEmpty(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	modules, err := graph.ModulesBetweenDependencyDepth(2, 1)
	if err != nil {
		t.Fatalf("ModulesBetweenDependencyDepth returned error: %v", err)
	}
	if modules != nil {
		t.Fatalf("expected nil modules for invalid range, got %#v", modules)
	}
}

func TestResolverModuleGraphModulesBetweenDependencyDepthReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ModulesBetweenDependencyDepth(0, 2)
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
