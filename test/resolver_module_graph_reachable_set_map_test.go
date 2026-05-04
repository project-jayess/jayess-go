package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsReachableModuleSetMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	sets, err := graph.ReachableModuleSetMap()
	if err != nil {
		t.Fatalf("ReachableModuleSetMap returned error: %v", err)
	}
	expected := map[string]map[string]bool{
		"app.js":    {"app.js": true, "model.js": true},
		"config.js": {"config.js": true},
		"main.js":   {"app.js": true, "config.js": true, "main.js": true, "model.js": true},
		"model.js":  {"model.js": true},
		"worker.js": {"model.js": true, "worker.js": true},
	}
	if !reflect.DeepEqual(sets, expected) {
		t.Fatalf("expected reachable module set map %#v, got %#v", expected, sets)
	}
}

func TestResolverModuleGraphReachableModuleSetMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableModuleSetMap()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphReachableModuleSetMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	sets, err := graph.ReachableModuleSetMap()
	if err != nil {
		t.Fatalf("ReachableModuleSetMap returned error: %v", err)
	}
	if sets != nil {
		t.Fatalf("expected nil reachable module set map for empty graph, got %#v", sets)
	}
}
