package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsInitializationBatchesForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)
	graph.AddModule("unused.js", nil)

	batches, err := graph.InitializationBatchesFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("InitializationBatchesFor returned error: %v", err)
	}

	expected := [][]string{
		{"config.js", "model.js", "worker-model.js"},
		{"app.js", "worker.js"},
		{"main.js"},
	}
	if !reflect.DeepEqual(batches, expected) {
		t.Fatalf("expected initialization batches %#v, got %#v", expected, batches)
	}
}

func TestResolverModuleGraphInitializationBatchesForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	batches, err := graph.InitializationBatchesFor(nil)
	if err != nil {
		t.Fatalf("InitializationBatchesFor returned error: %v", err)
	}
	if batches != nil {
		t.Fatalf("expected nil batches for no entries, got %#v", batches)
	}
}

func TestResolverModuleGraphInitializationBatchesForEntriesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchesFor([]string{"main.js"})
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
