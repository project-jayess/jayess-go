package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsInitializationOrderPositions(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	positions, err := graph.InitializationOrderPositions("main.js")
	if err != nil {
		t.Fatalf("InitializationOrderPositions returned error: %v", err)
	}
	expected := map[string]int{
		"model.js":  0,
		"app.js":    1,
		"config.js": 2,
		"main.js":   3,
	}
	if !reflect.DeepEqual(positions, expected) {
		t.Fatalf("expected initialization order positions %#v, got %#v", expected, positions)
	}
}

func TestResolverModuleGraphInitializationOrderPositionsReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationOrderPositions("main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphInitializationOrderPositionsForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	positions, err := graph.InitializationOrderPositions("main.js")
	if err != nil {
		t.Fatalf("InitializationOrderPositions returned error: %v", err)
	}
	if positions != nil {
		t.Fatalf("expected nil initialization order positions for empty graph, got %#v", positions)
	}
}
