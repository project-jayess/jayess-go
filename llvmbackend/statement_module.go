package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
	nativebinding "jayess-go/binding"
	jayessruntime "jayess-go/runtime"
)

const (
	runtimeModuleExportBindingSymbol     = "jayess_module_export_binding"
	runtimeModuleExportValueSymbol       = "jayess_module_export_value"
	runtimeModuleImportBindingSymbol     = "jayess_module_import_binding"
	runtimeModuleImportDefaultSymbol     = "jayess_module_import_default"
	runtimeModuleImportNamespaceSymbol   = "jayess_module_import_namespace"
	runtimeModuleInitializeSymbol        = "jayess_module_initialize"
	runtimeModuleReexportAllSymbol       = "jayess_module_reexport_all"
	runtimeModuleReexportBindingSymbol   = "jayess_module_reexport_binding"
	runtimeModuleReexportNamespaceSymbol = "jayess_module_reexport_namespace"
	runtimeNativeBindingWrapperSymbol    = "jayess_native_binding_wrapper"
)

func (emitter *StatementEmitter) emitImportDeclaration(statement *ast.ImportDecl) error {
	if statement == nil {
		return fmt.Errorf("runtime import declaration must not be nil")
	}
	if err := emitter.emitModuleInitialize(statement.Source); err != nil {
		return err
	}
	if statement.SideEffect {
		return nil
	}
	for _, specifier := range statement.Specifiers {
		value, err := emitter.emitImportedBinding(statement.Source, specifier)
		if err != nil {
			return err
		}
		local := importedLocalName(specifier)
		if local == "" {
			continue
		}
		if err := emitter.expressions.DeclareLocal(local, value); err != nil {
			return err
		}
	}
	return nil
}

func (emitter *StatementEmitter) emitExportDeclaration(statement *ast.ExportDecl) error {
	if statement == nil {
		return fmt.Errorf("runtime export declaration must not be nil")
	}
	if statement.Declaration != nil {
		if err := emitter.EmitStatement(statement.Declaration); err != nil {
			return err
		}
	}
	if statement.Value != nil {
		value, err := emitter.expressions.EmitExpression(statement.Value)
		if err != nil {
			return err
		}
		return emitter.emitModuleExportValue("default", value)
	}
	if statement.All {
		return emitter.emitModuleReexportAll(statement.Source)
	}
	if statement.Namespace != "" {
		return emitter.emitModuleReexportNamespace(statement.Source, statement.Namespace)
	}
	for _, specifier := range statement.Specifiers {
		if statement.Source != "" {
			if err := emitter.emitModuleReexportBinding(statement.Source, specifier.Local, specifier.Exported); err != nil {
				return err
			}
			continue
		}
		if err := emitter.emitModuleExportBinding(specifier.Local, specifier.Exported); err != nil {
			return err
		}
	}
	if len(statement.Specifiers) == 0 {
		for _, local := range exportDeclarationLocalNames(statement.Declaration) {
			exported := local
			if statement.Default {
				exported = "default"
			}
			if err := emitter.emitModuleExportBinding(local, exported); err != nil {
				return err
			}
		}
	}
	return nil
}

func (emitter *StatementEmitter) emitImportedBinding(source string, specifier ast.ImportSpecifier) (string, error) {
	symbol := runtimeModuleImportBindingSymbol
	if specifier.Default {
		symbol = runtimeModuleImportDefaultSymbol
	}
	if specifier.Namespace {
		symbol = runtimeModuleImportNamespaceSymbol
	}
	if nativebinding.IsBindingModulePath(source) {
		symbol = runtimeNativeBindingWrapperSymbol
	}
	return emitter.emitRuntimeStringValueCall(symbol, source, importedName(specifier), importedLocalName(specifier))
}

func (emitter *StatementEmitter) emitModuleInitialize(source string) error {
	sourceValue, err := emitter.emitRuntimeString(source)
	if err != nil {
		return err
	}
	args := []RuntimeCallArg{{IRType: runtimeValueIRType, Value: sourceValue}}
	emitter.expressions.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeModuleInitializeSymbol, "void", args)})
	emitter.expressions.body = append(emitter.expressions.body, RuntimeVoidCall(runtimeModuleInitializeSymbol, args))
	return nil
}

func (emitter *StatementEmitter) emitModuleExportBinding(local string, exported string) error {
	if exported == "" {
		exported = local
	}
	if local == "" || exported == "" {
		return nil
	}
	return emitter.emitRuntimeStringVoidCall(runtimeModuleExportBindingSymbol, local, exported)
}

