package appdist

import "jayess-go/binding"

type DependencyKind string

const (
	SourceModuleDependency    DependencyKind = "source-module"
	JayessPackageDependency   DependencyKind = "jayess-package"
	NativeBindingDependency   DependencyKind = "native-binding"
	BuiltinPackageDependency  DependencyKind = "builtin-package"
	ExternalPackageDependency DependencyKind = "external-package"
)

type ImportedDependency struct {
	ImportPath   string
	ResolvedPath string
	PackageRoot  string
	Kind         DependencyKind
	BindingPlan  binding.BuildPlan
	Metadata     PackageMetadata
}

type DependencyGraph struct {
	Dependencies []ImportedDependency
}

func (graph DependencyGraph) Kinds() map[DependencyKind]int {
	counts := map[DependencyKind]int{}
	for _, dependency := range graph.Dependencies {
		counts[dependency.Kind]++
	}
	return counts
}

func PlanApplicationFromDependencies(executablePath string, outputDir string, graph DependencyGraph, targetName string) Plan {
	if outputDir == "" {
		outputDir = defaultOutputDir(executablePath)
	}
	assets, diagnostics := ResolveDependencyAssets(graph, targetName)
	return Plan{
		ExecutablePath: executablePath,
		OutputDir:      outputDir,
		Assets:         assets,
		Diagnostics:    diagnostics,
	}
}
