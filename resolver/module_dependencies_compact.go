package resolver

import "jayess-go/ast"

func ResolveCompactModuleDependencies(fromPath string, program *ast.Program) ([]ResolvedModuleDependency, error) {
	dependencies := ast.CompactModuleDependencies(ast.ModuleDependencies(program))
	if len(dependencies) == 0 {
		return nil, nil
	}
	resolved, err := resolveModuleDependencies(fromPath, dependencies)
	if err != nil {
		return nil, err
	}
	return CompactResolvedModuleDependencies(resolved), nil
}
