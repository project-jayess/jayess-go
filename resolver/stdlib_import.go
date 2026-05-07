package resolver

import "fmt"

const stdlibImportPrefix = "jayess:stdlib/"

var stdlibImportPaths = map[string]struct{}{
	"buffer":        {},
	"child_process": {},
	"compression":   {},
	"crypto":        {},
	"dns":           {},
	"ffi":           {},
	"fs":            {},
	"http":          {},
	"https":         {},
	"llvm":          {},
	"os":            {},
	"path":          {},
	"process":       {},
	"storage":       {},
	"stream":        {},
	"tcp":           {},
	"terminal":      {},
	"tls":           {},
	"udp":           {},
	"url":           {},
	"util":          {},
	"worker":        {},
}

func IsStdlibImportPath(importPath string) bool {
	_, ok := stdlibImportPaths[importPath]
	return ok
}

func ResolveStdlibImport(importPath string) (string, error) {
	if !IsStdlibImportPath(importPath) {
		return "", fmt.Errorf("stdlib import %q is not provided by Jayess", importPath)
	}
	return stdlibImportPrefix + importPath, nil
}

func IsResolvedStdlibImport(path string) bool {
	return len(path) >= len(stdlibImportPrefix) && path[:len(stdlibImportPrefix)] == stdlibImportPrefix
}
