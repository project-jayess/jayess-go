package lowering

import "jayess-go/ast"

type ModuleBindingPlan struct {
	Module  string
	Imports []ModuleImportBinding
	Exports []ModuleExportBinding
}

type ModuleImportBinding struct {
	Source     string
	Imported   string
	Local      string
	Default    bool
	Namespace  bool
	SideEffect bool
}

type ModuleExportBinding struct {
	Source    string
	Local     string
	Exported  string
	Default   bool
	All       bool
	Namespace string
}

func LowerModuleBindingPlan(module string, program *ast.Program) ModuleBindingPlan {
	plan := ModuleBindingPlan{Module: module}
	if program == nil {
		return plan
	}
	for _, statement := range program.Statements {
		switch stmt := statement.(type) {
		case *ast.ImportDecl:
			plan.Imports = append(plan.Imports, lowerImportBindings(stmt)...)
		case *ast.ExportDecl:
			plan.Exports = append(plan.Exports, lowerExportBindings(stmt)...)
		}
	}
	return plan
}

func lowerImportBindings(declaration *ast.ImportDecl) []ModuleImportBinding {
	if declaration.SideEffect {
		return []ModuleImportBinding{{
			Source:     declaration.Source,
			SideEffect: true,
		}}
	}
	imports := make([]ModuleImportBinding, 0, len(declaration.Specifiers))
	for _, specifier := range declaration.Specifiers {
		imports = append(imports, ModuleImportBinding{
			Source:    declaration.Source,
			Imported:  specifier.Imported,
			Local:     specifier.Local,
			Default:   specifier.Default,
			Namespace: specifier.Namespace,
		})
	}
	return imports
}
