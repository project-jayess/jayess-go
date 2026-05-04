package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsFullInitializationBatchIndexes(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	indexes, err := graph.InitializationBatchIndexesAll()
	if err != nil {
		t.Fatalf("InitializationBatchIndexesAll returned error: %v", err)
	}
	expected := map[string]int{
		"config.js": 0,
		"model.js":  0,
		"app.js":    1,
		"main.js":   2,
		"worker.js": 2,
	}
	if !reflect.DeepEqual(indexes, expected) {
		t.Fatalf("expected initialization batch indexes %#v, got %#v", expected, indexes)
	}
}

func TestResolverModuleGraphFullInitializationBatchIndexesForEmptyGraphIsNil(t *testing.T) {
	graph := resolver.NewModuleGraph()

	indexes, err := graph.InitializationBatchIndexesAll()
	if err != nil {
		t.Fatalf("InitializationBatchIndexesAll returned error: %v", err)
	}
	if indexes != nil {
		t.Fatalf("expected nil batch indexes for empty graph, got %#v", indexes)
	}
}

func TestResolverModuleGraphFullInitializationBatchIndexesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchIndexesAll()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
