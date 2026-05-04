package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphOrdersAllModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("shared.js", nil)

	order, err := graph.InitializationOrderAll()
	if err != nil {
		t.Fatalf("InitializationOrderAll returned error: %v", err)
	}
	expected := []string{"config.js", "main.js", "shared.js", "worker.js"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("expected initialization order %#v, got %#v", expected, order)
	}
}

func TestResolverModuleGraphOrdersAllModulesDetectsDisconnectedCycle(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationOrderAll()
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
