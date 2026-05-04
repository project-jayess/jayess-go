package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFindsWidestInitializationBatchesForEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)

	indexes, width, err := graph.WidestInitializationBatches("main.js")
	if err != nil {
		t.Fatalf("WidestInitializationBatches returned error: %v", err)
	}
	expectedIndexes := []int{0}
	if !reflect.DeepEqual(indexes, expectedIndexes) {
		t.Fatalf("expected widest batch indexes %#v, got %#v", expectedIndexes, indexes)
	}
	if width != 2 {
		t.Fatalf("expected widest batch width 2, got %d", width)
	}
}

func TestResolverModuleGraphWidestInitializationBatchesForUnknownEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()

	indexes, width, err := graph.WidestInitializationBatches("main.js")
	if err != nil {
		t.Fatalf("WidestInitializationBatches returned error: %v", err)
	}
	expectedIndexes := []int{0}
	if !reflect.DeepEqual(indexes, expectedIndexes) {
		t.Fatalf("expected widest batch indexes %#v, got %#v", expectedIndexes, indexes)
	}
	if width != 1 {
		t.Fatalf("expected widest batch width 1, got %d", width)
	}
}

func TestResolverModuleGraphWidestInitializationBatchesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, _, err := graph.WidestInitializationBatches("main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
