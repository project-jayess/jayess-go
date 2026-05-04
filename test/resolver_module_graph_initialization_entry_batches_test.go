package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsInitializationBatchesForEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)

	batches, err := graph.InitializationBatches("main.js")
	if err != nil {
		t.Fatalf("InitializationBatches returned error: %v", err)
	}

	expected := [][]string{
		{"config.js", "model.js"},
		{"app.js"},
		{"main.js"},
	}
	if !reflect.DeepEqual(batches, expected) {
		t.Fatalf("expected initialization batches %#v, got %#v", expected, batches)
	}
}

func TestResolverModuleGraphInitializationBatchesForUnknownEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()

	batches, err := graph.InitializationBatches("main.js")
	if err != nil {
		t.Fatalf("InitializationBatches returned error: %v", err)
	}
	expected := [][]string{{"main.js"}}
	if !reflect.DeepEqual(batches, expected) {
		t.Fatalf("expected unknown entry batch %#v, got %#v", expected, batches)
	}
}

func TestResolverModuleGraphInitializationBatchesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatches("main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
