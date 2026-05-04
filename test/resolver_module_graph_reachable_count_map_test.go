package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsReachableModuleCountMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	counts, err := graph.ReachableModuleCountMap()
	if err != nil {
		t.Fatalf("ReachableModuleCountMap returned error: %v", err)
	}
	expected := map[string]int{
		"app.js":    2,
		"config.js": 1,
		"main.js":   4,
		"model.js":  1,
		"worker.js": 2,
	}
	if !reflect.DeepEqual(counts, expected) {
		t.Fatalf("expected reachable module count map %#v, got %#v", expected, counts)
	}
}

func TestResolverModuleGraphReachableModuleCountMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableModuleCountMap()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphReachableModuleCountMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	counts, err := graph.ReachableModuleCountMap()
	if err != nil {
		t.Fatalf("ReachableModuleCountMap returned error: %v", err)
	}
	if counts != nil {
		t.Fatalf("expected nil reachable module count map for empty graph, got %#v", counts)
	}
}
