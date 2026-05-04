package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFindsLongestDependencyPath(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js", "app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", []string{"storage.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("storage.js", nil)

	path, err := graph.LongestDependencyPath("main.js")
	if err != nil {
		t.Fatalf("LongestDependencyPath returned error: %v", err)
	}
	expected := []string{"main.js", "app.js", "model.js", "storage.js"}
	if !reflect.DeepEqual(path, expected) {
		t.Fatalf("expected longest dependency path %#v, got %#v", expected, path)
	}
}

func TestResolverModuleGraphLongestDependencyPathForLeafAndMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	path, err := graph.LongestDependencyPath("main.js")
	if err != nil {
		t.Fatalf("LongestDependencyPath returned error for leaf: %v", err)
	}
	expected := []string{"main.js"}
	if !reflect.DeepEqual(path, expected) {
		t.Fatalf("expected leaf path %#v, got %#v", expected, path)
	}

	path, err = graph.LongestDependencyPath("missing.js")
	if err != nil {
		t.Fatalf("LongestDependencyPath returned error for missing module: %v", err)
	}
	if path != nil {
		t.Fatalf("expected nil missing module path, got %#v", path)
	}
}

func TestResolverModuleGraphLongestDependencyPathReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.LongestDependencyPath("main.js")
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
	expected := []string{"a.js", "b.js", "a.js"}
	if !reflect.DeepEqual(cycleErr.Cycle, expected) {
		t.Fatalf("expected cycle %#v, got %#v", expected, cycleErr.Cycle)
	}
}

func TestResolverModuleGraphLongestDependencyPathDoesNotShareSlices(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", nil)

	path, err := graph.LongestDependencyPath("main.js")
	if err != nil {
		t.Fatalf("LongestDependencyPath returned error: %v", err)
	}
	path[1] = "changed.js"

	expected := []string{"main.js", "app.js"}
	if path, err := graph.LongestDependencyPath("main.js"); err != nil {
		t.Fatalf("LongestDependencyPath returned error after mutation: %v", err)
	} else if !reflect.DeepEqual(path, expected) {
		t.Fatalf("expected longest dependency path %#v, got %#v", expected, path)
	}
}
