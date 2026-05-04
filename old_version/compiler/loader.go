package compiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"jayess-go/ast"
)

var (
	bareImportLinePattern            = regexp.MustCompile(`^\s*import\s+["']([^"']+)["']\s*;\s*$`)
	nativeImportLinePattern          = regexp.MustCompile(`^\s*native\s+import\s+["']([^"']+)["']\s*;\s*$`)
	namedImportLinePattern           = regexp.MustCompile(`^\s*import\s*\{\s*([^}]*)\s*\}\s*from\s+["']([^"']+)["']\s*;\s*$`)
	defaultImportLinePattern         = regexp.MustCompile(`^\s*import\s+([A-Za-z_][A-Za-z0-9_]*)\s*from\s+["']([^"']+)["']\s*;\s*$`)
	defaultAndNamedImportLinePattern = regexp.MustCompile(`^\s*import\s+([A-Za-z_][A-Za-z0-9_]*)\s*,\s*\{\s*([^}]*)\s*\}\s*from\s+["']([^"']+)["']\s*;\s*$`)
	namespaceImportLinePattern       = regexp.MustCompile(`^\s*import\s+\*\s+as\s+([A-Za-z_][A-Za-z0-9_]*)\s*from\s+["']([^"']+)["']\s*;\s*$`)

	exportFunctionLinePattern        = regexp.MustCompile(`^(\s*)export\s+function\b`)
	exportDefaultFunctionLinePattern = regexp.MustCompile(`^(\s*)export\s+default\s+function\b`)
	exportClassLinePattern           = regexp.MustCompile(`^(\s*)export\s+class\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
	exportDefaultClassLinePattern    = regexp.MustCompile(`^(\s*)export\s+default\s+class\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
	exportConstVarLinePattern        = regexp.MustCompile(`^\s*export\s+(const|var)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=`)
	exportDefaultExprLinePattern     = regexp.MustCompile(`^\s*export\s+default\s+(.+)\s*;\s*$`)
	exportListLinePattern            = regexp.MustCompile(`^\s*export\s*\{\s*([^}]*)\s*\}\s*;\s*$`)
	exportFromLinePattern            = regexp.MustCompile(`^\s*export\s*\{\s*([^}]*)\s*\}\s*from\s+["']([^"']+)["']\s*;\s*$`)
	exportStarFromLinePattern        = regexp.MustCompile(`^\s*export\s+\*\s+from\s+["']([^"']+)["']\s*;\s*$`)
	exportStarAsFromLinePattern      = regexp.MustCompile(`^\s*export\s+\*\s+as\s+([A-Za-z_][A-Za-z0-9_]*)\s*from\s+["']([^"']+)["']\s*;\s*$`)

	functionHeaderPattern = regexp.MustCompile(`^\s*function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)`)
	functionNamePattern   = regexp.MustCompile(`^\s*function\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
	classNamePattern      = regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
	globalDeclPattern     = regexp.MustCompile(`^\s*(const|var)\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
)

const defaultExportLocalName = "__jayess_default_export"

type loadedModule struct {
	exports       map[string]exportInfo
	namespaces    map[string]map[string]exportInfo
	defaultExport *exportInfo
	body          string
	declared      map[string]exportInfo
}

type exportInfo struct {
	kind       string
	paramCount int
	localName  string
	visibility string
}

type importSpecifier struct {
	exported string
	local    string
}

type packageJSON struct {
	Jayess string `json:"jayess"`
	Module string `json:"module"`
	Main   string `json:"main"`
	Native string `json:"native"`
}

type bindExportSpec struct {
	Symbol      string
	Type        string
	BorrowsArgs bool
}

type resolvedNativeImport struct {
	DisplayPath string
	Sources     []string
	IncludeDirs []string
	CFlags      []string
	LDFlags     []string
	PkgConfig   []string
	Exports     map[string]bindExportSpec
}

type loadedSourceTree struct {
	Source             string
	NativeImports      []string
	NativeIncludeDirs  []string
	NativeCompileFlags []string
	NativeLinkFlags    []string
	NativeSymbols      []*ast.ExternFunctionDecl
}

type LoaderDiagnosticError struct {
	File    string
	Line    int
	Column  int
	Message string
	Notes   []string
}

func (e *LoaderDiagnosticError) Error() string {
	if e == nil {
		return ""
	}
	location := e.File
	if e.Line > 0 {
		location = fmt.Sprintf("%s:%d", location, e.Line)
		if e.Column > 0 {
			location = fmt.Sprintf("%s:%d", location, e.Column)
		}
	}
	if location != "" {
		return fmt.Sprintf("%s: %s", location, e.Message)
	}
	return e.Message
}

func loadSourceTree(entryPath string, targetTriple string) (*loadedSourceTree, error) {
	modules := map[string]*loadedModule{}
	active := map[string]bool{}
	var parts []string
	nativeSet := map[string]bool{}
	var nativeImports []string
	nativeIncludeSet := map[string]bool{}
	var nativeIncludeDirs []string
	nativeCompileFlagSet := map[string]bool{}
	var nativeCompileFlags []string
	nativeLinkFlagSet := map[string]bool{}
	var nativeLinkFlags []string
	var nativeSymbols []*ast.ExternFunctionDecl

	absEntry, err := filepath.Abs(entryPath)
	if err != nil {
		return nil, fmt.Errorf("resolve entry path: %w", err)
	}

	if _, err := loadSourceFile(absEntry, targetTriple, modules, active, &parts, nativeSet, &nativeImports, nativeIncludeSet, &nativeIncludeDirs, nativeCompileFlagSet, &nativeCompileFlags, nativeLinkFlagSet, &nativeLinkFlags, &nativeSymbols, nil); err != nil {
		return nil, err
	}

	return &loadedSourceTree{
		Source:             strings.Join(parts, "\n\n"),
		NativeImports:      nativeImports,
		NativeIncludeDirs:  nativeIncludeDirs,
		NativeCompileFlags: nativeCompileFlags,
		NativeLinkFlags:    nativeLinkFlags,
		NativeSymbols:      nativeSymbols,
	}, nil
}

func loadSourceFile(path string, targetTriple string, modules map[string]*loadedModule, active map[string]bool, parts *[]string, nativeSet map[string]bool, nativeImports *[]string, nativeIncludeSet map[string]bool, nativeIncludeDirs *[]string, nativeCompileFlagSet map[string]bool, nativeCompileFlags *[]string, nativeLinkFlagSet map[string]bool, nativeLinkFlags *[]string, nativeSymbols *[]*ast.ExternFunctionDecl, stack []string) (*loadedModule, error) {
	if module, ok := modules[path]; ok {
		return module, nil
	}
	if active[path] {
		return nil, &LoaderDiagnosticError{File: path, Message: fmt.Sprintf("import cycle detected at %s", filepath.ToSlash(path))}
	}

	sourceBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read source %s: %w", path, err)
	}

	active[path] = true
	defer delete(active, path)
	module := &loadedModule{
		exports:    map[string]exportInfo{},
		namespaces: map[string]map[string]exportInfo{},
		declared:   map[string]exportInfo{},
	}

	var bodyLines []string
	braceDepth := 0
	namespaceImports := map[string]map[string]exportInfo{}
	importedLocals := map[string]string{}
	for index, line := range strings.Split(string(sourceBytes), "\n") {
		lineNumber := index + 1
		column := firstSignificantColumn(line)
		if braceDepth != 0 {
			bodyLines = append(bodyLines, applyNamespaceRewrites(line, namespaceImports))
			braceDepth = updateBraceDepth(braceDepth, line)
			continue
		}
		switch {
		case nativeImportLinePattern.MatchString(line):
			matches := nativeImportLinePattern.FindStringSubmatch(line)
			nativeImport, err := resolveBindImport(path, matches[1], targetTriple)
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			registerNativeImport(nativeImport, nativeSet, nativeImports, nativeIncludeSet, nativeIncludeDirs, nativeCompileFlagSet, nativeCompileFlags, nativeLinkFlagSet, nativeLinkFlags)

		case bareImportLinePattern.MatchString(line):
			matches := bareImportLinePattern.FindStringSubmatch(line)
			if _, ok, err := maybeResolveBindImport(path, matches[1], targetTriple); ok {
				if err != nil {
					return nil, wrapLoaderImportError(path, lineNumber, column, err)
				}
				return nil, loaderError(path, lineNumber, column, "native binding modules are not Jayess source modules; use named imports from *.bind.js")
			}
			importedPath, err := resolveImportPath(path, matches[1])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if active[importedPath] {
				return nil, loaderErrorWithNotes(path, lineNumber, column, "import cycle detected", []string{formatImportCycle(append(stack, path), importedPath)})
			}
			if _, err := loadSourceFile(importedPath, targetTriple, modules, active, parts, nativeSet, nativeImports, nativeIncludeSet, nativeIncludeDirs, nativeCompileFlagSet, nativeCompileFlags, nativeLinkFlagSet, nativeLinkFlags, nativeSymbols, append(stack, path)); err != nil {
				return nil, err
			}

		case defaultAndNamedImportLinePattern.MatchString(line):
			matches := defaultAndNamedImportLinePattern.FindStringSubmatch(line)
			if _, ok, err := maybeResolveBindImport(path, matches[3], targetTriple); ok {
				if err != nil {
					return nil, wrapLoaderImportError(path, lineNumber, column, err)
				}
				return nil, loaderError(path, lineNumber, column, "native binding modules are not Jayess source modules; use named imports from *.bind.js")
			}
			importedPath, err := resolveImportPath(path, matches[3])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if active[importedPath] {
				return nil, loaderErrorWithNotes(path, lineNumber, column, "import cycle detected", []string{formatImportCycle(append(stack, path), importedPath)})
			}
			importedModule, err := loadSourceFile(importedPath, targetTriple, modules, active, parts, nativeSet, nativeImports, nativeIncludeSet, nativeIncludeDirs, nativeCompileFlagSet, nativeCompileFlags, nativeLinkFlagSet, nativeLinkFlags, nativeSymbols, append(stack, path))
			if err != nil {
				return nil, err
			}
			aliasLine, err := buildDefaultImportDeclaration(matches[1], importedModule, matches[3])
			if err != nil {
				return nil, loaderError(path, lineNumber, column, err.Error())
			}
			if err := registerImportedLocal(importedLocals, matches[1], matches[3]); err != nil {
				return nil, loaderError(path, lineNumber, column, err.Error())
			}
			bodyLines = append(bodyLines, aliasLine)
			for _, spec := range parseImportedNames(matches[2]) {
				if err := registerImportedLocal(importedLocals, spec.local, matches[3]); err != nil {
					return nil, loaderError(path, lineNumber, column, err.Error())
				}
				if namespace, ok := importedModule.namespaces[spec.exported]; ok {
					namespaceImports[spec.local] = namespace
					continue
				}
				aliasLine, err := buildNamedImportDeclaration(spec, importedModule, matches[3])
				if err != nil {
					return nil, loaderError(path, lineNumber, column, err.Error())
				}
				if aliasLine != "" {
					bodyLines = append(bodyLines, aliasLine)
				}
			}

		case defaultImportLinePattern.MatchString(line):
			matches := defaultImportLinePattern.FindStringSubmatch(line)
			if _, ok, err := maybeResolveBindImport(path, matches[2], targetTriple); ok {
				if err != nil {
					return nil, wrapLoaderImportError(path, lineNumber, column, err)
				}
				return nil, loaderError(path, lineNumber, column, "native binding modules are not Jayess source modules; use named imports from *.bind.js")
			}
			importedPath, err := resolveImportPath(path, matches[2])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if active[importedPath] {
				return nil, loaderErrorWithNotes(path, lineNumber, column, "import cycle detected", []string{formatImportCycle(append(stack, path), importedPath)})
			}
			importedModule, err := loadSourceFile(importedPath, targetTriple, modules, active, parts, nativeSet, nativeImports, nativeIncludeSet, nativeIncludeDirs, nativeCompileFlagSet, nativeCompileFlags, nativeLinkFlagSet, nativeLinkFlags, nativeSymbols, append(stack, path))
			if err != nil {
				return nil, err
			}
			aliasLine, err := buildDefaultImportDeclaration(matches[1], importedModule, matches[2])
			if err != nil {
				return nil, loaderError(path, lineNumber, column, err.Error())
			}
			if err := registerImportedLocal(importedLocals, matches[1], matches[2]); err != nil {
				return nil, loaderError(path, lineNumber, column, err.Error())
			}
			bodyLines = append(bodyLines, aliasLine)

		case namespaceImportLinePattern.MatchString(line):
			matches := namespaceImportLinePattern.FindStringSubmatch(line)
			if _, ok, err := maybeResolveBindImport(path, matches[2], targetTriple); ok {
				if err != nil {
					return nil, wrapLoaderImportError(path, lineNumber, column, err)
				}
				return nil, loaderError(path, lineNumber, column, "native binding modules are not Jayess source modules; use named imports from *.bind.js")
			}
			importedPath, err := resolveImportPath(path, matches[2])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if active[importedPath] {
				return nil, loaderErrorWithNotes(path, lineNumber, column, "import cycle detected", []string{formatImportCycle(append(stack, path), importedPath)})
			}
			importedModule, err := loadSourceFile(importedPath, targetTriple, modules, active, parts, nativeSet, nativeImports, nativeIncludeSet, nativeIncludeDirs, nativeCompileFlagSet, nativeCompileFlags, nativeLinkFlagSet, nativeLinkFlags, nativeSymbols, append(stack, path))
			if err != nil {
				return nil, err
			}
			if err := registerImportedLocal(importedLocals, matches[1], matches[2]); err != nil {
				return nil, loaderError(path, lineNumber, column, err.Error())
			}
			namespaceImports[matches[1]] = exportedBindings(importedModule)

		case namedImportLinePattern.MatchString(line):
			matches := namedImportLinePattern.FindStringSubmatch(line)
			if nativeImport, ok, err := maybeResolveBindImport(path, matches[2], targetTriple); ok {
				if err != nil {
					return nil, wrapLoaderImportError(path, lineNumber, column, err)
				}
				registerNativeImport(nativeImport, nativeSet, nativeImports, nativeIncludeSet, nativeIncludeDirs, nativeCompileFlagSet, nativeCompileFlags, nativeLinkFlagSet, nativeLinkFlags)
				for _, spec := range parseImportedNames(matches[1]) {
					if err := registerImportedLocal(importedLocals, spec.local, matches[2]); err != nil {
						return nil, loaderError(path, lineNumber, column, err.Error())
					}
					nativeSpec := bindExportSpec{Symbol: spec.exported, Type: "function"}
					if len(nativeImport.Exports) > 0 {
						resolvedSpec, ok := nativeImport.Exports[spec.exported]
						if !ok {
							return nil, loaderErrorf(path, lineNumber, column, "native module %s does not export %s", filepath.ToSlash(matches[2]), spec.exported)
						}
						nativeSpec = resolvedSpec
					}
					switch nativeSpec.Type {
					case "", "function":
						*nativeSymbols = append(*nativeSymbols, &ast.ExternFunctionDecl{
							Name:         spec.local,
							NativeSymbol: nativeSpec.Symbol,
							BorrowsArgs:  nativeSpec.BorrowsArgs,
							Variadic:     true,
						})
					case "value":
						getterName := "__jayess_bind_value_" + spec.local
						*nativeSymbols = append(*nativeSymbols, &ast.ExternFunctionDecl{
							Name:         getterName,
							NativeSymbol: nativeSpec.Symbol,
						})
						bodyLines = append(bodyLines, fmt.Sprintf("const %s = %s();", spec.local, getterName))
					default:
						return nil, loaderErrorf(path, lineNumber, column, "native module %s export %s has unsupported type %s", filepath.ToSlash(matches[2]), spec.exported, nativeSpec.Type)
					}
				}
				continue
			}
			importedPath, err := resolveImportPath(path, matches[2])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if active[importedPath] {
				return nil, loaderErrorWithNotes(path, lineNumber, column, "import cycle detected", []string{formatImportCycle(append(stack, path), importedPath)})
			}
			importedModule, err := loadSourceFile(importedPath, targetTriple, modules, active, parts, nativeSet, nativeImports, nativeIncludeSet, nativeIncludeDirs, nativeCompileFlagSet, nativeCompileFlags, nativeLinkFlagSet, nativeLinkFlags, nativeSymbols, append(stack, path))
			if err != nil {
				return nil, err
			}
			for _, spec := range parseImportedNames(matches[1]) {
				if err := registerImportedLocal(importedLocals, spec.local, matches[2]); err != nil {
					return nil, loaderError(path, lineNumber, column, err.Error())
				}
				if namespace, ok := importedModule.namespaces[spec.exported]; ok {
					namespaceImports[spec.local] = namespace
					continue
				}
				aliasLine, err := buildNamedImportDeclaration(spec, importedModule, matches[2])
				if err != nil {
					return nil, loaderError(path, lineNumber, column, err.Error())
				}
				if aliasLine != "" {
					bodyLines = append(bodyLines, aliasLine)
				}
			}

		case exportDefaultFunctionLinePattern.MatchString(line):
			processed := exportDefaultFunctionLinePattern.ReplaceAllString(line, "${1}function")
			name := parseFunctionName(processed)
			if name == "" {
				return nil, loaderError(path, lineNumber, column, "default export function must be named")
			}
			info := exportInfo{kind: "function", paramCount: parseFunctionParamCount(processed), localName: name, visibility: "public"}
			module.defaultExport = &info
			module.declared[name] = info
			bodyLines = append(bodyLines, processed)

		case exportDefaultClassLinePattern.MatchString(line):
			matches := exportDefaultClassLinePattern.FindStringSubmatch(line)
			info := exportInfo{kind: "function", localName: matches[2], visibility: "public"}
			module.defaultExport = &info
			module.declared[matches[2]] = info
			bodyLines = append(bodyLines, strings.Replace(line, "export default ", "", 1))

		case exportClassLinePattern.MatchString(line):
			matches := exportClassLinePattern.FindStringSubmatch(line)
			info := exportInfo{kind: "function", localName: matches[2], visibility: "public"}
			if err := registerExport(module, matches[2], info); err != nil {
				return nil, err
			}
			module.declared[matches[2]] = info
			bodyLines = append(bodyLines, strings.Replace(line, "export ", "", 1))

		case exportFunctionLinePattern.MatchString(line):
			processed := exportFunctionLinePattern.ReplaceAllString(line, "${1}function")
			name := parseFunctionName(processed)
			if name == "" {
				return nil, loaderError(path, lineNumber, column, "exported function must be named")
			}
			info := exportInfo{kind: "function", paramCount: parseFunctionParamCount(processed), localName: name, visibility: "public"}
			if err := registerExport(module, name, info); err != nil {
				return nil, loaderError(path, lineNumber, column, err.Error())
			}
			module.declared[name] = info
			bodyLines = append(bodyLines, processed)

		case exportConstVarLinePattern.MatchString(line):
			matches := exportConstVarLinePattern.FindStringSubmatch(line)
			info := exportInfo{kind: matches[1], localName: matches[2], visibility: "public"}
			if err := registerExport(module, matches[2], info); err != nil {
				return nil, loaderError(path, lineNumber, column, err.Error())
			}
			module.declared[matches[2]] = info
			bodyLines = append(bodyLines, strings.Replace(line, "export ", "", 1))

		case exportListLinePattern.MatchString(line):
			matches := exportListLinePattern.FindStringSubmatch(line)
			for _, spec := range parseImportedNames(matches[1]) {
				info, ok := module.declared[spec.exported]
				if !ok {
					return nil, loaderErrorf(path, lineNumber, column, "cannot export unknown symbol %s", spec.exported)
				}
				if info.visibility == "private" {
					return nil, loaderErrorf(path, lineNumber, column, "cannot export private symbol %s", spec.exported)
				}
				if err := registerExport(module, spec.local, exportInfo{
					kind:       info.kind,
					paramCount: info.paramCount,
					localName:  info.localName,
					visibility: "public",
				}); err != nil {
					return nil, loaderError(path, lineNumber, column, err.Error())
				}
			}

		case exportFromLinePattern.MatchString(line):
			matches := exportFromLinePattern.FindStringSubmatch(line)
			importedPath, err := resolveImportPath(path, matches[2])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if active[importedPath] {
				return nil, loaderErrorWithNotes(path, lineNumber, column, "import cycle detected", []string{formatImportCycle(append(stack, path), importedPath)})
			}
			importedModule, err := loadSourceFile(importedPath, targetTriple, modules, active, parts, nativeSet, nativeImports, nativeIncludeSet, nativeIncludeDirs, nativeCompileFlagSet, nativeCompileFlags, nativeLinkFlagSet, nativeLinkFlags, nativeSymbols, append(stack, path))
			if err != nil {
				return nil, err
			}
			for _, spec := range parseImportedNames(matches[1]) {
				info, ok := importedModule.exports[spec.exported]
				if !ok {
					return nil, loaderErrorf(path, lineNumber, column, "module %s does not export %s", filepath.ToSlash(matches[2]), spec.exported)
				}
				if err := registerExport(module, spec.local, info); err != nil {
					return nil, loaderError(path, lineNumber, column, err.Error())
				}
			}

		case exportStarFromLinePattern.MatchString(line):
			matches := exportStarFromLinePattern.FindStringSubmatch(line)
			importedPath, err := resolveImportPath(path, matches[1])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if active[importedPath] {
				return nil, loaderErrorWithNotes(path, lineNumber, column, "import cycle detected", []string{formatImportCycle(append(stack, path), importedPath)})
			}
			importedModule, err := loadSourceFile(importedPath, targetTriple, modules, active, parts, nativeSet, nativeImports, nativeIncludeSet, nativeIncludeDirs, nativeCompileFlagSet, nativeCompileFlags, nativeLinkFlagSet, nativeLinkFlags, nativeSymbols, append(stack, path))
			if err != nil {
				return nil, err
			}
			for name, info := range importedModule.exports {
				if err := registerExport(module, name, info); err != nil {
					return nil, loaderError(path, lineNumber, column, err.Error())
				}
			}

		case exportStarAsFromLinePattern.MatchString(line):
			matches := exportStarAsFromLinePattern.FindStringSubmatch(line)
			importedPath, err := resolveImportPath(path, matches[2])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if active[importedPath] {
				return nil, loaderErrorWithNotes(path, lineNumber, column, "import cycle detected", []string{formatImportCycle(append(stack, path), importedPath)})
			}
			importedModule, err := loadSourceFile(importedPath, targetTriple, modules, active, parts, nativeSet, nativeImports, nativeIncludeSet, nativeIncludeDirs, nativeCompileFlagSet, nativeCompileFlags, nativeLinkFlagSet, nativeLinkFlags, nativeSymbols, append(stack, path))
			if err != nil {
				return nil, err
			}
			info := exportInfo{kind: "const", localName: matches[1], visibility: "public"}
			if err := registerExport(module, matches[1], info); err != nil {
				return nil, loaderError(path, lineNumber, column, err.Error())
			}
			module.namespaces[matches[1]] = exportedBindings(importedModule)

		case exportDefaultExprLinePattern.MatchString(line):
			matches := exportDefaultExprLinePattern.FindStringSubmatch(line)
			info := exportInfo{kind: "const", localName: defaultExportLocalName, visibility: "public"}
			module.defaultExport = &info
			module.declared[defaultExportLocalName] = info
			bodyLines = append(bodyLines, fmt.Sprintf("const %s = %s;", defaultExportLocalName, strings.TrimSpace(matches[1])))

		default:
			processedLine := applyNamespaceRewrites(line, namespaceImports)
			if info, name, ok := parseDeclaredSymbol(processedLine); ok {
				module.declared[name] = info
			}
			bodyLines = append(bodyLines, processedLine)
		}
		braceDepth = updateBraceDepth(braceDepth, line)
	}

	module.body = strings.Join(bodyLines, "\n")
	modules[path] = module
	*parts = append(*parts, module.body)
	return module, nil
}

func loaderError(file string, line int, column int, message string) error {
	return &LoaderDiagnosticError{
		File:    file,
		Line:    line,
		Column:  column,
		Message: message,
	}
}

func loaderErrorf(file string, line int, column int, format string, args ...any) error {
	return loaderError(file, line, column, fmt.Sprintf(format, args...))
}

func loaderErrorWithNotes(file string, line int, column int, message string, notes []string) error {
	return &LoaderDiagnosticError{
		File:    file,
		Line:    line,
		Column:  column,
		Message: message,
		Notes:   notes,
	}
}

func wrapLoaderImportError(file string, line int, column int, err error) error {
	var diagnostic *LoaderDiagnosticError
	if errors.As(err, &diagnostic) {
		return diagnostic
	}
	return loaderError(file, line, column, err.Error())
}

func firstSignificantColumn(line string) int {
	for index, r := range line {
		if r != ' ' && r != '\t' {
			return index + 1
		}
	}
	return 1
}

func formatImportCycle(stack []string, repeated string) string {
	start := 0
	for index, item := range stack {
		if item == repeated {
			start = index
			break
		}
	}
	cycle := append(append([]string{}, stack[start:]...), repeated)
	for index := range cycle {
		cycle[index] = filepath.ToSlash(cycle[index])
	}
	return "import cycle: " + strings.Join(cycle, " -> ")
}

func buildDefaultImportDeclaration(local string, module *loadedModule, importPath string) (string, error) {
	if module.defaultExport == nil {
		return "", fmt.Errorf("module %s does not provide a default export", filepath.ToSlash(importPath))
	}
	if local == module.defaultExport.localName {
		return "", nil
	}
	return buildAliasDeclaration(local, *module.defaultExport), nil
}

func buildNamedImportDeclaration(spec importSpecifier, module *loadedModule, importPath string) (string, error) {
	info, ok := module.exports[spec.exported]
	if !ok {
		return "", fmt.Errorf("module %s does not export %s", filepath.ToSlash(importPath), spec.exported)
	}
	if spec.local == info.localName {
		return "", nil
	}
	return buildAliasDeclaration(spec.local, info), nil
}

func parseImportedNames(raw string) []importSpecifier {
	var names []importSpecifier
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		spec := importSpecifier{exported: part, local: part}
		if strings.Contains(part, " as ") {
			pieces := strings.SplitN(part, " as ", 2)
			spec.exported = strings.TrimSpace(pieces[0])
			spec.local = strings.TrimSpace(pieces[1])
		}
		names = append(names, spec)
	}
	return names
}

