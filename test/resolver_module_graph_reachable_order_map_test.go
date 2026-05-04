package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsReachableModuleOrderMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	orders, err := graph.ReachableModuleOrderMap()
	if err != nil {
		t.Fatalf("ReachableModuleOrderMap returned error: %v", err)
	}
	expected := map[string][]string{
		"app.js":    {"model.js", "app.js"},
		"config.js": {"config.js"},
		"main.js":   {"model.js", "app.js", "config.js", "main.js"},
		"model.js":  {"model.js"},
		"worker.js": {"model.js", "worker.js"},
	}
	if !reflect.DeepEqual(orders, expected) {
		t.Fatalf("expected reachable module order map %#v, got %#v", expected, orders)
	}
}

func TestResolverModuleGraphReachableModuleOrderMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableModuleOrderMap()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphReachableModuleOrderMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	orders, err := graph.ReachableModuleOrderMap()
	if err != nil {
		t.Fatalf("ReachableModuleOrderMap returned error: %v", err)
	}
	if orders != nil {
		t.Fatalf("expected nil reachable module order map for empty graph, got %#v", orders)
	}
}
