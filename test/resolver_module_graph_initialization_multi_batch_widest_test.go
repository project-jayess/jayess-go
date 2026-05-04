package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFindsWidestInitializationBatchesForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)
	graph.AddModule("unused.js", nil)

	indexes, width, err := graph.WidestInitializationBatchesFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("WidestInitializationBatchesFor returned error: %v", err)
	}
	expectedIndexes := []int{0}
	if !reflect.DeepEqual(indexes, expectedIndexes) {
		t.Fatalf("expected widest batch indexes %#v, got %#v", expectedIndexes, indexes)
	}
	if width != 3 {
		t.Fatalf("expected widest batch width 3, got %d", width)
	}
}

func TestResolverModuleGraphWidestInitializationBatchesForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	indexes, width, err := graph.WidestInitializationBatchesFor(nil)
	if err != nil {
		t.Fatalf("WidestInitializationBatchesFor returned error: %v", err)
	}
	if indexes != nil {
		t.Fatalf("expected nil widest batch indexes for no entries, got %#v", indexes)
	}
	if width != 0 {
		t.Fatalf("expected widest batch width 0, got %d", width)
	}
}

func TestResolverModuleGraphWidestInitializationBatchesForReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, _, err := graph.WidestInitializationBatchesFor([]string{"main.js"})
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