func parseDeclaredSymbol(line string) (exportInfo, string, bool) {
	trimmed := strings.TrimSpace(line)
	if name := parseFunctionName(trimmed); name != "" {
		return exportInfo{kind: "function", paramCount: parseFunctionParamCount(trimmed), localName: name}, name, true
	}
	if matches := classNamePattern.FindStringSubmatch(trimmed); matches != nil {
		return exportInfo{kind: "function", localName: matches[1]}, matches[1], true
	}
	matches := globalDeclPattern.FindStringSubmatch(trimmed)
	if matches == nil {
		return exportInfo{}, "", false
	}
	return exportInfo{kind: matches[1], localName: matches[2]}, matches[2], true
}

func parseFunctionName(line string) string {
	matches := functionNamePattern.FindStringSubmatch(line)
	if matches == nil {
		return ""
	}
	return matches[1]
}

func parseFunctionParamCount(line string) int {
	matches := functionHeaderPattern.FindStringSubmatch(line)
	if matches == nil {
		return 0
	}
	params := strings.TrimSpace(matches[2])
	if params == "" {
		return 0
	}
	count := 0
	for _, part := range strings.Split(params, ",") {
		if strings.TrimSpace(part) != "" {
			count++
		}
	}
	return count
}

func buildAliasDeclaration(local string, info exportInfo) string {
	switch info.kind {
	case "function":
		var params []string
		var args []string
		for i := 0; i < info.paramCount; i++ {
			name := fmt.Sprintf("__arg%d", i)
			params = append(params, name)
			args = append(args, name)
		}
		return fmt.Sprintf("function %s(%s) { return %s(%s); }", local, strings.Join(params, ", "), info.localName, strings.Join(args, ", "))
	case "var":
		return fmt.Sprintf("var %s = %s;", local, info.localName)
	default:
		return fmt.Sprintf("const %s = %s;", local, info.localName)
	}
}

