package test

import (
	"testing"

	"jayess-go/mongoose"
)

func TestMongooseEventCallbackRules(t *testing.T) {
	rules := mongoose.EventRules()
	for _, want := range []mongoose.EventRule{
		mongoose.JayessCallbackSafe,
		mongoose.CallbackLifetimeSafe,
		mongoose.ServerLoopIntegration,
		mongoose.ErrorDiagnostics,
	} {
		if !hasMongooseEventRule(rules, want) {
			t.Fatalf("expected Mongoose event rule %s in %#v", want, rules)
		}
	}
}

func TestMongooseDiagnosticKinds(t *testing.T) {
	kinds := mongoose.DiagnosticKinds()
	for _, want := range []mongoose.DiagnosticKind{
		mongoose.MissingHeaders,
		mongoose.MissingSource,
		mongoose.BuildFailure,
		mongoose.RuntimeError,
	} {
		if !hasMongooseDiagnosticKind(kinds, want) {
			t.Fatalf("expected Mongoose diagnostic kind %s in %#v", want, kinds)
		}
	}
}

func hasMongooseEventRule(rules []mongoose.EventRule, want mongoose.EventRule) bool {
	for _, rule := range rules {
		if rule == want {
			return true
		}
	}
	return false
}

func hasMongooseDiagnosticKind(kinds []mongoose.DiagnosticKind, want mongoose.DiagnosticKind) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}
	return false
}
