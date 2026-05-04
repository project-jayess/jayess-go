package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsNarrowestInitializationBatchMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	summaries, err := graph.NarrowestInitializationBatchMap()
	if err != nil {
		t.Fatalf("NarrowestInitializationBatchMap returned error: %v", err)
	}
	expected := map[string]resolver.NarrowestInitializationBatchSummary{
		"app.js":    {Indexes: []int{0, 1}, Width: 1},
		"config.js": {Indexes: []int{0}, Width: 1},
		"main.js":   {Indexes: []int{1, 2}, Width: 1},
		"model.js":  {Indexes: []int{0}, Width: 1},
		"worker.js": {Indexes: []int{0, 1, 2}, Width: 1},
	}
	if !reflect.DeepEqual(summaries, expected) {
		t.Fatalf("expected narrowest initialization batch map %#v, got %#v", expected, summaries)
	}
}

func TestResolverModuleGraphNarrowestInitializationBatchMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.NarrowestInitializationBatchMap()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphNarrowestInitializationBatchMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	summaries, err := graph.NarrowestInitializationBatchMap()
	if err != nil {
		t.Fatalf("NarrowestInitializationBatchMap returned error: %v", err)
	}
	if summaries != nil {
		t.Fatalf("expected nil narrowest initialization batch map for empty graph, got %#v", summaries)
	}
}
