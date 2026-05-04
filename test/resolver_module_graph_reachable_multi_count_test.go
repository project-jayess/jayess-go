package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphCountsReachableModulesForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)
	graph.AddModule("unused.js", nil)

	count, err := graph.ReachableModuleCountFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("ReachableModuleCountFor returned error: %v", err)
	}
	if count != 6 {
		t.Fatalf("expected 6 reachable modules, got %d", count)
	}
}

func TestResolverModuleGraphReachableModuleCountForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	count, err := graph.ReachableModuleCountFor(nil)
	if err != nil {
		t.Fatalf("ReachableModuleCountFor returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no-entry reachable count 0, got %d", count)
	}
}

func TestResolverModuleGraphReachableModuleCountForEntriesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableModuleCountFor([]string{"main.js"})
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