func registerImportedLocal(imports map[string]string, local string, importPath string) error {
	if existing, ok := imports[local]; ok {
		return fmt.Errorf("duplicate import binding %s from %s; already imported from %s", local, filepath.ToSlash(importPath), filepath.ToSlash(existing))
	}
	imports[local] = importPath
	return nil
}

func registerExport(module *loadedModule, name string, info exportInfo) error {
	if _, exists := module.exports[name]; exists {
		return fmt.Errorf("duplicate export %s", name)
	}
	module.exports[name] = info
	return nil
}

func resolveImportPath(fromPath, importPath string) (string, error) {
	if strings.HasPrefix(importPath, ".") {
		resolved := filepath.Join(filepath.Dir(fromPath), filepath.FromSlash(importPath))
		return resolveSourceFile(resolved)
	}
	return resolvePackageImport(filepath.Dir(fromPath), importPath)
}

func registerNativeImport(nativeImport resolvedNativeImport, nativeSet map[string]bool, nativeImports *[]string, nativeIncludeSet map[string]bool, nativeIncludeDirs *[]string, nativeCompileFlagSet map[string]bool, nativeCompileFlags *[]string, nativeLinkFlagSet map[string]bool, nativeLinkFlags *[]string) {
	for _, source := range nativeImport.Sources {
		if !nativeSet[source] {
			nativeSet[source] = true
			*nativeImports = append(*nativeImports, source)
		}
	}
	for _, includeDir := range nativeImport.IncludeDirs {
		if !nativeIncludeSet[includeDir] {
			nativeIncludeSet[includeDir] = true
			*nativeIncludeDirs = append(*nativeIncludeDirs, includeDir)
		}
	}
	for _, flag := range nativeImport.CFlags {
		if !nativeCompileFlagSet[flag] {
			nativeCompileFlagSet[flag] = true
			*nativeCompileFlags = append(*nativeCompileFlags, flag)
		}
	}
	for _, flag := range nativeImport.LDFlags {
		if !nativeLinkFlagSet[flag] {
			nativeLinkFlagSet[flag] = true
			*nativeLinkFlags = append(*nativeLinkFlags, flag)
		}
	}
}

