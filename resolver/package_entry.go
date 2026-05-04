package resolver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type packageJSON struct {
	Jayess string `json:"jayess"`
	Module string `json:"module"`
	Main   string `json:"main"`
}

func ResolvePackageEntry(packageDir string, importPath string) (string, error) {
	if packageDir == "" {
		return "", fmt.Errorf("package %q has no package directory", importPath)
	}
	if resolved, err := resolvePackageJSONEntry(packageDir, importPath); err == nil {
		return resolved, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if resolved, err := resolveSourceFile(filepath.Join(packageDir, "index.js")); err == nil {
		return resolved, nil
	}
	return "", fmt.Errorf("package %q does not expose a supported Jayess .js entrypoint via jayess/module/main or index.js", importPath)
}

func resolvePackageJSONEntry(packageDir string, importPath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(packageDir, "package.json"))
	if err != nil {
		return "", err
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", fmt.Errorf("package %q has an invalid package.json: %w", importPath, err)
	}
	var firstEntryErr error
	for _, entry := range []string{pkg.Jayess, pkg.Module, pkg.Main} {
		trimmedEntry := strings.TrimSpace(entry)
		if entry != "" && trimmedEntry == "" {
			return "", fmt.Errorf("package %q entry %q is not a safe package-relative path", importPath, filepath.ToSlash(entry))
		}
		if trimmedEntry == "" {
			continue
		}
		normalizedEntry, err := normalizePackageEntryPath(trimmedEntry)
		if err != nil {
			return "", fmt.Errorf("package %q entry %q is not a safe package-relative path", importPath, filepath.ToSlash(trimmedEntry))
		}
		if filepath.Ext(normalizedEntry) != "" && strings.ToLower(filepath.Ext(normalizedEntry)) != ".js" {
			return "", fmt.Errorf("package %q entry %q is not a supported Jayess .js module", importPath, filepath.ToSlash(trimmedEntry))
		}
		resolved, err := resolveSourceFile(filepath.Join(packageDir, filepath.FromSlash(normalizedEntry)))
		if err == nil {
			return resolved, nil
		}
		if firstEntryErr == nil {
			firstEntryErr = fmt.Errorf("package %q entry %q could not be resolved: %w", importPath, filepath.ToSlash(trimmedEntry), err)
		}
	}
	if firstEntryErr != nil {
		return "", firstEntryErr
	}
	return "", os.ErrNotExist
}

func normalizePackageEntryPath(entry string) (string, error) {
	if filepath.IsAbs(entry) || isWindowsAbsolutePath(entry) {
		return "", fmt.Errorf("absolute package entry")
	}
	if strings.Contains(entry, ":") {
		return "", fmt.Errorf("scheme-like package entry")
	}
	if strings.ContainsAny(entry, "?#") {
		return "", fmt.Errorf("package entry with query string or fragment")
	}
	if strings.Contains(entry, "\\") {
		return "", fmt.Errorf("backslash package entry")
	}
	normalized := strings.TrimPrefix(filepath.ToSlash(entry), "./")
	for _, segment := range strings.Split(normalized, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("invalid package entry segment")
		}
	}
	return normalized, nil
}

func isWindowsAbsolutePath(path string) bool {
	if len(path) < 3 || path[1] != ':' {
		return false
	}
	drive := path[0]
	if !(drive >= 'A' && drive <= 'Z' || drive >= 'a' && drive <= 'z') {
		return false
	}
	return path[2] == '/' || path[2] == '\\'
}

func resolveSourceFile(path string) (string, error) {
	candidates := []string{path}
	if filepath.Ext(path) == "" {
		candidates = append(candidates, path+".js", filepath.Join(path, "index.js"))
	} else if strings.ToLower(filepath.Ext(path)) != ".js" {
		return "", fmt.Errorf("source file %q is not a supported Jayess .js module", filepath.ToSlash(path))
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
