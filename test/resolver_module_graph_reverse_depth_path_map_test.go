package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsLongestDependentPathMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	paths, err := graph.LongestDependentPathMap()
	if err != nil {
		t.Fatalf("LongestDependentPathMap returned error: %v", err)
	}
	expected := map[string][]string{
		"app.js":    []string{"app.js", "main.js"},
		"config.js": []string{"config.js", "main.js"},
		"main.js":   []string{"main.js"},
		"model.js":  []string{"model.js", "app.js", "main.js"},
		"worker.js": []string{"worker.js"},
	}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("expected longest dependent path map %#v, got %#v", expected, paths)
	}
}

func TestResolverModuleGraphLongestDependentPathMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.LongestDependentPathMap()
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphLongestDependentPathMapDoesNotShareSlices(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", nil)

	paths, err := graph.LongestDependentPathMap()
	if err != nil {
		t.Fatalf("LongestDependentPathMap returned error: %v", err)
	}
	paths["app.js"][1] = "changed.js"

	expected := []string{"app.js", "main.js"}
	if path, err := graph.LongestDependentPath("app.js"); err != nil {
		t.Fatalf("LongestDependentPath returned error after mutation: %v", err)
	} else if !reflect.DeepEqual(path, expected) {
		t.Fatalf("expected longest dependent path %#v, got %#v", expected, path)
	}
}
