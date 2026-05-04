package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsLongestDependencyPathMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	paths, err := graph.LongestDependencyPathMap()
	if err != nil {
		t.Fatalf("LongestDependencyPathMap returned error: %v", err)
	}
	expected := map[string][]string{
		"app.js":    []string{"app.js", "model.js"},
		"config.js": []string{"config.js"},
		"main.js":   []string{"main.js", "app.js", "model.js"},
		"model.js":  []string{"model.js"},
		"worker.js": []string{"worker.js", "model.js"},
	}
	if !reflect.DeepEqual(paths, expected) {
		t.Fatalf("expected longest dependency path map %#v, got %#v", expected, paths)
	}
}

func TestResolverModuleGraphLongestDependencyPathMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.LongestDependencyPathMap()
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphLongestDependencyPathMapDoesNotShareSlices(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", nil)

	paths, err := graph.LongestDependencyPathMap()
	if err != nil {
		t.Fatalf("LongestDependencyPathMap returned error: %v", err)
	}
	paths["main.js"][1] = "changed.js"

	expected := []string{"main.js", "app.js"}
	if path, err := graph.LongestDependencyPath("main.js"); err != nil {
		t.Fatalf("LongestDependencyPath returned error after mutation: %v", err)
	} else if !reflect.DeepEqual(path, expected) {
		t.Fatalf("expected longest dependency path %#v, got %#v", expected, path)
	}
}
