package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsInitializationBatchesAll(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	batches, err := graph.InitializationBatchesAll()
	if err != nil {
		t.Fatalf("InitializationBatchesAll returned error: %v", err)
	}

	expected := [][]string{
		{"config.js", "model.js"},
		{"app.js"},
		{"main.js", "worker.js"},
	}
	if !reflect.DeepEqual(batches, expected) {
		t.Fatalf("expected initialization batches %#v, got %#v", expected, batches)
	}
}

func TestResolverModuleGraphInitializationBatchesAllForEmptyGraph(t *testing.T) {
	graph := resolver.NewModuleGraph()

	batches, err := graph.InitializationBatchesAll()
	if err != nil {
		t.Fatalf("InitializationBatchesAll returned error: %v", err)
	}
	if batches != nil {
		t.Fatalf("expected nil batches for empty graph, got %#v", batches)
	}
}

func TestResolverModuleGraphInitializationBatchesAllReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchesAll()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
