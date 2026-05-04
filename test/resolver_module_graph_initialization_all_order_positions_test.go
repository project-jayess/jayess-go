package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsInitializationOrderPositionsAll(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("shared.js", nil)

	positions, err := graph.InitializationOrderPositionsAll()
	if err != nil {
		t.Fatalf("InitializationOrderPositionsAll returned error: %v", err)
	}
	expected := map[string]int{
		"config.js": 0,
		"main.js":   1,
		"shared.js": 2,
		"worker.js": 3,
	}
	if !reflect.DeepEqual(positions, expected) {
		t.Fatalf("expected initialization order positions %#v, got %#v", expected, positions)
	}
}

func TestResolverModuleGraphInitializationOrderPositionsAllReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationOrderPositionsAll()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphInitializationOrderPositionsAllForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	positions, err := graph.InitializationOrderPositionsAll()
	if err != nil {
		t.Fatalf("InitializationOrderPositionsAll returned error: %v", err)
	}
	if positions != nil {
		t.Fatalf("expected nil initialization order positions for empty graph, got %#v", positions)
	}
}
