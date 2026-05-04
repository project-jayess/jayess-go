package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsReachableModuleMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	reachable, err := graph.ReachableModuleMap()
	if err != nil {
		t.Fatalf("ReachableModuleMap returned error: %v", err)
	}
	expected := map[string][]string{
		"app.js":    {"app.js", "model.js"},
		"config.js": {"config.js"},
		"main.js":   {"app.js", "config.js", "main.js", "model.js"},
		"model.js":  {"model.js"},
		"worker.js": {"model.js", "worker.js"},
	}
	if !reflect.DeepEqual(reachable, expected) {
		t.Fatalf("expected reachable module map %#v, got %#v", expected, reachable)
	}
}

func TestResolverModuleGraphReachableModuleMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableModuleMap()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphReachableModuleMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	reachable, err := graph.ReachableModuleMap()
	if err != nil {
		t.Fatalf("ReachableModuleMap returned error: %v", err)
	}
	if reachable != nil {
		t.Fatalf("expected nil reachable module map for empty graph, got %#v", reachable)
	}
}
