package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsInitializationBatchWidthsForEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)

	widths, err := graph.InitializationBatchWidths("main.js")
	if err != nil {
		t.Fatalf("InitializationBatchWidths returned error: %v", err)
	}

	expected := []int{2, 1, 1}
	if !reflect.DeepEqual(widths, expected) {
		t.Fatalf("expected initialization batch widths %#v, got %#v", expected, widths)
	}
}

func TestResolverModuleGraphInitializationBatchWidthsForUnknownEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()

	widths, err := graph.InitializationBatchWidths("main.js")
	if err != nil {
		t.Fatalf("InitializationBatchWidths returned error: %v", err)
	}
	expected := []int{1}
	if !reflect.DeepEqual(widths, expected) {
		t.Fatalf("expected unknown entry widths %#v, got %#v", expected, widths)
	}
}

func TestResolverModuleGraphInitializationBatchWidthsReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationBatchWidths("main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
