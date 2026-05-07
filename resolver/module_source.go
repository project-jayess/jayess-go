package resolver

import (
	"fmt"
	"os"
)

func loadResolvedModuleSource(path string) ([]byte, string, error) {
	sourcePath, err := resolvedModuleSourcePath(path)
	if err != nil {
		return nil, "", err
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, sourcePath, err
	}
	return source, sourcePath, nil
}

func resolvedModuleSourcePath(path string) (string, error) {
	if sourcePath, ok, err := ResolvedStdlibSourcePath(path); err != nil {
		return "", err
	} else if ok {
		return sourcePath, nil
	}
	if IsResolvedStdlibImport(path) {
		return "", fmt.Errorf("stdlib module %q has no source entry", path)
	}
	return path, nil
}
