package runtime

import (
	"fmt"
	"path/filepath"
	"strings"
)

func NormalizeSourcePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("source path must not be empty")
	}
	if strings.Contains(path, "\x00") {
		return "", fmt.Errorf("source path must not contain NUL bytes")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

func ResolveSourcePath(importerPath string, importPath string) (string, error) {
	if strings.TrimSpace(importerPath) == "" {
		return "", fmt.Errorf("importer path must not be empty")
	}
	if strings.TrimSpace(importPath) == "" {
		return "", fmt.Errorf("import path must not be empty")
	}
	if filepath.IsAbs(importPath) {
		return NormalizeSourcePath(importPath)
	}
	return NormalizeSourcePath(filepath.Join(filepath.Dir(importerPath), filepath.FromSlash(importPath)))
}