func (emitter *StatementEmitter) emitModuleExportValue(exported string, value string) error {
	exportedValue, err := emitter.emitRuntimeString(exported)
	if err != nil {
		return err
	}
	args := []RuntimeCallArg{
		{IRType: runtimeValueIRType, Value: exportedValue},
		{IRType: runtimeValueIRType, Value: value},
	}
	emitter.expressions.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeModuleExportValueSymbol, "void", args)})
	emitter.expressions.body = append(emitter.expressions.body, RuntimeVoidCall(runtimeModuleExportValueSymbol, args))
	return nil
}

func (emitter *StatementEmitter) emitModuleReexportAll(source string) error {
	if err := emitter.emitModuleInitialize(source); err != nil {
		return err
	}
	return emitter.emitRuntimeStringVoidCall(runtimeModuleReexportAllSymbol, source)
}

func (emitter *StatementEmitter) emitModuleReexportNamespace(source string, exported string) error {
	if err := emitter.emitModuleInitialize(source); err != nil {
		return err
	}
	return emitter.emitRuntimeStringVoidCall(runtimeModuleReexportNamespaceSymbol, source, exported)
}

func (emitter *StatementEmitter) emitModuleReexportBinding(source string, imported string, exported string) error {
	if err := emitter.emitModuleInitialize(source); err != nil {
		return err
	}
	return emitter.emitRuntimeStringVoidCall(runtimeModuleReexportBindingSymbol, source, imported, exported)
}

func (emitter *StatementEmitter) emitRuntimeStringValueCall(symbol string, values ...string) (string, error) {
	args, err := emitter.runtimeStringArgs(values...)
	if err != nil {
		return "", err
	}
	return emitter.expressions.emitRuntimeValueCall(symbol, args)
}

func (emitter *StatementEmitter) emitRuntimeStringVoidCall(symbol string, values ...string) error {
	args, err := emitter.runtimeStringArgs(values...)
	if err != nil {
		return err
	}
	emitter.expressions.addDeclarations([]Declaration{RuntimeCallDeclaration(symbol, "void", args)})
	emitter.expressions.body = append(emitter.expressions.body, RuntimeVoidCall(symbol, args))
	return nil
}

func (emitter *StatementEmitter) runtimeStringArgs(values ...string) ([]RuntimeCallArg, error) {
	args := make([]RuntimeCallArg, 0, len(values))
	for _, value := range values {
		lowered, err := emitter.emitRuntimeString(value)
		if err != nil {
			return nil, err
		}
		args = append(args, RuntimeCallArg{IRType: runtimeValueIRType, Value: lowered})
	}
	return args, nil
}

func (emitter *StatementEmitter) emitRuntimeString(value string) (string, error) {
	return emitter.expressions.EmitRuntimeLiteral(RuntimeLiteral{Kind: jayessruntime.StringValue, Text: value})
}

func importedName(specifier ast.ImportSpecifier) string {
	if specifier.Namespace {
		return "*"
	}
	if specifier.Default {
		return "default"
	}
	return specifier.Imported
}

func importedLocalName(specifier ast.ImportSpecifier) string {
	if specifier.Local != "" {
		return specifier.Local
	}
	return importedName(specifier)
}

func exportDeclarationLocalNames(statement ast.Statement) []string {
	switch stmt := statement.(type) {
	case *ast.VariableDecl:
		if stmt.Name != "" {
			return []string{stmt.Name}
		}
		return exportBindingPatternNames(stmt.Pattern)
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

func exportBindingPatternNames(pattern ast.BindingPattern) []string {
	switch binding := pattern.(type) {
	case *ast.BindingName:
		if binding.Name == "" {
			return nil
		}
		return []string{binding.Name}
	case *ast.BindingDefault:
		return exportBindingPatternNames(binding.Pattern)
	case *ast.BindingRest:
		return exportBindingPatternNames(binding.Pattern)
	case *ast.ArrayBindingPattern:
		var names []string
		for _, element := range binding.Elements {
			names = append(names, exportBindingPatternNames(element)...)
		}
		return names
	case *ast.ObjectBindingPattern:
		var names []string
		for _, property := range binding.Properties {
			names = append(names, exportBindingPatternNames(property.Pattern)...)
		}
		return names
	default:
		return nil
	}
}
