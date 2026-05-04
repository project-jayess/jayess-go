package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphCountsInitializationBatchesForEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)

	count, err := graph.InitializationBatchCount("main.js")
	if err != nil {
		t.Fatalf("InitializationBatchCount returned error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 initialization batches, got %d", count)
	}
}

func TestResolverModuleGraphInitializationBatchCountForUnknownEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()

	count, err := graph.InitializationBatchCount("main.js")
	if err != nil {
		t.Fatalf("InitializationBatchCount returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected unknown entry batch count 1, got %d", count)
	}
}

func TestResolverModuleGraphInitializationBatchCountReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchCount("main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
