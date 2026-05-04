package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphCountsInitializationBatchesForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)
	graph.AddModule("unused.js", nil)

	count, err := graph.InitializationBatchCountFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("InitializationBatchCountFor returned error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 initialization batches, got %d", count)
	}
}

func TestResolverModuleGraphInitializationBatchCountForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	count, err := graph.InitializationBatchCountFor(nil)
	if err != nil {
		t.Fatalf("InitializationBatchCountFor returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no-entry batch count 0, got %d", count)
	}
}

func TestResolverModuleGraphInitializationBatchCountForEntriesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchCountFor([]string{"main.js"})
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
