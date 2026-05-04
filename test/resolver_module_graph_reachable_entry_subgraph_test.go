package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphBuildsReachableSubgraphForEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)

	subgraph, err := graph.ReachableSubgraph("main.js")
	if err != nil {
		t.Fatalf("ReachableSubgraph returned error: %v", err)
	}

	modules := subgraph.Modules()
	expectedModules := []string{"app.js", "config.js", "main.js", "model.js"}
	if !reflect.DeepEqual(modules, expectedModules) {
		t.Fatalf("expected reachable modules %#v, got %#v", expectedModules, modules)
	}

	order, err := subgraph.InitializationOrderAll()
	if err != nil {
		t.Fatalf("subgraph InitializationOrderAll returned error: %v", err)
	}
	expectedOrder := []string{"model.js", "app.js", "config.js", "main.js"}
	if !reflect.DeepEqual(order, expectedOrder) {
		t.Fatalf("expected reachable initialization order %#v, got %#v", expectedOrder, order)
	}
}

func TestResolverModuleGraphReachableSubgraphForUnknownEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()

	subgraph, err := graph.ReachableSubgraph("main.js")
	if err != nil {
		t.Fatalf("ReachableSubgraph returned error: %v", err)
	}
	modules := subgraph.Modules()
	expected := []string{"main.js"}
	if !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected unknown entry subgraph modules %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphReachableSubgraphReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableSubgraph("main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
