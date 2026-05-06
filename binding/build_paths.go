package binding

import "path/filepath"

func appendUniqueLDFlag(plan *BuildPlan, seen map[string]struct{}, flag string) {
	if _, exists := seen[flag]; exists {
		return
	}
	seen[flag] = struct{}{}
	plan.LDFlags = append(plan.LDFlags, flag)
}

func includeRuntimeHeader(includeDirs []string, runtimeHeaderDir string) []string {
	merged := append([]string{}, includeDirs...)
	if runtimeHeaderDir == "" {
		return merged
	}
	for _, dir := range merged {
		if dir == runtimeHeaderDir {
			return merged
		}
	}
	return append(merged, runtimeHeaderDir)
}

func resolveBindingPaths(modulePath string, paths []string) []string {
	resolved := make([]string, 0, len(paths))
	for _, path := range paths {
		resolved = append(resolved, normalizeSourceKey(modulePath, path))
	}
	return resolved
}

func normalizeSourceKey(modulePath string, source string) string {
	if filepath.IsAbs(source) {
		return filepath.Clean(source)
	}
	base := filepath.Dir(modulePath)
	return filepath.Clean(filepath.Join(base, filepath.FromSlash(source)))
}
