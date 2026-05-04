package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsInitializationOrderPositionMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	positions, err := graph.InitializationOrderPositionMap()
	if err != nil {
		t.Fatalf("InitializationOrderPositionMap returned error: %v", err)
	}
	expected := map[string]map[string]int{
		"app.js": {
			"model.js": 0,
			"app.js":   1,
		},
		"config.js": {
			"config.js": 0,
		},
		"main.js": {
			"model.js":  0,
			"app.js":    1,
			"config.js": 2,
			"main.js":   3,
		},
		"model.js": {
			"model.js": 0,
		},
		"worker.js": {
			"model.js":  0,
			"worker.js": 1,
		},
	}
	if !reflect.DeepEqual(positions, expected) {
		t.Fatalf("expected initialization order position map %#v, got %#v", expected, positions)
	}
}

func TestResolverModuleGraphInitializationOrderPositionMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationOrderPositionMap()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphInitializationOrderPositionMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	positions, err := graph.InitializationOrderPositionMap()
	if err != nil {
		t.Fatalf("InitializationOrderPositionMap returned error: %v", err)
	}
	if positions != nil {
		t.Fatalf("expected nil initialization order position map for empty graph, got %#v", positions)
	}
}
