package llvmbackend

import (
	"path/filepath"
	"strings"
)

func SharedLibraryToolchainCommands(plan SharedLibraryBuildPlan, workDir string) []ToolchainCommand {
	if !plan.CanBuildSharedLibrary() {
		return nil
	}
	return []ToolchainCommand{SharedLibraryIRCommand(plan, workDir)}
}

func SharedLibraryIRCommand(plan SharedLibraryBuildPlan, workDir string) ToolchainCommand {
	return ToolchainCommand{
		Step:    ClangLinkStep,
		Program: "clang",
		Args:    sharedLibraryClangArgs(plan, SharedLibraryIRPath(plan, workDir)),
	}
}

func SharedLibraryLinkCommand(plan SharedLibraryBuildPlan, workDir string) ToolchainCommand {
	return ToolchainCommand{
		Step:    ClangLinkStep,
		Program: "clang",
		Args:    sharedLibraryClangArgs(plan, SharedLibraryObjectPath(plan, workDir)),
	}
}

func SharedLibraryIRPath(plan SharedLibraryBuildPlan, workDir string) string {
	return filepath.Join(sharedLibraryWorkDir(plan, workDir), sharedLibraryCommandBase(plan.OutputPath)+".ll")
}

func SharedLibraryObjectPath(plan SharedLibraryBuildPlan, workDir string) string {
	return filepath.Join(sharedLibraryWorkDir(plan, workDir), sharedLibraryCommandBase(plan.OutputPath)+targetObjectExtension(plan.Target))
}

func sharedLibraryWorkDir(plan SharedLibraryBuildPlan, workDir string) string {
	if workDir != "" {
		return workDir
	}
	return filepath.Dir(plan.OutputPath)
}

func sharedLibraryClangArgs(plan SharedLibraryBuildPlan, objectPath string) []string {
	args := []string{"-target", plan.Target.Triple, objectPath}
	args = append(args, plan.ExtraObjectFiles...)
	args = append(args, plan.LinkFlags...)
	args = append(args, "-o", plan.OutputPath)
	return args
}

func sharedLibraryCommandBase(outputPath string) string {
	base := filepath.Base(outputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		return "jayess"
	}
	return name
}

func targetObjectExtension(target TargetConfig) string {
	if target.Name == "windows-x64" {
		return ".obj"
	}
	return ".o"
}
