package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphCountsModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js", "config.js"})

	if count := graph.ModuleCount(); count != 3 {
		t.Fatalf("expected 3 modules, got %d", count)
	}
}

func TestResolverModuleGraphCountsImportEdges(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js", "config.js"})
	graph.AddModule("worker.js", []string{"shared.js"})

	if count := graph.ImportEdgeCount(); count != 3 {
		t.Fatalf("expected 3 import edges, got %d", count)
	}
}

func TestResolverModuleGraphCountsDirectDependencies(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js", "config.js"})

	if count := graph.DependencyCount("main.js"); count != 2 {
		t.Fatalf("expected 2 dependencies, got %d", count)
	}
	if count := graph.DependencyCount("missing.js"); count != 0 {
		t.Fatalf("expected no dependencies for missing module, got %d", count)
	}
}

func TestResolverModuleGraphCountsDirectDependents(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("shared.js", nil)

	if count := graph.DependentCount("shared.js"); count != 2 {
		t.Fatalf("expected 2 dependents, got %d", count)
	}
	if count := graph.DependentCount("missing.js"); count != 0 {
		t.Fatalf("expected no dependents for missing module, got %d", count)
	}
}

func TestResolverModuleGraphExportsDependencyCountMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})

	expected := map[string]int{
		"app.js":    1,
		"config.js": 0,
		"main.js":   2,
		"model.js":  0,
		"worker.js": 1,
	}
	if counts := graph.DependencyCountMap(); !reflect.DeepEqual(counts, expected) {
		t.Fatalf("expected dependency count map %#v, got %#v", expected, counts)
	}
}

func TestResolverModuleGraphExportsDependentCountMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})

	expected := map[string]int{
		"app.js":    1,
		"config.js": 1,
		"main.js":   0,
		"model.js":  2,
		"worker.js": 0,
	}
	if counts := graph.DependentCountMap(); !reflect.DeepEqual(counts, expected) {
		t.Fatalf("expected dependent count map %#v, got %#v", expected, counts)
	}
}
