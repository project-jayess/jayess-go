package llvmbackend

type ExecutableBuildPlan struct {
	InputIR              string
	OutputPath           string
	Target               TargetConfig
	Steps                []ToolchainStep
	ExtraObjectFiles     []string
	LinkFlags            []string
	Diagnostics          []string
	ToolchainDiagnostics []string
}

func PlanExecutableFromIR(inputIR string, outputPath string, target TargetConfig) ExecutableBuildPlan {
	interop := DefaultToolchainInterop()
	var diagnostics []string
	if inputIR == "" {
		diagnostics = append(diagnostics, "missing LLVM IR input")
	}
	if outputPath == "" {
		diagnostics = append(diagnostics, "missing executable output path")
	}
	if target.Triple == "" {
		diagnostics = append(diagnostics, "missing LLVM target triple")
	}
	return ExecutableBuildPlan{
		InputIR:              inputIR,
		OutputPath:           outputPath,
		Target:               target,
		Steps:                []ToolchainStep{LLVMVerifyStep, LLCStep, ClangLinkStep},
		Diagnostics:          diagnostics,
		ToolchainDiagnostics: append([]string{}, interop.Diagnostics...),
	}
}

func (plan ExecutableBuildPlan) CanBuildExecutable() bool {
	return plan.InputIR != "" && plan.OutputPath != "" && plan.Target.Triple != "" && len(plan.Diagnostics) == 0
}