func resolveBindImport(fromPath, importPath string, targetTriple string) (resolvedNativeImport, error) {
	absPath, err := resolveBindFilePath(filepath.Dir(fromPath), importPath)
	if err != nil {
		return resolvedNativeImport{}, err
	}
	return loadBindFile(absPath, importPath, targetTriple)
}

func resolveBindFilePath(startDir, importPath string) (string, error) {
	if strings.HasPrefix(importPath, ".") {
		return resolveConcreteBindFile(filepath.Join(startDir, filepath.FromSlash(importPath)), importPath)
	}
	return resolvePackageBindImport(startDir, importPath)
}

func maybeResolveBindImport(fromPath, importPath string, targetTriple string) (resolvedNativeImport, bool, error) {
	if isBindImportSpec(importPath) {
		nativeImport, err := resolveBindImport(fromPath, importPath, targetTriple)
		return nativeImport, true, err
	}
	if !strings.HasPrefix(importPath, ".") {
		nativeImport, err := resolveBindImport(fromPath, importPath, targetTriple)
		if err == nil {
			return nativeImport, true, nil
		}
	}
	return resolvedNativeImport{}, false, nil
}

func isBindImportSpec(importPath string) bool {
	return strings.HasSuffix(strings.ToLower(importPath), ".bind.js")
}

