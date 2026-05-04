package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFindsInitializationBatchWidthRangeForEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)

	widthRange, err := graph.InitializationBatchWidthRange("main.js")
	if err != nil {
		t.Fatalf("InitializationBatchWidthRange returned error: %v", err)
	}
	expected := resolver.InitializationBatchWidthRange{Min: 1, Max: 2}
	if widthRange != expected {
		t.Fatalf("expected initialization batch width range %#v, got %#v", expected, widthRange)
	}
}

func TestResolverModuleGraphInitializationBatchWidthRangeForUnknownEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()

	widthRange, err := graph.InitializationBatchWidthRange("main.js")
	if err != nil {
		t.Fatalf("InitializationBatchWidthRange returned error: %v", err)
	}
	expected := resolver.InitializationBatchWidthRange{Min: 1, Max: 1}
	if widthRange != expected {
		t.Fatalf("expected unknown entry width range %#v, got %#v", expected, widthRange)
	}
}

func TestResolverModuleGraphInitializationBatchWidthRangeReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchWidthRange("main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
