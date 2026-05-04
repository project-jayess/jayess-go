package e2e

import (
	"strings"

	"jayess-go/binding"
	"jayess-go/llvmbackend"
)

type ScenarioKind string

const (
	NativeBindingExecutable ScenarioKind = "native-binding-executable"
	AudioBindingExecutable  ScenarioKind = "audio-binding-executable"
	LLVMExecutable          ScenarioKind = "llvm-executable"
)

type Scenario struct {
	Name       string
	Kind       ScenarioKind
	SourceFile string
	Bindings   []binding.Module
	IR         string
	Target     llvmbackend.TargetConfig
}

type Plan struct {
	Scenario        Scenario
	BindingPlan     binding.BuildPlan
	ExecutablePlan  llvmbackend.ExecutableBuildPlan
	RequiredOutputs []string
	Diagnostics     []string
}

func PlanScenario(scenario Scenario, runtimeHeaderDir string, outputPath string) Plan {
	bindingPlan := binding.PlanBuild(scenario.Bindings, scenario.Target.Name, runtimeHeaderDir)
	executablePlan := llvmbackend.PlanExecutableFromIR(scenario.IR, outputPath, scenario.Target)
	diagnostics := validateScenario(scenario, outputPath)
	return Plan{
		Scenario:       scenario,
		BindingPlan:    bindingPlan,
		ExecutablePlan: executablePlan,
		RequiredOutputs: []string{
			outputPath,
		},
		Diagnostics: diagnostics,
	}
}

func (plan Plan) Ready() bool {
	return plan.ExecutablePlan.CanBuildExecutable() && len(plan.BindingPlan.Diagnostics) == 0 && len(plan.Diagnostics) == 0
}

func validateScenario(scenario Scenario, outputPath string) []string {
	var diagnostics []string
	if scenario.Name == "" {
		diagnostics = append(diagnostics, "missing e2e scenario name")
	}
	if scenario.Kind == "" {
		diagnostics = append(diagnostics, "missing e2e scenario kind")
	} else if !isScenarioKind(scenario.Kind) {
		diagnostics = append(diagnostics, "unknown e2e scenario kind")
	}
	if scenario.SourceFile == "" {
		diagnostics = append(diagnostics, "missing Jayess source file")
	} else if !strings.HasSuffix(scenario.SourceFile, ".js") {
		diagnostics = append(diagnostics, "Jayess source file must use .js extension")
	}
	if outputPath != "" && !strings.HasPrefix(outputPath, "./temp/") && !strings.HasPrefix(outputPath, "temp/") {
		diagnostics = append(diagnostics, "e2e executable outputs must be placed in ./temp")
	}
	return diagnostics
}

func isScenarioKind(kind ScenarioKind) bool {
	switch kind {
	case NativeBindingExecutable, AudioBindingExecutable, LLVMExecutable:
		return true
	default:
		return false
	}
}
