package resolver

import (
	"fmt"
	"path/filepath"
	"strings"
)

func ResolveSourceImport(fromPath string, importPath string) (string, error) {
	if fromPath == "" {
		return "", fmt.Errorf("source import %q has no importer path", importPath)
	}
	if importPath == "" {
		return "", fmt.Errorf("source import must not be empty")
	}
	if strings.TrimSpace(importPath) != importPath {
		return "", fmt.Errorf("source import %q is malformed", importPath)
	}
	if filepath.IsAbs(importPath) || isWindowsAbsolutePath(importPath) {
		return "", fmt.Errorf("source import %q must be relative", importPath)
	}
	if strings.Contains(importPath, ":") {
		return "", fmt.Errorf("source import %q must not be scheme-like", importPath)
	}
	if strings.ContainsAny(importPath, "?#") {
		return "", fmt.Errorf("source import %q must not include query strings or fragments", importPath)
	}
	if strings.Contains(importPath, "\\") {
		return "", fmt.Errorf("source import %q must use / as the path separator", importPath)
	}
	if !strings.HasPrefix(importPath, ".") {
		return "", fmt.Errorf("source import %q must be relative", importPath)
	}
	if !isRelativeSourceImportPath(importPath) {
		return "", fmt.Errorf("source import %q must start with ./ or ../", importPath)
	}
	if err := validateRelativeSourceImportSegments(importPath); err != nil {
		return "", err
	}
	return resolveSourceFile(filepath.Join(filepath.Dir(fromPath), filepath.FromSlash(importPath)))
}

func isRelativeSourceImportPath(importPath string) bool {
	return strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../")
}

func validateRelativeSourceImportSegments(importPath string) error {
	for index, segment := range strings.Split(importPath, "/") {
		if segment == "" || segment == "." && index != 0 {
			return fmt.Errorf("source import %q is malformed", importPath)
		}
	}
	return nil
}
