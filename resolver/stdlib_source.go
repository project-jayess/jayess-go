package resolver

import (
	"fmt"
	"path/filepath"
	"runtime"
)

var stdlibSourceEntryPaths = map[string]string{
	"@jayess/webview": filepath.Join("stdlib", "@jayess", "webview", "index.js"),
}

func ResolvedStdlibSourcePath(path string) (string, bool, error) {
	importPath, ok := resolvedStdlibImportPath(path)
	if !ok {
		return "", false, nil
	}
	entry, ok := stdlibSourceEntryPaths[importPath]
	if !ok {
		return "", false, nil
	}
	resolved, err := repoRootJoinedPath(entry)
	if err != nil {
		return "", false, fmt.Errorf("resolve stdlib source %q: %w", importPath, err)
	}
	return resolved, true, nil
}

func resolvedStdlibImportPath(path string) (string, bool) {
	if !IsResolvedStdlibImport(path) {
		return "", false
	}
	return path[len(stdlibImportPrefix):], true
}

func repoRootJoinedPath(rel string) (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve repo root: runtime.Caller failed")
	}
	root := filepath.Dir(filepath.Dir(file))
	joined := filepath.Join(root, rel)
	resolved, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}
