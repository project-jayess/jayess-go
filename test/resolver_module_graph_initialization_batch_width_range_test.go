package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFindsInitializationBatchWidthRangeAll(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	widthRange, err := graph.InitializationBatchWidthRangeAll()
	if err != nil {
		t.Fatalf("InitializationBatchWidthRangeAll returned error: %v", err)
	}
	expected := resolver.InitializationBatchWidthRange{Min: 1, Max: 2}
	if widthRange != expected {
		t.Fatalf("expected initialization batch width range %#v, got %#v", expected, widthRange)
	}
}

func TestResolverModuleGraphInitializationBatchWidthRangeAllForEmptyGraphIsZero(t *testing.T) {
	graph := resolver.NewModuleGraph()

	widthRange, err := graph.InitializationBatchWidthRangeAll()
	if err != nil {
		t.Fatalf("InitializationBatchWidthRangeAll returned error: %v", err)
	}
	if widthRange != (resolver.InitializationBatchWidthRange{}) {
		t.Fatalf("expected zero width range for empty graph, got %#v", widthRange)
	}
}

func TestResolverModuleGraphInitializationBatchWidthRangeAllReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchWidthRangeAll()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
