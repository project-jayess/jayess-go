package test

import (
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksTransitiveDependents(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)

	if !graph.TransitivelyDependedOnBy("model.js", "main.js") {
		t.Fatalf("expected model.js to be transitively depended on by main.js")
	}
	if graph.TransitivelyDependedOnBy("main.js", "model.js") {
		t.Fatalf("did not expect main.js to be transitively depended on by model.js")
	}
	if graph.TransitivelyDependedOnBy("missing.js", "main.js") {
		t.Fatalf("did not expect missing module to have transitive dependents")
	}
}
