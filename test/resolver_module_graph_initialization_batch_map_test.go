package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsInitializationBatchMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	batches, err := graph.InitializationBatchMap()
	if err != nil {
		t.Fatalf("InitializationBatchMap returned error: %v", err)
	}
	expected := map[string][][]string{
		"app.js": {
			{"model.js"},
			{"app.js"},
		},
		"config.js": {
			{"config.js"},
		},
		"main.js": {
			{"config.js", "model.js"},
			{"app.js"},
			{"main.js"},
		},
		"model.js": {
			{"model.js"},
		},
		"worker.js": {
			{"model.js"},
			{"app.js"},
			{"worker.js"},
		},
	}
	if !reflect.DeepEqual(batches, expected) {
		t.Fatalf("expected initialization batch map %#v, got %#v", expected, batches)
	}
}

func TestResolverModuleGraphInitializationBatchMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchMap()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphInitializationBatchMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	batches, err := graph.InitializationBatchMap()
	if err != nil {
		t.Fatalf("InitializationBatchMap returned error: %v", err)
	}
	if batches != nil {
		t.Fatalf("expected nil initialization batch map for empty graph, got %#v", batches)
	}
}
