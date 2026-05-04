package resolver

import "jayess-go/ast"

func (g *ModuleGraph) AddProgramModule(modulePath string, program *ast.Program) ([]ResolvedModuleDependency, error) {
	dependencies, err := ResolveModuleDependencies(modulePath, program)
	if err != nil {
		return nil, err
	}
	g.AddResolvedModule(modulePath, dependencies)
	return dependencies, nil
}

func (g *ModuleGraph) AddCompactProgramModule(modulePath string, program *ast.Program) ([]ResolvedModuleDependency, error) {
	dependencies, err := ResolveCompactModuleDependencies(modulePath, program)
	if err != nil {
		return nil, err
	}
	g.AddResolvedModule(modulePath, dependencies)
	return dependencies, nil
}
