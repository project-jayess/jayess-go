package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphReachableModuleSetForEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)

	set, err := graph.ReachableModuleSet("main.js")
	if err != nil {
		t.Fatalf("ReachableModuleSet returned error: %v", err)
	}
	expected := map[string]bool{
		"app.js":    true,
		"config.js": true,
		"main.js":   true,
		"model.js":  true,
	}
	if !reflect.DeepEqual(set, expected) {
		t.Fatalf("expected reachable module set %#v, got %#v", expected, set)
	}
}

func TestResolverModuleGraphReachableModuleSetForUnknownEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()

	set, err := graph.ReachableModuleSet("main.js")
	if err != nil {
		t.Fatalf("ReachableModuleSet returned error: %v", err)
	}
	expected := map[string]bool{"main.js": true}
	if !reflect.DeepEqual(set, expected) {
		t.Fatalf("expected reachable module set %#v, got %#v", expected, set)
	}
}

func TestResolverModuleGraphReachableModuleSetReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableModuleSet("main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
