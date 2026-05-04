package test

import (
	"strings"
	"testing"

	"jayess-go/binding"
)

func requireStringSlice(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	}
}

func requireDiagnostic(t *testing.T, diagnostics []binding.Diagnostic, text string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, text) {
			return
		}
	}
	t.Fatalf("expected diagnostic containing %q, got %#v", text, diagnostics)
}