func loadBindFile(absPath string, importPath string, targetTriple string) (resolvedNativeImport, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return resolvedNativeImport{}, fmt.Errorf("read binding file %q: %w", filepath.ToSlash(importPath), err)
	}
	text := string(data)
	baseDir := filepath.Dir(absPath)
	spec, err := parseBindModuleSpec(text)
	if err != nil {
		return resolvedNativeImport{}, fmt.Errorf("binding file %q: %w", filepath.ToSlash(importPath), err)
	}

	sources := append([]string{}, spec.Sources...)
	if len(sources) == 0 {
		return resolvedNativeImport{}, fmt.Errorf("binding file %q must declare at least one source file", filepath.ToSlash(importPath))
	}
	includeDirs := append([]string{}, spec.IncludeDirs...)
	cflags := append([]string{}, spec.CFlags...)
	ldflags := append([]string{}, spec.LDFlags...)
	pkgConfig := append([]string{}, spec.PkgConfig...)
	exports := spec.Exports
	if len(exports) == 0 {
		return resolvedNativeImport{}, fmt.Errorf("binding file %q must declare at least one export", filepath.ToSlash(importPath))
	}
	if platformKey := bindPlatformKey(targetTriple); platformKey != "" {
		if platformSpec, ok := spec.Platforms[platformKey]; ok {
			sources = append(sources, platformSpec.Sources...)
			includeDirs = append(includeDirs, platformSpec.IncludeDirs...)
			cflags = append(cflags, platformSpec.CFlags...)
			ldflags = append(ldflags, platformSpec.LDFlags...)
			pkgConfig = append(pkgConfig, platformSpec.PkgConfig...)
		}
	}
	if len(pkgConfig) != 0 {
		pkgCFlags, pkgLDFlags, err := resolvePkgConfigFlags(pkgConfig)
		if err != nil {
			return resolvedNativeImport{}, fmt.Errorf("binding file %q: %w", filepath.ToSlash(importPath), err)
		}
		cflags = append(cflags, pkgCFlags...)
		ldflags = append(ldflags, pkgLDFlags...)
	}

	result := resolvedNativeImport{
		DisplayPath: filepath.ToSlash(importPath),
		Exports:     map[string]bindExportSpec{},
		CFlags:      append([]string{}, cflags...),
		LDFlags:     append([]string{}, ldflags...),
		PkgConfig:   append([]string{}, pkgConfig...),
	}
	for _, source := range sources {
		resolvedSource, err := resolveNativeManifestSource(baseDir, source)
		if err != nil {
			return resolvedNativeImport{}, fmt.Errorf("binding file %q: %w", filepath.ToSlash(importPath), err)
		}
		result.Sources = append(result.Sources, resolvedSource)
	}
	for _, includeDir := range includeDirs {
		resolvedIncludeDir, err := resolveNativeManifestDir(baseDir, includeDir)
		if err != nil {
			return resolvedNativeImport{}, fmt.Errorf("binding file %q: %w", filepath.ToSlash(importPath), err)
		}
		result.IncludeDirs = append(result.IncludeDirs, resolvedIncludeDir)
	}
	for exportName, spec := range exports {
		if spec.Symbol == "" {
			return resolvedNativeImport{}, fmt.Errorf("binding file %q export %q must declare a symbol", filepath.ToSlash(importPath), exportName)
		}
		switch spec.Type {
		case "", "function", "value":
		default:
			return resolvedNativeImport{}, fmt.Errorf("binding file %q export %q has unsupported type %q", filepath.ToSlash(importPath), exportName, spec.Type)
		}
		if spec.Type == "" {
			spec.Type = "function"
		}
		result.Exports[exportName] = spec
	}
	return result, nil
}

