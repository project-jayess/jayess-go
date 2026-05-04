package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFindsNarrowestInitializationBatchesForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)
	graph.AddModule("unused.js", nil)

	indexes, width, err := graph.NarrowestInitializationBatchesFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("NarrowestInitializationBatchesFor returned error: %v", err)
	}
	expectedIndexes := []int{2}
	if !reflect.DeepEqual(indexes, expectedIndexes) {
		t.Fatalf("expected narrowest batch indexes %#v, got %#v", expectedIndexes, indexes)
	}
	if width != 1 {
		t.Fatalf("expected narrowest batch width 1, got %d", width)
	}
}

func TestResolverModuleGraphNarrowestInitializationBatchesForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	indexes, width, err := graph.NarrowestInitializationBatchesFor(nil)
	if err != nil {
		t.Fatalf("NarrowestInitializationBatchesFor returned error: %v", err)
	}
	if indexes != nil {
		t.Fatalf("expected nil narrowest batch indexes for no entries, got %#v", indexes)
	}
	if width != 0 {
		t.Fatalf("expected narrowest batch width 0, got %d", width)
	}
}

func TestResolverModuleGraphNarrowestInitializationBatchesForReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, _, err := graph.NarrowestInitializationBatchesFor([]string{"main.js"})
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
