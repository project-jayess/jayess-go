package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksEntryInitializationSameBatch(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	same, err := graph.InitializesInSameBatch("main.js", "config.js", "model.js")
	if err != nil {
		t.Fatalf("InitializesInSameBatch returned error: %v", err)
	}
	if !same {
		t.Fatal("expected config.js and model.js to initialize in the same batch")
	}

	same, err = graph.InitializesInSameBatch("main.js", "app.js", "main.js")
	if err != nil {
		t.Fatalf("InitializesInSameBatch returned error: %v", err)
	}
	if same {
		t.Fatal("did not expect app.js and main.js to initialize in the same batch")
	}
}

func TestResolverModuleGraphEntryInitializationSameBatchMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", nil)

	same, err := graph.InitializesInSameBatch("main.js", "app.js", "missing.js")
	if err != nil {
		t.Fatalf("InitializesInSameBatch returned error: %v", err)
	}
	if same {
		t.Fatal("did not expect missing module to participate in an initialization batch")
	}
}

func TestResolverModuleGraphEntryInitializationSameBatchSameModule(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	same, err := graph.InitializesInSameBatch("main.js", "main.js", "main.js")
	if err != nil {
		t.Fatalf("InitializesInSameBatch returned error: %v", err)
	}
	if !same {
		t.Fatal("expected a module to share an initialization batch with itself")
	}
}

func TestResolverModuleGraphEntryInitializationSameBatchReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializesInSameBatch("main.js", "a.js", "b.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