func resolvePkgConfigFlags(packages []string) ([]string, []string, error) {
	if len(packages) == 0 {
		return nil, nil, nil
	}
	pkgConfigPath, err := exec.LookPath("pkg-config")
	if err != nil {
		return nil, nil, fmt.Errorf("pkg-config was not found in PATH")
	}
	cflagsCmd := exec.Command(pkgConfigPath, append([]string{"--cflags"}, packages...)...)
	cflagsOut, err := cflagsCmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("pkg-config cflags lookup failed for %s: %s", strings.Join(packages, ", "), strings.TrimSpace(string(cflagsOut)))
	}
	libsCmd := exec.Command(pkgConfigPath, append([]string{"--libs"}, packages...)...)
	libsOut, err := libsCmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("pkg-config libs lookup failed for %s: %s", strings.Join(packages, ", "), strings.TrimSpace(string(libsOut)))
	}
	return strings.Fields(strings.TrimSpace(string(cflagsOut))), strings.Fields(strings.TrimSpace(string(libsOut))), nil
}

type bindModuleSpec struct {
	Sources     []string
	IncludeDirs []string
	CFlags      []string
	LDFlags     []string
	PkgConfig   []string
	Exports     map[string]bindExportSpec
	Platforms   map[string]bindPlatformSpec
}

type bindPlatformSpec struct {
	Sources     []string
	IncludeDirs []string
	CFlags      []string
	LDFlags     []string
	PkgConfig   []string
}

func parseBindModuleSpec(text string) (bindModuleSpec, error) {
	content, err := parseBindDefaultExportObject(text)
	if err != nil {
		return bindModuleSpec{}, err
	}
	fields, err := parseTopLevelObjectFields(content)
	if err != nil {
		return bindModuleSpec{}, err
	}
	spec := bindModuleSpec{Exports: map[string]bindExportSpec{}, Platforms: map[string]bindPlatformSpec{}}
	if spec.Sources, err = parseBindArrayField(fields, "sources"); err != nil {
		return bindModuleSpec{}, err
	}
	if spec.IncludeDirs, err = parseBindArrayField(fields, "includeDirs"); err != nil {
		return bindModuleSpec{}, err
	}
	if spec.CFlags, err = parseBindArrayField(fields, "cflags"); err != nil {
		return bindModuleSpec{}, err
	}
	if spec.LDFlags, err = parseBindArrayField(fields, "ldflags"); err != nil {
		return bindModuleSpec{}, err
	}
	if spec.PkgConfig, err = parseBindArrayField(fields, "pkgConfig"); err != nil {
		return bindModuleSpec{}, err
	}
	if spec.Exports, err = parseBindExports(fields["exports"]); err != nil {
		return bindModuleSpec{}, err
	}
	if spec.Platforms, err = parseBindPlatforms(fields["platforms"]); err != nil {
		return bindModuleSpec{}, err
	}
	return spec, nil
}

func parseBindDefaultExportObject(text string) (string, error) {
	start := strings.Index(text, "export default")
	if start < 0 {
		return "", fmt.Errorf(`binding file must declare "export default { ... }"`)
	}
	open := strings.Index(text[start:], "{")
	if open < 0 {
		return "", fmt.Errorf(`binding file must declare "export default { ... }"`)
	}
	open += start
	close := findMatchingDelimiter(text, open, '{', '}')
	if close < 0 {
		return "", fmt.Errorf(`binding file export default object has an unterminated object literal`)
	}
	return text[open+1 : close], nil
}

func parseBindArrayField(fields map[string]string, field string) ([]string, error) {
	raw, ok := fields[field]
	if !ok {
		return nil, nil
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if raw[0] != '[' {
		return nil, fmt.Errorf("field %q must be an array literal", field)
	}
	close := findMatchingDelimiter(raw, 0, '[', ']')
	if close != len(raw)-1 {
		return nil, fmt.Errorf("field %q must be an array literal", field)
	}
	content := raw[1:close]
	var values []string
	for _, match := range regexp.MustCompile(`["']([^"']+)["']`).FindAllStringSubmatch(content, -1) {
		values = append(values, strings.TrimSpace(match[1]))
	}
	return values, nil
}

func parseBindExports(raw string) (map[string]bindExportSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if raw[0] != '{' {
		return nil, fmt.Errorf(`field "exports" must be an object literal`)
	}
	close := findMatchingDelimiter(raw, 0, '{', '}')
	if close != len(raw)-1 {
		return nil, fmt.Errorf(`field "exports" has an unterminated object literal`)
	}
	content := raw[1:close]
	exports := map[string]bindExportSpec{}
	index := 0
	for index < len(content) {
		if whitespace := strings.TrimLeft(content[index:], " \t\r\n,"); len(whitespace) != len(content[index:]) {
			index = len(content) - len(whitespace)
		}
		if index >= len(content) {
			break
		}
		colonOffset := strings.Index(content[index:], ":")
		if colonOffset < 0 {
			break
		}
		colon := index + colonOffset
		name := strings.TrimSpace(content[index:colon])
		name = strings.Trim(name, `"'`)
		if name == "" {
			return nil, fmt.Errorf(`field "exports" contains an empty export name`)
		}
		valueStart := colon + 1
		for valueStart < len(content) && (content[valueStart] == ' ' || content[valueStart] == '\t' || content[valueStart] == '\r' || content[valueStart] == '\n') {
			valueStart++
		}
		if valueStart >= len(content) || content[valueStart] != '{' {
			return nil, fmt.Errorf(`export %q must be an object literal`, name)
		}
		valueEnd := findMatchingDelimiter(content, valueStart, '{', '}')
		if valueEnd < 0 {
			return nil, fmt.Errorf(`export %q has an unterminated object literal`, name)
		}
		spec, err := parseBindExportSpec(content[valueStart+1 : valueEnd])
		if err != nil {
			return nil, fmt.Errorf("export %q: %w", name, err)
		}
		exports[name] = spec
		index = valueEnd + 1
	}
	return exports, nil
}

func parseBindPlatforms(raw string) (map[string]bindPlatformSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]bindPlatformSpec{}, nil
	}
	if raw[0] != '{' {
		return nil, fmt.Errorf(`field "platforms" must be an object literal`)
	}
	close := findMatchingDelimiter(raw, 0, '{', '}')
	if close != len(raw)-1 {
		return nil, fmt.Errorf(`field "platforms" has an unterminated object literal`)
	}
	content := raw[1:close]
	fields, err := parseTopLevelObjectFields(content)
	if err != nil {
		return nil, fmt.Errorf(`field "platforms": %w`, err)
	}
	platforms := map[string]bindPlatformSpec{}
	for platformName, platformRaw := range fields {
		platformRaw = strings.TrimSpace(platformRaw)
		if platformRaw == "" {
			continue
		}
		if platformRaw[0] != '{' {
			return nil, fmt.Errorf(`platform %q must be an object literal`, platformName)
		}
		platformClose := findMatchingDelimiter(platformRaw, 0, '{', '}')
		if platformClose != len(platformRaw)-1 {
			return nil, fmt.Errorf(`platform %q has an unterminated object literal`, platformName)
		}
		platformFields, err := parseTopLevelObjectFields(platformRaw[1:platformClose])
		if err != nil {
			return nil, fmt.Errorf("platform %q: %w", platformName, err)
		}
		spec := bindPlatformSpec{}
		if spec.Sources, err = parseBindArrayField(platformFields, "sources"); err != nil {
			return nil, fmt.Errorf("platform %q: %w", platformName, err)
		}
		if spec.IncludeDirs, err = parseBindArrayField(platformFields, "includeDirs"); err != nil {
			return nil, fmt.Errorf("platform %q: %w", platformName, err)
		}
		if spec.CFlags, err = parseBindArrayField(platformFields, "cflags"); err != nil {
			return nil, fmt.Errorf("platform %q: %w", platformName, err)
		}
		if spec.LDFlags, err = parseBindArrayField(platformFields, "ldflags"); err != nil {
			return nil, fmt.Errorf("platform %q: %w", platformName, err)
		}
		if spec.PkgConfig, err = parseBindArrayField(platformFields, "pkgConfig"); err != nil {
			return nil, fmt.Errorf("platform %q: %w", platformName, err)
		}
		platforms[platformName] = spec
	}
	return platforms, nil
}

