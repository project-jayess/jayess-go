package resolver

import (
	"fmt"
	"path/filepath"

	"jayess-go/ast"
	"jayess-go/binding"
)

func ResolveBindingModules(fromPath string, program *ast.Program) ([]binding.Module, []binding.Diagnostic, error) {
	if program == nil {
		return nil, nil, nil
	}
	collector := bindingModuleCollector{
		bindingModules: map[string]binding.Module{},
		visitedSources: map[string]struct{}{},
	}
	if err := collector.collectFromProgram(fromPath, program); err != nil {
		return nil, collector.diagnostics, err
	}
	return collector.modules(), collector.diagnostics, nil
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

type bindingModuleCollector struct {
	bindingModules map[string]binding.Module
	visitedSources map[string]struct{}
	diagnostics    []binding.Diagnostic
}

func (c *bindingModuleCollector) collectFromProgram(fromPath string, program *ast.Program) error {
	for _, statement := range program.Statements {
		importDecl, ok := statement.(*ast.ImportDecl)
		if !ok {
			continue
		}
		if err := c.collectImport(fromPath, importDecl); err != nil {
			return err
		}
	}
	return nil
}

func (c *bindingModuleCollector) collectImport(fromPath string, importDecl *ast.ImportDecl) error {
	path, err := ResolveImport(fromPath, importDecl.Source)
	if err != nil {
		return fmt.Errorf("resolve binding import %q: %w", importDecl.Source, err)
	}
	importedProgram, sourcePath, ok, err := parseBindingImportProgram(path)
	if err != nil {
		return fmt.Errorf("parse binding import %q: %w", importDecl.Source, err)
	}
	if !ok {
		return nil
	}
	if binding.IsBindingProgram(importedProgram) {
		manifest, manifestDiagnostics := binding.ManifestFromProgram(importedProgram)
		bindingPath := filepath.ToSlash(sourcePath)
		c.diagnostics = append(c.diagnostics, manifestDiagnostics...)
		c.diagnostics = append(c.diagnostics, binding.ValidateManifest(manifest)...)
		c.diagnostics = append(c.diagnostics, binding.ValidateImportSpec(importSpecForImport(bindingPath, importDecl), manifest)...)
		if _, exists := c.bindingModules[bindingPath]; !exists {
			c.bindingModules[bindingPath] = binding.Module{Path: bindingPath, Manifest: manifest}
		}
		return nil
	}
	if _, exists := c.visitedSources[sourcePath]; exists {
		return nil
	}
	c.visitedSources[sourcePath] = struct{}{}
	return c.collectFromProgram(sourcePath, importedProgram)
}

func (c *bindingModuleCollector) modules() []binding.Module {
	modules := make([]binding.Module, 0, len(c.bindingModules))
	for _, module := range c.bindingModules {
		modules = append(modules, module)
	}
	return modules
}

func parseResolvedProgram(path string) (*ast.Program, error) {
	source, _, err := loadResolvedModuleSource(path)
	if err != nil {
		return nil, err
	}
	return parseProgramSource(source)
}

func parseBindingImportProgram(path string) (*ast.Program, string, bool, error) {
	if IsResolvedStdlibImport(path) {
		sourcePath, ok, err := ResolvedStdlibSourcePath(path)
		if err != nil {
			return nil, "", false, err
		}
		if !ok {
			return nil, "", false, nil
		}
		program, err := parseResolvedProgram(path)
		return program, sourcePath, true, err
	}
	program, err := parseResolvedProgram(path)
	return program, path, true, err
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
