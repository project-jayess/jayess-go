package test

import (
	"strings"
	"testing"

	"jayess-go/binding"
)

func TestBindingModuleClassification(t *testing.T) {
	if !binding.IsBindingModulePath("./native/math.bind.js") {
		t.Fatal("expected .bind.js module path")
	}
	if binding.ClassifyModulePath("./native/math.bind.js") != binding.NativeBindingModule {
		t.Fatal("expected native binding module classification")
	}
	if binding.ClassifyModulePath("./native/math.js") != binding.SourceModule {
		t.Fatal("expected normal Jayess source module classification")
	}
}

func TestBindingTargetValidationRejectsUnsupportedFormats(t *testing.T) {
	for _, target := range []string{"./native/math.json", "./native/math.c"} {
		err := binding.ValidateBindingTarget(target)
		if err == nil {
			t.Fatalf("expected unsupported binding target diagnostic for %s", target)
		}
		if !strings.Contains(err.Error(), ".js") {
			t.Fatalf("expected .js diagnostic, got %v", err)
		}
	}
}

func TestBindingTargetValidationAcceptsAnyJSFile(t *testing.T) {
	for _, target := range []string{"./native/math.js", "./native/math.bind.js"} {
		if err := binding.ValidateBindingTarget(target); err != nil {
			t.Fatalf("expected valid binding target for %s, got %v", target, err)
		}
	}
}

func TestBindingTargetValidationRejectsMalformedTargets(t *testing.T) {
	for _, target := range []string{" ./native/math.bind.js", "./native\\math.bind.js", "./native/math.bind.js?raw"} {
		err := binding.ValidateBindingTarget(target)
		if err == nil {
			t.Fatalf("expected malformed target diagnostic for %q", target)
		}
		if !strings.Contains(err.Error(), "binding target") {
			t.Fatalf("expected binding target diagnostic, got %v", err)
		}
	}
}
