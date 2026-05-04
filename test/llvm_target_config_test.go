package test

import (
	"testing"

	"jayess-go/llvmbackend"
)

func TestLLVMTargetConfigSelection(t *testing.T) {
	config, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux-x64 target config")
	}
	if config.Triple != "x86_64-pc-linux-gnu" {
		t.Fatalf("unexpected target triple %q", config.Triple)
	}
	config = llvmbackend.WithCPU(config, "native")
	config = llvmbackend.WithFeatures(config, "+sse4.2", "+avx2")
	if config.CPU != "native" {
		t.Fatalf("expected native CPU, got %q", config.CPU)
	}
	requireStringSlice(t, config.Features, []string{"+sse4.2", "+avx2"})
}

func TestLLVMHostTargetConfigDetection(t *testing.T) {
	config, ok := llvmbackend.HostTargetConfig()
	if !ok {
		t.Skip("current host target is not in Jayess target metadata")
	}
	if config.Triple == "" {
		t.Fatalf("expected host target triple")
	}
}

func TestLLVMRelocationAndCodeModelDefaults(t *testing.T) {
	config, ok := llvmbackend.TargetConfigFor("windows-x64")
	if !ok {
		t.Fatal("expected windows-x64 target config")
	}
	if config.RelocationModel != llvmbackend.RelocDefault {
		t.Fatalf("expected default relocation model, got %s", config.RelocationModel)
	}
	if config.CodeModel != llvmbackend.CodeModelDefault {
		t.Fatalf("expected default code model, got %s", config.CodeModel)
	}
}
