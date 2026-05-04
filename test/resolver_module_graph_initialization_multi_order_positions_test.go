package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsInitializationOrderPositionsFor(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js", "worker_dep.js"})
	graph.AddModule("shared.js", nil)
	graph.AddModule("worker_dep.js", nil)

	positions, err := graph.InitializationOrderPositionsFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("InitializationOrderPositionsFor returned error: %v", err)
	}
	expected := map[string]int{
		"shared.js":     0,
		"main.js":       1,
		"worker_dep.js": 2,
		"worker.js":     3,
	}
	if !reflect.DeepEqual(positions, expected) {
		t.Fatalf("expected initialization order positions %#v, got %#v", expected, positions)
	}
}

func TestResolverModuleGraphInitializationOrderPositionsForReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationOrderPositionsFor([]string{"main.js", "a.js"})
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphMultiInitializationOrderPositionsForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	positions, err := graph.InitializationOrderPositionsFor([]string{"main.js"})
	if err != nil {
		t.Fatalf("InitializationOrderPositionsFor returned error: %v", err)
	}
	if positions != nil {
		t.Fatalf("expected nil initialization order positions for empty graph, got %#v", positions)
	}
}
