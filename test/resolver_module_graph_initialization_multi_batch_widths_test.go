package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsInitializationBatchWidthsForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)
	graph.AddModule("unused.js", nil)

	widths, err := graph.InitializationBatchWidthsFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("InitializationBatchWidthsFor returned error: %v", err)
	}

	expected := []int{3, 2, 1}
	if !reflect.DeepEqual(widths, expected) {
		t.Fatalf("expected initialization batch widths %#v, got %#v", expected, widths)
	}
}

func TestResolverModuleGraphInitializationBatchWidthsForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	widths, err := graph.InitializationBatchWidthsFor(nil)
	if err != nil {
		t.Fatalf("InitializationBatchWidthsFor returned error: %v", err)
	}
	if widths != nil {
		t.Fatalf("expected nil widths for no entries, got %#v", widths)
	}
}

func TestResolverModuleGraphInitializationBatchWidthsForEntriesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchWidthsFor([]string{"main.js"})
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
