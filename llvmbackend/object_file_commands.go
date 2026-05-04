package llvmbackend

import "path/filepath"

func ObjectFileIRCommand(target TargetConfig, irPath string, outputPath string) ToolchainCommand {
	return ToolchainCommand{
		Step:    ClangCompileStep,
		Program: "clang",
		Args:    []string{"-target", target.Triple, "-c", irPath, "-o", outputPath},
	}
}

func ObjectFileIRPath(outputPath string, workDir string) string {
	if workDir == "" {
		workDir = filepath.Dir(outputPath)
	}
	base := filepath.Base(outputPath)
	ext := filepath.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	if base == "" {
		base = "jayess"
	}
	return filepath.Join(workDir, base+".ll")
}
