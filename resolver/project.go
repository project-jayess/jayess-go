package resolver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/parser"
)

type ProjectModule struct {
	Path         string
	Program      *ast.Program
	Dependencies []ResolvedModuleDependency
}

type Project struct {
	Entry       string
	Modules     []ProjectModule
	Graph       *ModuleGraph
	Diagnostics []ProjectDiagnostic
}

func LoadProject(entryPath string) (Project, error) {
	entry, err := normalizeProjectPath(entryPath)
	if err != nil {
		return Project{}, err
	}
	if _, err := os.Stat(entry); err != nil {
		return Project{}, fmt.Errorf("load project entry: %w", err)
	}
	loader := projectLoader{
		graph:   NewModuleGraph(),
		modules: map[string]ProjectModule{},
		queued:  map[string]bool{},
	}
	loader.enqueue(entry)
	loader.load()
	return Project{
		Entry:       entry,
		Modules:     loader.sortedModules(),
		Graph:       loader.graph,
		Diagnostics: loader.diagnostics,
	}, nil
}

type projectLoader struct {
	queue       []string
	queued      map[string]bool
	modules     map[string]ProjectModule
	graph       *ModuleGraph
	diagnostics []ProjectDiagnostic
}

func (l *projectLoader) load() {
	for len(l.queue) > 0 {
		path := l.queue[0]
		l.queue = l.queue[1:]
		if _, ok := l.modules[path]; ok {
			continue
		}
		module := l.loadModule(path)
		l.modules[path] = module
	}
}

func (l *projectLoader) loadModule(path string) ProjectModule {
	program, ok := l.parseModule(path)
	if !ok {
		l.graph.AddModule(path, nil)
		return ProjectModule{Path: path}
	}
	dependencies := l.resolveDependencies(path, program)
	l.graph.AddResolvedModule(path, dependencies)
	for _, dependency := range dependencies {
		l.enqueue(dependency.Path)
	}
	return ProjectModule{Path: path, Program: program, Dependencies: dependencies}
}

func (l *projectLoader) parseModule(path string) (*ast.Program, bool) {
	source, _, err := loadResolvedModuleSource(path)
	if err != nil {
		l.diagnostics = append(l.diagnostics, ProjectDiagnostic{Path: path, Message: "read module: " + err.Error()})
		return nil, false
	}
	program, err := parser.New(lexer.New(string(source))).ParseProgram()
	if err != nil {
		l.diagnostics = append(l.diagnostics, parseProjectDiagnostic(path, err))
		return nil, false
	}
	return program, true
}

func (l *projectLoader) resolveDependencies(path string, program *ast.Program) []ResolvedModuleDependency {
	importerPath, err := resolvedModuleSourcePath(path)
	if err != nil {
		l.diagnostics = append(l.diagnostics, ProjectDiagnostic{Path: path, Message: "resolve importer path: " + err.Error()})
		return nil
	}
	dependencies := ast.CompactModuleDependencies(ast.ModuleDependencies(program))
	resolved := make([]ResolvedModuleDependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		resolvedPath, err := ResolveImport(importerPath, dependency.Source)
		if err != nil {
			l.diagnostics = append(l.diagnostics, projectDiagnostic(
				path,
				dependencyPosition(program, dependency),
				fmt.Sprintf("resolve module dependency %q: %s", dependency.Source, err),
			))
			continue
		}
		resolved = append(resolved, ResolvedModuleDependency{
			Source:     dependency.Source,
			Path:       normalizeResolvedImport(resolvedPath),
			ReExport:   dependency.ReExport,
			SideEffect: dependency.SideEffect,
		})
	}
	return CompactResolvedModuleDependencies(resolved)
}

func (l *projectLoader) enqueue(path string) {
	if l.queued[path] {
		return
	}
	l.queued[path] = true
	l.queue = append(l.queue, path)
}

func (l *projectLoader) sortedModules() []ProjectModule {
	paths := make([]string, 0, len(l.modules))
	for path := range l.modules {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	modules := make([]ProjectModule, 0, len(paths))
	for _, path := range paths {
		modules = append(modules, l.modules[path])
	}
	return modules
}

func normalizeProjectPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("project path must not be empty")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

func normalizeResolvedImport(path string) string {
	if IsResolvedStdlibImport(path) {
		return path
	}
	normalized, err := normalizeProjectPath(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return normalized
}
