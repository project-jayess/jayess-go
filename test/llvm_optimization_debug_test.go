package test

import (
	"testing"

	"jayess-go/llvmbackend"
)

func TestLLVMOptimizationPipelines(t *testing.T) {
	for _, level := range []llvmbackend.OptLevel{
		llvmbackend.OptO0,
		llvmbackend.OptO1,
		llvmbackend.OptO2,
		llvmbackend.OptO3,
		llvmbackend.OptOz,
	} {
		pipeline := llvmbackend.OptimizationPipelineFor(level)
		if pipeline.Level != level || !pipeline.VerifyIR || len(pipeline.Passes) == 0 {
			t.Fatalf("expected configured optimization pipeline for %s: %#v", level, pipeline)
		}
	}
	if !llvmbackend.OptimizationPipelineFor(llvmbackend.OptO0).DebugFriendly {
		t.Fatal("expected O0 pipeline to be debug-friendly")
	}
}

func TestLLVMDebugConfigPreservesSourceMapping(t *testing.T) {
	config := llvmbackend.DefaultDebugConfig(true)
	if config.Kind != llvmbackend.DebugInfoDWARF {
		t.Fatalf("expected DWARF debug info, got %s", config.Kind)
	}
	if !config.PreserveSourceLocations || !config.PreserveFunctionNames || !config.CrashMapping {
		t.Fatalf("expected debuggable LLVM config, got %#v", config)
	}
}
