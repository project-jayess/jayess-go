package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphCountsReachableModulesForEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)

	count, err := graph.ReachableModuleCount("main.js")
	if err != nil {
		t.Fatalf("ReachableModuleCount returned error: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected 4 reachable modules, got %d", count)
	}
}

func TestResolverModuleGraphReachableModuleCountForUnknownEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()

	count, err := graph.ReachableModuleCount("main.js")
	if err != nil {
		t.Fatalf("ReachableModuleCount returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected unknown entry reachable count 1, got %d", count)
	}
}

func TestResolverModuleGraphReachableModuleCountReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableModuleCount("main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
