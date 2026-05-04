package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFindsNarrowestInitializationBatchesAll(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	indexes, width, err := graph.NarrowestInitializationBatchesAll()
	if err != nil {
		t.Fatalf("NarrowestInitializationBatchesAll returned error: %v", err)
	}
	expectedIndexes := []int{1}
	if !reflect.DeepEqual(indexes, expectedIndexes) {
		t.Fatalf("expected narrowest batch indexes %#v, got %#v", expectedIndexes, indexes)
	}
	if width != 1 {
		t.Fatalf("expected narrowest batch width 1, got %d", width)
	}
}

func TestResolverModuleGraphNarrowestInitializationBatchesAllForEmptyGraph(t *testing.T) {
	graph := resolver.NewModuleGraph()

	indexes, width, err := graph.NarrowestInitializationBatchesAll()
	if err != nil {
		t.Fatalf("NarrowestInitializationBatchesAll returned error: %v", err)
	}
	if indexes != nil {
		t.Fatalf("expected nil narrowest batch indexes for empty graph, got %#v", indexes)
	}
	if width != 0 {
		t.Fatalf("expected narrowest batch width 0, got %d", width)
	}
}

func TestResolverModuleGraphNarrowestInitializationBatchesAllReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, _, err := graph.NarrowestInitializationBatchesAll()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
