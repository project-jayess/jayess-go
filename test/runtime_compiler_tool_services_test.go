package test

import (
	"os"
	"path/filepath"
	"testing"

	"jayess-go/appdist"
	"jayess-go/binding"
	jayessruntime "jayess-go/runtime"
)

func TestCompilerToolRuntimeServicesAreExplicit(t *testing.T) {
	runtime := jayessruntime.DefaultCompilerToolRuntime()
	for _, service := range []jayessruntime.CompilerToolService{
		jayessruntime.SourceService,
		jayessruntime.PathService,
		jayessruntime.DiagnosticService,
		jayessruntime.DataService,
		jayessruntime.LLVMService,
		jayessruntime.LinkerService,
		jayessruntime.DistributionService,
	} {
		if !runtime.Has(service) {
			t.Fatalf("expected compiler tool service %s", service)
		}
	}
	if diagnostics := jayessruntime.ValidateCompilerToolRuntime(runtime); len(diagnostics) != 0 {
		t.Fatalf("expected default compiler tool runtime to validate, got %#v", diagnostics)
	}
}

func TestCompilerToolRuntimeReportsMissingServices(t *testing.T) {
	diagnostics := jayessruntime.ValidateCompilerToolRuntime(jayessruntime.CompilerToolRuntime{
		Services: []jayessruntime.CompilerToolService{jayessruntime.SourceService},
	})
	if len(diagnostics) == 0 {
		t.Fatal("expected missing service diagnostics")
	}
}

func TestCompilerUtilityDistributionCopiesExecutableAndAssets(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "jayess-lexer")
	license := filepath.Join(root, "LICENSE.txt")
	library := filepath.Join(root, "libsupport.so")
	writeFile(t, executable, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(executable, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, license, "license")
	writeFile(t, library, "shared")

	plan := appdist.PlanCompilerUtility(executable, filepath.Join(root, "dist"), binding.BuildPlan{
		SharedLibraryFiles: []string{library},
		LicenseFiles:       []string{license},
	}, "linux-x64")
	result, err := appdist.Create(plan)
	if err != nil {
		t.Fatal(err)
	}
	requireFile(t, filepath.Join(result.OutputDir, "jayess-lexer"))
	requireFile(t, filepath.Join(result.OutputDir, "libsupport.so"))
	requireFile(t, filepath.Join(result.OutputDir, "licenses", "LICENSE.txt"))
}
