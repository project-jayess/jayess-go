package resolver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolvePackageImport(startDir string, importPath string) (string, error) {
	if startDir == "" {
		return "", fmt.Errorf("package %q has no import start directory", importPath)
	}
	if err := validatePackageImportPath(importPath); err != nil {
		return "", err
	}
	packageName, subpath := splitPackageImport(importPath)
	for dir := startDir; ; dir = filepath.Dir(dir) {
		candidateBase := filepath.Join(dir, "node_modules", filepath.FromSlash(packageName))
		if info, err := os.Stat(candidateBase); err == nil && info.IsDir() {
			return resolvePackageTarget(candidateBase, subpath, importPath)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("package %q was not found in node_modules; run npm install or check package.json dependencies", importPath)
}

func resolvePackageTarget(packageDir string, subpath string, importPath string) (string, error) {
	if subpath != "" {
		return resolveSourceFile(filepath.Join(packageDir, filepath.FromSlash(subpath)))
	}
	return ResolvePackageEntry(packageDir, importPath)
}

func validatePackageImportPath(importPath string) error {
	if importPath == "" {
		return fmt.Errorf("package import must not be empty")
	}
	if strings.TrimSpace(importPath) != importPath {
		return fmt.Errorf("package import %q is malformed", importPath)
	}
	if filepath.IsAbs(importPath) || isWindowsAbsolutePath(importPath) {
		return fmt.Errorf("package import %q must be a bare package specifier", importPath)
	}
	if strings.Contains(importPath, ":") {
		return fmt.Errorf("package import %q must not be scheme-like", importPath)
	}
	if strings.ContainsAny(importPath, "?#") {
		return fmt.Errorf("package import %q must not include query strings or fragments", importPath)
	}
	if strings.Contains(importPath, "\\") {
		return fmt.Errorf("package import %q must use / as the path separator", importPath)
	}
	parts := strings.Split(importPath, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("package import %q is malformed", importPath)
		}
	}
	if strings.HasPrefix(importPath, "@") {
		if len(parts) < 2 || parts[0] == "@" {
			return fmt.Errorf("package import %q is malformed", importPath)
		}
	}
	if strings.HasPrefix(importPath, ".") {
		return fmt.Errorf("package import %q is malformed", importPath)
	}
	return nil
}

func splitPackageImport(importPath string) (string, string) {
	if strings.HasPrefix(importPath, "@") {
		parts := strings.Split(importPath, "/")
		if len(parts) <= 2 {
			return importPath, ""
		}
		return strings.Join(parts[:2], "/"), strings.Join(parts[2:], "/")
	}
	parts := strings.Split(importPath, "/")
	if len(parts) == 1 {
		return importPath, ""
	}
	return parts[0], strings.Join(parts[1:], "/")
}
