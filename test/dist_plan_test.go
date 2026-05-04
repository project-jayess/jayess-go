package test

import (
	"path/filepath"
	"testing"

	"jayess-go/dist"
)

func TestDistPlanUsesPlatformDirectory(t *testing.T) {
	plan, err := dist.BuildPlan(dist.Config{
		Platform: "linux-x64",
		Version:  "0.1.0",
		OutDir:   "dist",
	})
	if err != nil {
		t.Fatal(err)
	}
	expectedRoot := filepath.Join("dist", "linux-x64", "jayess-0.1.0-linux-x64")
	if plan.Root != expectedRoot {
		t.Fatalf("expected root %q, got %q", expectedRoot, plan.Root)
	}
	expectedTools := filepath.Join(expectedRoot, "tools", "bin")
	if plan.ToolBinDir != expectedTools {
		t.Fatalf("expected tool dir %q, got %q", expectedTools, plan.ToolBinDir)
	}
	expectedLicenses := filepath.Join(expectedRoot, "licenses")
	if plan.LicenseDir != expectedLicenses {
		t.Fatalf("expected license dir %q, got %q", expectedLicenses, plan.LicenseDir)
	}
	if filepath.Base(plan.CompilerPath) != "jayess" {
		t.Fatalf("expected linux compiler name jayess, got %q", plan.CompilerPath)
	}
	if filepath.Ext(plan.ArchivePath) != ".gz" {
		t.Fatalf("expected tar.gz archive path, got %q", plan.ArchivePath)
	}
}

func TestDistPlanUsesWindowsZip(t *testing.T) {
	plan, err := dist.BuildPlan(dist.Config{
		Platform: "windows-x64",
		Version:  "0.1.0",
		OutDir:   "dist",
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(plan.CompilerPath) != "jayess.exe" {
		t.Fatalf("expected windows compiler name jayess.exe, got %q", plan.CompilerPath)
	}
	if filepath.Ext(plan.ArchivePath) != ".zip" {
		t.Fatalf("expected zip archive path, got %q", plan.ArchivePath)
	}
}
