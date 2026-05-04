package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsInitializationBatchCountMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	counts, err := graph.InitializationBatchCountMap()
	if err != nil {
		t.Fatalf("InitializationBatchCountMap returned error: %v", err)
	}
	expected := map[string]int{
		"app.js":    2,
		"config.js": 1,
		"main.js":   3,
		"model.js":  1,
		"worker.js": 3,
	}
	if !reflect.DeepEqual(counts, expected) {
		t.Fatalf("expected initialization batch count map %#v, got %#v", expected, counts)
	}
}

func TestResolverModuleGraphInitializationBatchCountMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchCountMap()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphInitializationBatchCountMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	counts, err := graph.InitializationBatchCountMap()
	if err != nil {
		t.Fatalf("InitializationBatchCountMap returned error: %v", err)
	}
	if counts != nil {
		t.Fatalf("expected nil initialization batch count map for empty graph, got %#v", counts)
	}
}
