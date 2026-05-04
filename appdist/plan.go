package appdist

import (
	"path/filepath"

	"jayess-go/binding"
)

type RuntimeAsset struct {
	SourcePath string
	OutputName string
}

type Plan struct {
	ExecutablePath string
	OutputDir      string
	Assets         []RuntimeAsset
	Diagnostics    []string
}

func PlanApplication(executablePath string, outputDir string, bindingPlan binding.BuildPlan, targetName string) Plan {
	if outputDir == "" {
		outputDir = defaultOutputDir(executablePath)
	}
	assets, diagnostics := ResolveRuntimeAssets(bindingPlan, targetName)
	return Plan{
		ExecutablePath: executablePath,
		OutputDir:      outputDir,
		Assets:         assets,
		Diagnostics:    diagnostics,
	}
}

func defaultOutputDir(executablePath string) string {
	base := filepath.Base(executablePath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	if name == "" {
		name = "app"
	}
	return filepath.Join("dist", name)
}
