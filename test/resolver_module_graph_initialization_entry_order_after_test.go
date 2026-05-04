package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksEntryInitializationOrderAfter(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	after, err := graph.InitializesAfter("main.js", "main.js", "model.js")
	if err != nil {
		t.Fatalf("InitializesAfter returned error: %v", err)
	}
	if !after {
		t.Fatal("expected main.js to initialize after model.js")
	}

	after, err = graph.InitializesAfter("main.js", "model.js", "main.js")
	if err != nil {
		t.Fatalf("InitializesAfter returned error: %v", err)
	}
	if after {
		t.Fatal("did not expect model.js to initialize after main.js")
	}
}

func TestResolverModuleGraphEntryInitializationOrderAfterMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", nil)

	after, err := graph.InitializesAfter("main.js", "app.js", "missing.js")
	if err != nil {
		t.Fatalf("InitializesAfter returned error: %v", err)
	}
	if after {
		t.Fatal("did not expect missing module to participate in initialization order comparison")
	}
}

func TestResolverModuleGraphEntryInitializationOrderAfterSameModuleIsFalse(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	after, err := graph.InitializesAfter("main.js", "main.js", "main.js")
	if err != nil {
		t.Fatalf("InitializesAfter returned error: %v", err)
	}
	if after {
		t.Fatal("did not expect a module to initialize after itself")
	}
}

func TestResolverModuleGraphEntryInitializationOrderAfterReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializesAfter("main.js", "main.js", "a.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
