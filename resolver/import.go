package resolver

import (
	"fmt"
	"path/filepath"
	"strings"
)

func ResolveImport(fromPath string, importPath string) (string, error) {
	if fromPath == "" {
		return "", fmt.Errorf("import %q has no importer path", importPath)
	}
	if isDotPrefixedImportPath(importPath) {
		return ResolveSourceImport(fromPath, importPath)
	}
	if IsStdlibImportPath(importPath) {
		return ResolveStdlibImport(importPath)
	}
	return ResolvePackageImport(filepath.Dir(fromPath), importPath)
}

func isDotPrefixedImportPath(importPath string) bool {
	return strings.HasPrefix(importPath, ".")
}
