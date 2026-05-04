package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphBuildsReachableSubgraphForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)
	graph.AddModule("unused.js", nil)

	subgraph, err := graph.ReachableSubgraphFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("ReachableSubgraphFor returned error: %v", err)
	}

	modules := subgraph.Modules()
	expectedModules := []string{"app.js", "config.js", "main.js", "model.js", "worker-model.js", "worker.js"}
	if !reflect.DeepEqual(modules, expectedModules) {
		t.Fatalf("expected reachable modules %#v, got %#v", expectedModules, modules)
	}

	batches, err := subgraph.InitializationBatchesAll()
	if err != nil {
		t.Fatalf("subgraph InitializationBatchesAll returned error: %v", err)
	}
	expectedBatches := [][]string{
		{"config.js", "model.js", "worker-model.js"},
		{"app.js", "worker.js"},
		{"main.js"},
	}
	if !reflect.DeepEqual(batches, expectedBatches) {
		t.Fatalf("expected reachable batches %#v, got %#v", expectedBatches, batches)
	}
}

func TestResolverModuleGraphReachableSubgraphForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	subgraph, err := graph.ReachableSubgraphFor(nil)
	if err != nil {
		t.Fatalf("ReachableSubgraphFor returned error: %v", err)
	}
	if count := subgraph.ModuleCount(); count != 0 {
		t.Fatalf("expected empty subgraph, got %d modules", count)
	}
}

func TestResolverModuleGraphReachableSubgraphForEntriesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableSubgraphFor([]string{"main.js"})
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