func parseTopLevelObjectFields(content string) (map[string]string, error) {
	fields := map[string]string{}
	for index := 0; index < len(content); {
		for index < len(content) {
			switch content[index] {
			case ' ', '\t', '\r', '\n', ',':
				index++
			default:
				goto fieldStart
			}
		}
		break
	fieldStart:
		start := index
		if content[index] == '"' || content[index] == '\'' {
			quote := content[index]
			index++
			for index < len(content) && content[index] != quote {
				if content[index] == '\\' && index+1 < len(content) {
					index += 2
					continue
				}
				index++
			}
			if index >= len(content) {
				return nil, fmt.Errorf("unterminated quoted field name")
			}
			index++
		} else {
			for index < len(content) && isBindFieldNameChar(content[index]) {
				index++
			}
		}
		name := strings.TrimSpace(content[start:index])
		name = strings.Trim(name, `"'`)
		if name == "" {
			return nil, fmt.Errorf("empty field name")
		}
		for index < len(content) && (content[index] == ' ' || content[index] == '\t' || content[index] == '\r' || content[index] == '\n') {
			index++
		}
		if index >= len(content) || content[index] != ':' {
			return nil, fmt.Errorf("field %q is missing ':'", name)
		}
		index++
		for index < len(content) && (content[index] == ' ' || content[index] == '\t' || content[index] == '\r' || content[index] == '\n') {
			index++
		}
		valueStart := index
		valueEnd, err := findTopLevelValueEnd(content, valueStart)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", name, err)
		}
		fields[name] = strings.TrimSpace(content[valueStart:valueEnd])
		index = valueEnd
	}
	return fields, nil
}

func findTopLevelValueEnd(content string, start int) (int, error) {
	depthBrace := 0
	depthBracket := 0
	depthParen := 0
	var quote byte
	for index := start; index < len(content); index++ {
		ch := content[index]
		if quote != 0 {
			if ch == '\\' && index+1 < len(content) {
				index++
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '"', '\'':
			quote = ch
		case '{':
			depthBrace++
		case '}':
			if depthBrace == 0 && depthBracket == 0 && depthParen == 0 {
				return index, nil
			}
			depthBrace--
		case '[':
			depthBracket++
		case ']':
			depthBracket--
		case '(':
			depthParen++
		case ')':
			depthParen--
		case ',':
			if depthBrace == 0 && depthBracket == 0 && depthParen == 0 {
				return index, nil
			}
		}
		if depthBrace < 0 || depthBracket < 0 || depthParen < 0 {
			return 0, fmt.Errorf("unexpected closing delimiter")
		}
	}
	if quote != 0 || depthBrace != 0 || depthBracket != 0 || depthParen != 0 {
		return 0, fmt.Errorf("unterminated value")
	}
	return len(content), nil
}

func isBindFieldNameChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '$'
}

func bindPlatformKey(targetTriple string) string {
	lowered := strings.ToLower(targetTriple)
	switch {
	case strings.Contains(lowered, "windows"), strings.Contains(lowered, "win32"), strings.Contains(lowered, "mingw"), strings.Contains(lowered, "msvc"):
		return "windows"
	case strings.Contains(lowered, "darwin"), strings.Contains(lowered, "apple"), strings.Contains(lowered, "macos"):
		return "darwin"
	case strings.Contains(lowered, "linux"), strings.Contains(lowered, "gnu"), strings.Contains(lowered, "musl"):
		return "linux"
	default:
		return ""
	}
}

func parseBindExportSpec(text string) (bindExportSpec, error) {
	spec := bindExportSpec{}
	fields, err := parseTopLevelObjectFields(text)
	if err != nil {
		return bindExportSpec{}, err
	}
	for _, field := range []string{"symbol", "type"} {
		raw, ok := fields[field]
		if !ok {
			continue
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return bindExportSpec{}, fmt.Errorf("field %q is missing a value", field)
		}
		if raw[0] != '"' && raw[0] != '\'' {
			return bindExportSpec{}, fmt.Errorf("field %q must be a string literal", field)
		}
		quote := raw[0]
		valueEnd := 1
		for valueEnd < len(raw) && raw[valueEnd] != quote {
			valueEnd++
		}
		if valueEnd >= len(raw) {
			return bindExportSpec{}, fmt.Errorf("field %q has an unterminated string literal", field)
		}
		value := raw[1:valueEnd]
		switch field {
		case "symbol":
			spec.Symbol = strings.TrimSpace(value)
		case "type":
			spec.Type = strings.TrimSpace(value)
		}
	}
	if raw, ok := fields["borrowsArgs"]; ok {
		switch strings.TrimSpace(raw) {
		case "true":
			spec.BorrowsArgs = true
		case "false", "":
			spec.BorrowsArgs = false
		default:
			return bindExportSpec{}, fmt.Errorf(`field "borrowsArgs" must be true or false`)
		}
	}
	return spec, nil
}

