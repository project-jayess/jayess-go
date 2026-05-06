package lowering

import "jayess-go/ast"

func lowerExportBindings(declaration *ast.ExportDecl) []ModuleExportBinding {
	if declaration.All || declaration.Namespace != "" {
		return []ModuleExportBinding{{
			Source:    declaration.Source,
			All:       declaration.All,
			Namespace: declaration.Namespace,
			Exported:  declaration.Namespace,
		}}
	}
	if len(declaration.Specifiers) > 0 {
		exports := make([]ModuleExportBinding, 0, len(declaration.Specifiers))
		for _, specifier := range declaration.Specifiers {
			exports = append(exports, ModuleExportBinding{
				Source:   declaration.Source,
				Local:    specifier.Local,
				Exported: specifier.Exported,
			})
		}
		return exports
	}
	if declaration.Default {
		return []ModuleExportBinding{defaultExportBinding(declaration)}
	}
	locals := exportedDeclarationLocals(declaration.Declaration)
	exports := make([]ModuleExportBinding, 0, len(locals))
	for _, local := range locals {
		exports = append(exports, ModuleExportBinding{Local: local, Exported: local})
	}
	return exports
}

func defaultExportBinding(declaration *ast.ExportDecl) ModuleExportBinding {
	local := "default"
	for _, name := range exportedDeclarationLocals(declaration.Declaration) {
		local = name
		break
	}
	return ModuleExportBinding{
		Local:    local,
		Exported: "default",
		Default:  true,
	}
}

func exportedDeclarationLocals(statement ast.Statement) []string {
	switch stmt := statement.(type) {
	case *ast.VariableDecl:
		if stmt.Name != "" {
			return []string{stmt.Name}
		}
		return bindingNames(stmt.Pattern)
	case *ast.FunctionDecl:
		if stmt.Name != "" {
			return []string{stmt.Name}
		}
	case *ast.ClassDecl:
		if stmt.Name != "" {
			return []string{stmt.Name}
		}
	}
	return nil
}

func bindingNames(pattern ast.BindingPattern) []string {
	switch binding := pattern.(type) {
	case *ast.BindingName:
		if binding.Name == "" {
			return nil
		}
		return []string{binding.Name}
	case *ast.BindingDefault:
		return bindingNames(binding.Pattern)
	case *ast.BindingRest:
		return bindingNames(binding.Pattern)
	case *ast.ArrayBindingPattern:
		var names []string
		for _, element := range binding.Elements {
			names = append(names, bindingNames(element)...)
		}
		return names
	case *ast.ObjectBindingPattern:
		var names []string
		for _, property := range binding.Properties {
			names = append(names, bindingNames(property.Pattern)...)
		}
		return names
	default:
		return nil
	}
}
