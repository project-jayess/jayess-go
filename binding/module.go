package binding

import (
	"fmt"
	"path/filepath"
	"strings"

	"jayess-go/ast"
)

const ModuleSuffix = ".bind.js"
const SourceSuffix = ".js"

type ModuleKind string

const (
	SourceModule        ModuleKind = "source"
	NativeBindingModule ModuleKind = "native-binding"
)

func ClassifyModulePath(path string) ModuleKind {
	if IsBindingModulePath(path) {
		return NativeBindingModule
	}
	return SourceModule
}

func IsBindingModulePath(path string) bool {
	return strings.HasSuffix(strings.ToLower(filepath.ToSlash(path)), ModuleSuffix)
}

func ValidateBindingTarget(path string) error {
	if strings.TrimSpace(path) != path || path == "" {
		return fmt.Errorf("binding target %q is malformed", path)
	}
	if strings.ContainsAny(path, "?#") {
		return fmt.Errorf("binding target %q must not include query strings or fragments", path)
	}
	if strings.Contains(path, "\\") {
		return fmt.Errorf("binding target %q must use / as the path separator", path)
	}
	if !IsBindingSourcePath(path) {
		return fmt.Errorf("binding target %q must use a %s source file", path, SourceSuffix)
	}
	return nil
}

func IsBindingSourcePath(path string) bool {
	return strings.HasSuffix(strings.ToLower(filepath.ToSlash(path)), SourceSuffix)
}

func ClassifyModule(path string, program *ast.Program) ModuleKind {
	if IsBindingProgram(program) {
		return NativeBindingModule
	}
	return ClassifyModulePath(path)
}

func IsBindingProgram(program *ast.Program) bool {
	return BindingExport(program).Found
}

type BindingExportMatch struct {
	Found      bool
	ImportName string
}

func BindingExport(program *ast.Program) BindingExportMatch {
	call := bindingExportCall(program)
	if call == nil {
		return BindingExportMatch{}
	}
	return BindingExportMatch{Found: true, ImportName: call.Callee}
}

func bindingExportCall(program *ast.Program) *ast.CallExpression {
	if program == nil {
		return nil
	}
	bindLocal := importedBindName(program)
	if bindLocal == "" {
		return nil
	}
	for _, statement := range program.Statements {
		exportDecl, ok := statement.(*ast.ExportDecl)
		if !ok || !exportDecl.Default {
			continue
		}
		call, ok := exportDecl.Value.(*ast.CallExpression)
		if ok && call.Callee == bindLocal {
			return call
		}
	}
	return nil
}

func importedBindName(program *ast.Program) string {
	for _, statement := range program.Statements {
		importDecl, ok := statement.(*ast.ImportDecl)
		if !ok || importDecl.Source != "ffi" {
			continue
		}
		for _, specifier := range importDecl.Specifiers {
			if specifier.Imported == "bind" && !specifier.Default && !specifier.Namespace {
				return specifier.Local
			}
		}
	}
	return ""
}
