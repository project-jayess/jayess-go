package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsInitializationBatchWidthMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	widths, err := graph.InitializationBatchWidthMap()
	if err != nil {
		t.Fatalf("InitializationBatchWidthMap returned error: %v", err)
	}
	expected := map[string][]int{
		"app.js":    {1, 1},
		"config.js": {1},
		"main.js":   {2, 1, 1},
		"model.js":  {1},
		"worker.js": {1, 1, 1},
	}
	if !reflect.DeepEqual(widths, expected) {
		t.Fatalf("expected initialization batch width map %#v, got %#v", expected, widths)
	}
}

func TestResolverModuleGraphInitializationBatchWidthMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchWidthMap()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphInitializationBatchWidthMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	widths, err := graph.InitializationBatchWidthMap()
	if err != nil {
		t.Fatalf("InitializationBatchWidthMap returned error: %v", err)
	}
	if widths != nil {
		t.Fatalf("expected nil initialization batch width map for empty graph, got %#v", widths)
	}
}
