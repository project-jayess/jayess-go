package main

import "jayess-go/ast"

func stdlibImportsFromProgram(program *ast.Program) []string {
	if program == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var imports []string
	for _, statement := range program.Statements {
		importDecl, ok := statement.(*ast.ImportDecl)
		if !ok {
			continue
		}
		if _, exists := seen[importDecl.Source]; exists {
			continue
		}
		seen[importDecl.Source] = struct{}{}
		imports = append(imports, importDecl.Source)
	}
	return imports
}
