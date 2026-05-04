package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphCountsInitializationBatchesAll(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	count, err := graph.InitializationBatchCountAll()
	if err != nil {
		t.Fatalf("InitializationBatchCountAll returned error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 initialization batches, got %d", count)
	}
}

func TestResolverModuleGraphInitializationBatchCountAllForEmptyGraph(t *testing.T) {
	graph := resolver.NewModuleGraph()

	count, err := graph.InitializationBatchCountAll()
	if err != nil {
		t.Fatalf("InitializationBatchCountAll returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected empty graph batch count 0, got %d", count)
	}
}

func TestResolverModuleGraphInitializationBatchCountAllReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchCountAll()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
