package tooling

import (
	"jayess-go/binding"
	"jayess-go/llvmbackend"
)

type CompileRequest struct {
	Emit             EmitKind
	InputIR          string
	OutputPath       string
	Target           llvmbackend.TargetConfig
	BindingModules   []binding.Module
	BindingPlatform  string
	RuntimeHeaderDir string
}

type CompilePlan struct {
	Request           CompileRequest
	Artifact          llvmbackend.ArtifactKind
	Executable        llvmbackend.ExecutableBuildPlan
	SharedLibrary     llvmbackend.SharedLibraryBuildPlan
	BindingBuild      binding.BuildPlan
	Diagnostics       []string
	UnsupportedReason string
}

func PlanCompileFromIR(request CompileRequest) CompilePlan {
	plan := CompilePlan{Request: request, Artifact: artifactForEmit(request.Emit)}
	plan.BindingBuild = binding.PlanBuild(request.BindingModules, request.BindingPlatform, request.RuntimeHeaderDir)
	plan.Diagnostics = append(plan.Diagnostics, bindingDiagnosticMessages(plan.BindingBuild.Diagnostics)...)
	bindingObjects := bindingObjectFiles(plan.BindingBuild, request.Target)
	if !HasEmitKind(request.Emit) {
		plan.Diagnostics = append(plan.Diagnostics, "unsupported emit kind")
		plan.UnsupportedReason = "unsupported emit kind"
		return plan
	}
	switch request.Emit {
	case EmitNative:
		executable := llvmbackend.PlanExecutableFromIR(request.InputIR, request.OutputPath, request.Target)
		executable.ExtraObjectFiles = bindingObjects
		executable.LinkFlags = append(executable.LinkFlags, plan.BindingBuild.LDFlags...)
		plan.Executable = executable
		plan.Diagnostics = append(plan.Diagnostics, executable.Diagnostics...)
	case EmitShared:
		shared := llvmbackend.PlanSharedLibraryFromIR(request.InputIR, request.OutputPath, request.Target)
		shared.ExtraObjectFiles = bindingObjects
		shared.LinkFlags = append(shared.LinkFlags, plan.BindingBuild.LDFlags...)
		plan.SharedLibrary = shared
		plan.Diagnostics = append(plan.Diagnostics, shared.Diagnostics...)
	}
	return plan
}

func bindingDiagnosticMessages(diagnostics []binding.Diagnostic) []string {
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

func (plan CompilePlan) CanBuild() bool {
	if len(plan.Diagnostics) != 0 {
		return false
	}
	switch plan.Request.Emit {
	case EmitNative:
		return plan.Executable.CanBuildExecutable()
	case EmitShared:
		return plan.SharedLibrary.CanBuildSharedLibrary()
	default:
		return plan.UnsupportedReason == ""
	}
}
