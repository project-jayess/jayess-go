package main

import (
	"os"
	"path/filepath"
)

func toolchainSearchDirs(targetName string) []string {
	var dirs []string
	if root := os.Getenv("JAYESS_TOOLCHAIN"); root != "" {
		dirs = append(dirs,
			filepath.Join(root, targetName, "bin"),
			filepath.Join(root, targetName),
			filepath.Join(root, "bin"),
			root,
		)
	}
	dirs = append(dirs, executableToolchainDirs(targetName)...)
	dirs = append(dirs,
		filepath.Join("tools", targetName, "bin"),
		filepath.Join("tools", targetName),
		filepath.Join("tools", "bin"),
		"tools",
		filepath.Join("refs", "llvm", "bin"),
		filepath.Join("refs", "llvm", "build", "bin"),
		filepath.Join("refs", "llvm-project", "build", "bin"),
		filepath.Join("refs", "llvm-project", "bin"),
		filepath.Join("refs", "llvm-project", "llvm", "build", "bin"),
	)
	return dedupeStrings(dirs)
}

func executableToolchainDirs(targetName string) []string {
	executable, err := os.Executable()
	if err != nil {
		return nil
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	base := filepath.Dir(executable)
	return []string{
		filepath.Join(base, "tools", "bin"),
		filepath.Join(base, "tools"),
		filepath.Join(base, "tools", targetName, "bin"),
		filepath.Join(base, "tools", targetName),
	}
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
