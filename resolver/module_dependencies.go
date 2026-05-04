package resolver

import (
	"fmt"

	"jayess-go/ast"
)

type ResolvedModuleDependency struct {
	Source     string
	Path       string
	ReExport   bool
	SideEffect bool
}

func ResolveModuleDependencies(fromPath string, program *ast.Program) ([]ResolvedModuleDependency, error) {
	dependencies := ast.ModuleDependencies(program)
	if len(dependencies) == 0 {
		return nil, nil
	}
	return resolveModuleDependencies(fromPath, dependencies)
}

func resolveModuleDependencies(fromPath string, dependencies []ast.ModuleDependency) ([]ResolvedModuleDependency, error) {
	resolved := make([]ResolvedModuleDependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		path, err := ResolveImport(fromPath, dependency.Source)
		if err != nil {
			return nil, fmt.Errorf("resolve module dependency %q: %w", dependency.Source, err)
		}
		resolved = append(resolved, ResolvedModuleDependency{
			Source:     dependency.Source,
			Path:       path,
			ReExport:   dependency.ReExport,
			SideEffect: dependency.SideEffect,
		})
	}
	return resolved, nil
}
