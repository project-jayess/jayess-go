package llvmbackend

type SharedLibraryBuildPlan struct {
	InputIR              string
	OutputPath           string
	Target               TargetConfig
	Steps                []ToolchainStep
	ExtraObjectFiles     []string
	LinkFlags            []string
	Diagnostics          []string
	ToolchainDiagnostics []string
}

func PlanSharedLibraryFromIR(inputIR string, outputPath string, target TargetConfig) SharedLibraryBuildPlan {
	interop := DefaultToolchainInterop()
	var diagnostics []string
	if inputIR == "" {
		diagnostics = append(diagnostics, "missing LLVM IR input")
	}
	if outputPath == "" {
		diagnostics = append(diagnostics, "missing shared library output path")
	}
	if target.Triple == "" {
		diagnostics = append(diagnostics, "missing LLVM target triple")
	}
	return SharedLibraryBuildPlan{
		InputIR:              inputIR,
		OutputPath:           outputPath,
		Target:               target,
		Steps:                []ToolchainStep{LLVMVerifyStep, LLCStep, ClangLinkStep},
		LinkFlags:            SharedLibraryLinkFlags(target),
		Diagnostics:          diagnostics,
		ToolchainDiagnostics: append([]string{}, interop.Diagnostics...),
	}
}

func (plan SharedLibraryBuildPlan) CanBuildSharedLibrary() bool {
	return plan.InputIR != "" && plan.OutputPath != "" && plan.Target.Triple != "" && len(plan.Diagnostics) == 0
}