func findMatchingDelimiter(text string, start int, open byte, close byte) int {
	depth := 0
	for index := start; index < len(text); index++ {
		switch text[index] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func resolveNativeManifestSource(baseDir string, source string) (string, error) {
	resolvedPath := source
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Join(baseDir, filepath.FromSlash(source))
	}
	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("resolve native source %q: %w", filepath.ToSlash(source), err)
	}
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("native source %q was not found", filepath.ToSlash(source))
	}
	switch strings.ToLower(filepath.Ext(absPath)) {
	case ".c", ".cc", ".cpp", ".cxx":
		return absPath, nil
	default:
		return "", fmt.Errorf("native source %q must point to a .c/.cc/.cpp/.cxx file", filepath.ToSlash(source))
	}
}

func resolveNativeManifestDir(baseDir string, dir string) (string, error) {
	resolvedPath := dir
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Join(baseDir, filepath.FromSlash(dir))
	}
	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("resolve include directory %q: %w", filepath.ToSlash(dir), err)
	}
	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("include directory %q was not found", filepath.ToSlash(dir))
	}
	return absPath, nil
}

func resolvePackageBindImport(startDir, importPath string) (string, error) {
	packageName, subpath := splitPackageImport(importPath)
	for dir := startDir; ; dir = filepath.Dir(dir) {
		candidateBase := filepath.Join(dir, "node_modules", filepath.FromSlash(packageName))
		if info, err := os.Stat(candidateBase); err == nil && info.IsDir() {
			return resolvePackageBindEntry(candidateBase, subpath, importPath)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("package %q was not found in node_modules; run npm install or check package.json dependencies", importPath)
}

func resolvePackageBindEntry(packageDir, subpath, importPath string) (string, error) {
	if subpath != "" {
		return resolveConcreteBindFile(filepath.Join(packageDir, filepath.FromSlash(subpath)), importPath)
	}

	for _, candidate := range []string{
		filepath.Join(packageDir, "index.bind.js"),
	} {
		if resolved, err := resolveConcreteBindFile(candidate, importPath); err == nil {
			return resolved, nil
		}
	}

	return "", fmt.Errorf("package %q does not expose a supported binding entrypoint via *.bind.js", importPath)
}

func resolveConcreteBindFile(path string, importPath string) (string, error) {
	candidates := []string{path}
	if filepath.Ext(path) == "" {
		candidates = append(candidates,
			path+".bind.js",
			filepath.Join(path, "index.bind.js"),
		)
	}
	for _, candidate := range candidates {
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		info, err := os.Stat(absPath)
		if err != nil || info.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(absPath)) {
		case ".js":
			if strings.HasSuffix(strings.ToLower(absPath), ".bind.js") {
				return absPath, nil
			}
		}
	}
	return "", fmt.Errorf("native import %q must point to a .bind.js file", filepath.ToSlash(importPath))
}

func resolvePackageImport(startDir, importPath string) (string, error) {
	packageName, subpath := splitPackageImport(importPath)
	for dir := startDir; ; dir = filepath.Dir(dir) {
		candidateBase := filepath.Join(dir, "node_modules", filepath.FromSlash(packageName))
		if info, err := os.Stat(candidateBase); err == nil && info.IsDir() {
			return resolvePackageEntry(candidateBase, subpath, importPath)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("package %q was not found in node_modules; run npm install or check package.json dependencies", importPath)
}

func resolvePackageEntry(packageDir, subpath, importPath string) (string, error) {
	if subpath != "" {
		return resolveSourceFile(filepath.Join(packageDir, filepath.FromSlash(subpath)))
	}

	packageJSONPath := filepath.Join(packageDir, "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err == nil {
		var pkg packageJSON
		if err := json.Unmarshal(data, &pkg); err != nil {
			return "", fmt.Errorf("package %q has an invalid package.json: %w", importPath, err)
		}
		var firstEntryErr error
		for _, entry := range []string{pkg.Jayess, pkg.Module, pkg.Main} {
			if strings.TrimSpace(entry) == "" {
				continue
			}
			if filepath.Ext(entry) != "" && strings.ToLower(filepath.Ext(entry)) != ".js" {
				return "", fmt.Errorf("package %q entry %q is not a supported Jayess .js module", importPath, filepath.ToSlash(entry))
			}
			resolved, err := resolveSourceFile(filepath.Join(packageDir, filepath.FromSlash(entry)))
			if err == nil {
				return resolved, nil
			}
			if firstEntryErr == nil {
				firstEntryErr = fmt.Errorf("package %q entry %q could not be resolved: %w", importPath, filepath.ToSlash(entry), err)
			}
		}
		if firstEntryErr != nil {
			return "", firstEntryErr
		}
	}

	resolved, err := resolveSourceFile(filepath.Join(packageDir, "index.js"))
	if err == nil {
		return resolved, nil
	}
	return "", fmt.Errorf("package %q does not expose a supported Jayess .js entrypoint via jayess/module/main or index.js", importPath)
}

func resolveSourceFile(path string) (string, error) {
	candidates := []string{path}
	if filepath.Ext(path) == "" {
		candidates = append(candidates, path+".js", filepath.Join(path, "index.js"))
	}
	for _, candidate := range candidates {
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		info, err := os.Stat(absPath)
		if err == nil && !info.IsDir() {
			return absPath, nil
		}
	}
	return "", fmt.Errorf("source file %q was not found", filepath.ToSlash(path))
}

func splitPackageImport(spec string) (string, string) {
	if strings.HasPrefix(spec, "@") {
		parts := strings.Split(spec, "/")
		if len(parts) <= 2 {
			return spec, ""
		}
		return strings.Join(parts[:2], "/"), strings.Join(parts[2:], "/")
	}
	parts := strings.Split(spec, "/")
	if len(parts) == 1 {
		return spec, ""
	}
	return parts[0], strings.Join(parts[1:], "/")
}

func updateBraceDepth(depth int, line string) int {
	for _, r := range line {
		switch r {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		}
	}
	return depth
}

func exportedBindings(module *loadedModule) map[string]exportInfo {
	result := map[string]exportInfo{}
	for name, info := range module.exports {
		result[name] = info
	}
	for namespace, exports := range module.namespaces {
		for exportName, info := range exports {
			result[namespace+"."+exportName] = info
		}
	}
	if module.defaultExport != nil {
		result["default"] = *module.defaultExport
	}
	return result
}

func applyNamespaceRewrites(line string, bindings map[string]map[string]exportInfo) string {
	rewritten := line
	for namespace, exports := range bindings {
		for exportName, info := range exports {
			if exportName == "default" {
				continue
			}
			rewritten = strings.ReplaceAll(rewritten, namespace+"."+exportName, info.localName)
		}
	}
	return rewritten
}
