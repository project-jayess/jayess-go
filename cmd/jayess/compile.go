package main

import (
	"fmt"

	"jayess-go/appdist"
	"jayess-go/binding"
	"jayess-go/llvmbackend"
	"jayess-go/resolver"
	"jayess-go/tooling"
)

func run(args []string) error {
	cfg, err := parseCLI(args)
	if err != nil {
		return err
	}
	return compile(cfg)
}

func compile(cfg cliConfig) error {
	target, err := resolveTarget(cfg.targetName)
	if err != nil {
		return err
	}
	input, err := lowerInput(cfg.inputPath, target)
	if err != nil {
		return err
	}
	out := outputPath(cfg, target)
	switch cfg.emit {
	case tooling.EmitLLVMIR:
		return writeFile(out, []byte(input.IR))
	case tooling.EmitObject:
		folded := lowerFoldedReturnCodeProgram(cfg.inputPath, target, input.Program)
		return compileObjectFile(llvmbackend.EmitLLVMIR(folded), target, out)
	case tooling.EmitShared:
		bindingModules, bindingDiagnostics, err := resolver.ResolveBindingModules(cfg.inputPath, input.Program)
		if err != nil {
			return err
		}
		plan := tooling.PlanCompileFromIR(tooling.CompileRequest{
			Emit:             cfg.emit,
			InputIR:          input.IR,
			OutputPath:       out,
			Target:           target,
			BindingModules:   bindingModules,
			BindingPlatform:  bindingPlatformForTarget(target.Name),
			RuntimeHeaderDir: "runtime",
		})
		plan.Diagnostics = append(plan.Diagnostics, bindingPlanDiagnosticMessages(bindingDiagnostics)...)
		return compileSharedLibrary(input.IR, plan)
	case tooling.EmitDist:
		if cfg.executablePath == "" {
			return fmt.Errorf("--emit=dist requires --executable")
		}
		bindingModules, bindingDiagnostics, err := resolver.ResolveBindingModules(cfg.inputPath, input.Program)
		if err != nil {
			return err
		}
		bindingPlan := binding.PlanBuild(bindingModules, bindingPlatformForTarget(target.Name), "runtime")
		bindingPlan.Diagnostics = append(bindingDiagnostics, bindingPlan.Diagnostics...)
		if len(bindingPlan.Diagnostics) != 0 {
			return fmt.Errorf("binding plan is not packageable: %v", bindingPlanDiagnosticMessages(bindingPlan.Diagnostics))
		}
		plan := appdist.PlanApplicationWithStdlibImports(
			cfg.executablePath,
			out,
			bindingPlan,
			target.Name,
			stdlibImportsFromProgram(input.Program),
			"runtime/assets",
		)
		if len(plan.Diagnostics) != 0 {
			return fmt.Errorf("app distribution plan has diagnostics: %v", plan.Diagnostics)
		}
		_, err = appdist.Create(plan)
		return err
	default:
		return fmt.Errorf("--emit=%s is parsed but not executable in this CLI yet", cfg.emit)
	}
}

func bindingPlatformForTarget(targetName string) string {
	switch targetName {
	case "macos-x64", "macos-arm64":
		return "darwin"
	case "windows-x64":
		return "windows"
	default:
		return "linux"
	}
}

func bindingPlanDiagnosticMessages(diagnostics []binding.Diagnostic) []string {
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Field == "" {
			messages = append(messages, diagnostic.Message)
			continue
		}
		messages = append(messages, diagnostic.Field+": "+diagnostic.Message)
	}
	return messages
}
