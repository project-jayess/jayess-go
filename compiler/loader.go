package compiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
}

type loadedSourceTree struct {
	Source        string
	NativeImports []string
	NativeSymbols []*ast.ExternFunctionDecl
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

func loadSourceTree(entryPath string) (*loadedSourceTree, error) {
	modules := map[string]*loadedModule{}
	active := map[string]bool{}
	var parts []string
	nativeSet := map[string]bool{}
	var nativeImports []string
	var nativeSymbols []*ast.ExternFunctionDecl

	absEntry, err := filepath.Abs(entryPath)
	if err != nil {
		return nil, fmt.Errorf("resolve entry path: %w", err)
	}

	if _, err := loadSourceFile(absEntry, modules, active, &parts, nativeSet, &nativeImports, &nativeSymbols, nil); err != nil {
		return nil, err
	}

	return &loadedSourceTree{Source: strings.Join(parts, "\n\n"), NativeImports: nativeImports, NativeSymbols: nativeSymbols}, nil
}

func loadSourceFile(path string, modules map[string]*loadedModule, active map[string]bool, parts *[]string, nativeSet map[string]bool, nativeImports *[]string, nativeSymbols *[]*ast.ExternFunctionDecl, stack []string) (*loadedModule, error) {
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
			nativePath, err := resolveNativeImportPath(path, matches[1])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if !nativeSet[nativePath] {
				nativeSet[nativePath] = true
				*nativeImports = append(*nativeImports, nativePath)
			}

		case bareImportLinePattern.MatchString(line):
			matches := bareImportLinePattern.FindStringSubmatch(line)
			if isNativeImportSpec(matches[1]) {
				nativePath, err := resolveNativeImportPath(path, matches[1])
				if err != nil {
					return nil, wrapLoaderImportError(path, lineNumber, column, err)
				}
				if !nativeSet[nativePath] {
					nativeSet[nativePath] = true
					*nativeImports = append(*nativeImports, nativePath)
				}
				continue
			}
			importedPath, err := resolveImportPath(path, matches[1])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if active[importedPath] {
				return nil, loaderErrorWithNotes(path, lineNumber, column, "import cycle detected", []string{formatImportCycle(append(stack, path), importedPath)})
			}
			if _, err := loadSourceFile(importedPath, modules, active, parts, nativeSet, nativeImports, nativeSymbols, append(stack, path)); err != nil {
				return nil, err
			}

		case defaultAndNamedImportLinePattern.MatchString(line):
			matches := defaultAndNamedImportLinePattern.FindStringSubmatch(line)
			if isNativeImportSpec(matches[3]) {
				return nil, loaderError(path, lineNumber, column, "default imports from native sources are not supported")
			}
			importedPath, err := resolveImportPath(path, matches[3])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if active[importedPath] {
				return nil, loaderErrorWithNotes(path, lineNumber, column, "import cycle detected", []string{formatImportCycle(append(stack, path), importedPath)})
			}
			importedModule, err := loadSourceFile(importedPath, modules, active, parts, nativeSet, nativeImports, nativeSymbols, append(stack, path))
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
			if isNativeImportSpec(matches[2]) {
				return nil, loaderError(path, lineNumber, column, "default imports from native sources are not supported")
			}
			importedPath, err := resolveImportPath(path, matches[2])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if active[importedPath] {
				return nil, loaderErrorWithNotes(path, lineNumber, column, "import cycle detected", []string{formatImportCycle(append(stack, path), importedPath)})
			}
			importedModule, err := loadSourceFile(importedPath, modules, active, parts, nativeSet, nativeImports, nativeSymbols, append(stack, path))
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
			if isNativeImportSpec(matches[2]) {
				return nil, loaderError(path, lineNumber, column, "namespace imports from native sources are not supported")
			}
			importedPath, err := resolveImportPath(path, matches[2])
			if err != nil {
				return nil, wrapLoaderImportError(path, lineNumber, column, err)
			}
			if active[importedPath] {
				return nil, loaderErrorWithNotes(path, lineNumber, column, "import cycle detected", []string{formatImportCycle(append(stack, path), importedPath)})
			}
			importedModule, err := loadSourceFile(importedPath, modules, active, parts, nativeSet, nativeImports, nativeSymbols, append(stack, path))
			if err != nil {
				return nil, err
			}
			if err := registerImportedLocal(importedLocals, matches[1], matches[2]); err != nil {
				return nil, loaderError(path, lineNumber, column, err.Error())
			}
			namespaceImports[matches[1]] = exportedBindings(importedModule)

		case namedImportLinePattern.MatchString(line):
			matches := namedImportLinePattern.FindStringSubmatch(line)
			if isNativeImportSpec(matches[2]) {
				nativePath, err := resolveNativeImportPath(path, matches[2])
				if err != nil {
					return nil, wrapLoaderImportError(path, lineNumber, column, err)
				}
				if !nativeSet[nativePath] {
					nativeSet[nativePath] = true
					*nativeImports = append(*nativeImports, nativePath)
				}
				for _, spec := range parseImportedNames(matches[1]) {
					if err := registerImportedLocal(importedLocals, spec.local, matches[2]); err != nil {
						return nil, loaderError(path, lineNumber, column, err.Error())
					}
					*nativeSymbols = append(*nativeSymbols, &ast.ExternFunctionDecl{
						Name:         spec.local,
						NativeSymbol: spec.exported,
						Variadic:     true,
					})
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
			importedModule, err := loadSourceFile(importedPath, modules, active, parts, nativeSet, nativeImports, nativeSymbols, append(stack, path))
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
			importedModule, err := loadSourceFile(importedPath, modules, active, parts, nativeSet, nativeImports, nativeSymbols, append(stack, path))
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
			importedModule, err := loadSourceFile(importedPath, modules, active, parts, nativeSet, nativeImports, nativeSymbols, append(stack, path))
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
			importedModule, err := loadSourceFile(importedPath, modules, active, parts, nativeSet, nativeImports, nativeSymbols, append(stack, path))
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

func resolveNativeImportPath(fromPath, importPath string) (string, error) {
	if !strings.HasPrefix(importPath, ".") {
		return "", fmt.Errorf("native import %q must use a relative path", importPath)
	}
	resolved := filepath.Join(filepath.Dir(fromPath), filepath.FromSlash(importPath))
	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve native import %q: %w", importPath, err)
	}
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("native source %q was not found", filepath.ToSlash(importPath))
	}
	switch strings.ToLower(filepath.Ext(absPath)) {
	case ".c", ".cc", ".cpp", ".cxx":
		return absPath, nil
	default:
		return "", fmt.Errorf("native import %q must point to a .c/.cc/.cpp/.cxx file", filepath.ToSlash(importPath))
	}
}

func isNativeImportSpec(importPath string) bool {
	switch strings.ToLower(filepath.Ext(importPath)) {
	case ".c", ".cc", ".cpp", ".cxx":
		return true
	default:
		return false
	}
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
