package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsWidestInitializationBatchMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	summaries, err := graph.WidestInitializationBatchMap()
	if err != nil {
		t.Fatalf("WidestInitializationBatchMap returned error: %v", err)
	}
	expected := map[string]resolver.WidestInitializationBatchSummary{
		"app.js":    {Indexes: []int{0, 1}, Width: 1},
		"config.js": {Indexes: []int{0}, Width: 1},
		"main.js":   {Indexes: []int{0}, Width: 2},
		"model.js":  {Indexes: []int{0}, Width: 1},
		"worker.js": {Indexes: []int{0, 1, 2}, Width: 1},
	}
	if !reflect.DeepEqual(summaries, expected) {
		t.Fatalf("expected widest initialization batch map %#v, got %#v", expected, summaries)
	}
}

func TestResolverModuleGraphWidestInitializationBatchMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.WidestInitializationBatchMap()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphWidestInitializationBatchMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	summaries, err := graph.WidestInitializationBatchMap()
	if err != nil {
		t.Fatalf("WidestInitializationBatchMap returned error: %v", err)
	}
	if summaries != nil {
		t.Fatalf("expected nil widest initialization batch map for empty graph, got %#v", summaries)
	}
}
