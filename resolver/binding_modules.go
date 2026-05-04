package resolver

import (
	"fmt"
	"os"

	"jayess-go/ast"
	"jayess-go/binding"
	"jayess-go/lexer"
	"jayess-go/parser"
)

func ResolveBindingModules(fromPath string, program *ast.Program) ([]binding.Module, []binding.Diagnostic, error) {
	if program == nil {
		return nil, nil, nil
	}
	var modules []binding.Module
	var diagnostics []binding.Diagnostic
	for _, statement := range program.Statements {
		importDecl, ok := statement.(*ast.ImportDecl)
		if !ok {
			continue
		}
		path, err := ResolveImport(fromPath, importDecl.Source)
		if err != nil {
			return nil, diagnostics, fmt.Errorf("resolve binding import %q: %w", importDecl.Source, err)
		}
		if IsResolvedStdlibImport(path) {
			continue
		}
		importedProgram, err := parseResolvedProgram(path)
		if err != nil {
			return nil, diagnostics, fmt.Errorf("parse binding import %q: %w", importDecl.Source, err)
		}
		if !binding.IsBindingProgram(importedProgram) {
			continue
		}
		manifest, manifestDiagnostics := binding.ManifestFromProgram(importedProgram)
		diagnostics = append(diagnostics, manifestDiagnostics...)
		diagnostics = append(diagnostics, binding.ValidateManifest(manifest)...)
		diagnostics = append(diagnostics, binding.ValidateImportSpec(importSpecForImport(path, importDecl), manifest)...)
		modules = append(modules, binding.Module{Path: path, Manifest: manifest})
	}
	return modules, diagnostics, nil
}

func ResolveBindingBuildPlan(fromPath string, program *ast.Program, platform string, runtimeHeaderDir string) (binding.BuildPlan, error) {
	modules, diagnostics, err := ResolveBindingModules(fromPath, program)
	if err != nil {
		return binding.BuildPlan{}, err
	}
	plan := binding.PlanBuild(modules, platform, runtimeHeaderDir)
	plan.Diagnostics = append(diagnostics, plan.Diagnostics...)
	return plan, nil
}

func parseResolvedProgram(path string) (*ast.Program, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	program, err := parser.New(lexer.New(string(source))).ParseProgram()
	if err != nil {
		return nil, err
	}
	return program, nil
}

func importSpecForImport(path string, importDecl *ast.ImportDecl) binding.ImportSpec {
	if importDecl.SideEffect {
		return binding.ImportSpec{Source: path, Kind: binding.SideEffectImport}
	}
	spec := binding.ImportSpec{Source: path, Kind: binding.NamedImport}
	for _, imported := range importDecl.Specifiers {
		switch {
		case imported.Default:
			spec.Kind = binding.DefaultImport
		case imported.Namespace:
			spec.Kind = binding.NamespaceImport
		default:
			spec.Names = append(spec.Names, imported.Imported)
		}
	}
	return spec
}
