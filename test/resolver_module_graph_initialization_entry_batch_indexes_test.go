package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsEntryInitializationBatchIndexes(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	indexes, err := graph.InitializationBatchIndexes("main.js")
	if err != nil {
		t.Fatalf("InitializationBatchIndexes returned error: %v", err)
	}
	expected := map[string]int{
		"config.js": 0,
		"model.js":  0,
		"app.js":    1,
		"main.js":   2,
	}
	if !reflect.DeepEqual(indexes, expected) {
		t.Fatalf("expected initialization batch indexes %#v, got %#v", expected, indexes)
	}
}

func TestResolverModuleGraphEntryInitializationBatchIndexesForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	indexes, err := graph.InitializationBatchIndexes("main.js")
	if err != nil {
		t.Fatalf("InitializationBatchIndexes returned error: %v", err)
	}
	if indexes != nil {
		t.Fatalf("expected nil batch indexes for empty graph, got %#v", indexes)
	}
}

func TestResolverModuleGraphEntryInitializationBatchIndexesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchIndexes("main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
