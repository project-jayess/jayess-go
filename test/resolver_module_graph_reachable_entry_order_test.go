package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsReachableModuleOrderForEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js", "app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)

	order, err := graph.ReachableModuleOrder("main.js")
	if err != nil {
		t.Fatalf("ReachableModuleOrder returned error: %v", err)
	}
	expected := []string{"config.js", "model.js", "app.js", "main.js"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("expected reachable order %#v, got %#v", expected, order)
	}
}

func TestResolverModuleGraphReachableModuleOrderForUnknownEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()

	order, err := graph.ReachableModuleOrder("main.js")
	if err != nil {
		t.Fatalf("ReachableModuleOrder returned error: %v", err)
	}
	expected := []string{"main.js"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("expected unknown entry order %#v, got %#v", expected, order)
	}
}

func TestResolverModuleGraphReachableModuleOrderReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableModuleOrder("main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
