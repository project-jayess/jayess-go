package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFindsInitializationBatchWidthRangeForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)
	graph.AddModule("unused.js", nil)

	widthRange, err := graph.InitializationBatchWidthRangeFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("InitializationBatchWidthRangeFor returned error: %v", err)
	}
	expected := resolver.InitializationBatchWidthRange{Min: 1, Max: 3}
	if widthRange != expected {
		t.Fatalf("expected initialization batch width range %#v, got %#v", expected, widthRange)
	}
}

func TestResolverModuleGraphInitializationBatchWidthRangeForNoEntriesIsZero(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	widthRange, err := graph.InitializationBatchWidthRangeFor(nil)
	if err != nil {
		t.Fatalf("InitializationBatchWidthRangeFor returned error: %v", err)
	}
	if widthRange != (resolver.InitializationBatchWidthRange{}) {
		t.Fatalf("expected zero width range for no entries, got %#v", widthRange)
	}
}

func TestResolverModuleGraphInitializationBatchWidthRangeForEntriesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchWidthRangeFor([]string{"main.js"})
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
