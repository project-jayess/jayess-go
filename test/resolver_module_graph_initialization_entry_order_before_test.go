package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksEntryInitializationOrderBefore(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	before, err := graph.InitializesBefore("main.js", "model.js", "main.js")
	if err != nil {
		t.Fatalf("InitializesBefore returned error: %v", err)
	}
	if !before {
		t.Fatal("expected model.js to initialize before main.js")
	}

	before, err = graph.InitializesBefore("main.js", "main.js", "model.js")
	if err != nil {
		t.Fatalf("InitializesBefore returned error: %v", err)
	}
	if before {
		t.Fatal("did not expect main.js to initialize before model.js")
	}
}

func TestResolverModuleGraphEntryInitializationOrderBeforeMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", nil)

	before, err := graph.InitializesBefore("main.js", "app.js", "missing.js")
	if err != nil {
		t.Fatalf("InitializesBefore returned error: %v", err)
	}
	if before {
		t.Fatal("did not expect missing module to participate in initialization order comparison")
	}
}

func TestResolverModuleGraphEntryInitializationOrderBeforeSameModuleIsFalse(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	before, err := graph.InitializesBefore("main.js", "main.js", "main.js")
	if err != nil {
		t.Fatalf("InitializesBefore returned error: %v", err)
	}
	if before {
		t.Fatal("did not expect a module to initialize before itself")
	}
}

func TestResolverModuleGraphEntryInitializationOrderBeforeReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializesBefore("main.js", "a.js", "main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
