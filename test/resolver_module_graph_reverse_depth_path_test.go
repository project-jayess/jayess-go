package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFindsLongestDependentPath(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", []string{"storage.js"})
	graph.AddModule("worker.js", []string{"storage.js"})
	graph.AddModule("storage.js", nil)

	path, err := graph.LongestDependentPath("storage.js")
	if err != nil {
		t.Fatalf("LongestDependentPath returned error: %v", err)
	}
	expected := []string{"storage.js", "model.js", "app.js", "main.js"}
	if !reflect.DeepEqual(path, expected) {
		t.Fatalf("expected longest dependent path %#v, got %#v", expected, path)
	}
}

func TestResolverModuleGraphLongestDependentPathForRootAndMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	path, err := graph.LongestDependentPath("main.js")
	if err != nil {
		t.Fatalf("LongestDependentPath returned error for root: %v", err)
	}
	expected := []string{"main.js"}
	if !reflect.DeepEqual(path, expected) {
		t.Fatalf("expected root path %#v, got %#v", expected, path)
	}

	path, err = graph.LongestDependentPath("missing.js")
	if err != nil {
		t.Fatalf("LongestDependentPath returned error for missing module: %v", err)
	}
	if path != nil {
		t.Fatalf("expected nil missing module path, got %#v", path)
	}
}

func TestResolverModuleGraphLongestDependentPathReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.LongestDependentPath("a.js")
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

func TestResolverModuleGraphLongestDependentPathDoesNotShareSlices(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", nil)

	path, err := graph.LongestDependentPath("app.js")
	if err != nil {
		t.Fatalf("LongestDependentPath returned error: %v", err)
	}
	path[1] = "changed.js"

	expected := []string{"app.js", "main.js"}
	if path, err := graph.LongestDependentPath("app.js"); err != nil {
		t.Fatalf("LongestDependentPath returned error after mutation: %v", err)
	} else if !reflect.DeepEqual(path, expected) {
		t.Fatalf("expected longest dependent path %#v, got %#v", expected, path)
	}
}
